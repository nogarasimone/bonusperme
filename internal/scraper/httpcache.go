package scraper

import (
	"bonusperme/internal/config"
	"bonusperme/internal/logger"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// CacheEntry stores a cached HTTP response.
type CacheEntry struct {
	Body         []byte
	ETag         string
	LastModified string
	FetchedAt    time.Time
	StatusCode   int
}

// CacheStats reports HTTP cache performance.
type CacheStats struct {
	Hits       int `json:"hits"`
	Misses     int `json:"misses"`
	NotChanged int `json:"not_changed"`
	BytesSaved int `json:"bytes_saved"`
}

// HTTPCache is a thread-safe in-memory cache keyed by URL.
type HTTPCache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
	stats   CacheStats
}

var globalHTTPCache = &HTTPCache{
	entries: make(map[string]*CacheEntry),
}

// GetHTTPCache returns the global HTTP cache instance.
func GetHTTPCache() *HTTPCache {
	return globalHTTPCache
}

// GetCacheStats returns a copy of current cache statistics.
func (c *HTTPCache) GetCacheStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

// FetchWithCache fetches a URL using conditional requests when cache is available.
// Returns the body, whether the content changed, and any error.
func (c *HTTPCache) FetchWithCache(rawURL string, client *http.Client) ([]byte, bool, error) {
	body, changed, err := c.fetchWithRetry(rawURL, client, 3)
	return body, changed, err
}

func (c *HTTPCache) fetchWithRetry(rawURL string, client *http.Client, maxRetries int) ([]byte, bool, error) {
	var lastErr error
	delays := []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			if attempt-1 < len(delays) {
				time.Sleep(delays[attempt-1])
			}
		}

		body, changed, err := c.doFetch(rawURL, client)
		if err == nil {
			return body, changed, nil
		}

		lastErr = err
		if !isRetryable(err) {
			return nil, false, err
		}

		logger.Warn("httpcache: retrying", map[string]interface{}{
			"url": rawURL, "attempt": attempt + 1, "error": err.Error(),
		})
	}

	// Stale-while-error: serve cached if < 24h old
	c.mu.RLock()
	entry, ok := c.entries[rawURL]
	c.mu.RUnlock()
	if ok && time.Since(entry.FetchedAt) < 24*time.Hour {
		logger.Warn("httpcache: serving stale cache", map[string]interface{}{
			"url": rawURL, "age": time.Since(entry.FetchedAt).String(),
		})
		return entry.Body, false, nil
	}

	return nil, false, fmt.Errorf("all retries exhausted: %w", lastErr)
}

func (c *HTTPCache) doFetch(rawURL string, client *http.Client) ([]byte, bool, error) {
	// Rate limit
	rateLimiter.wait(rawURL)

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", config.Cfg.UserAgent)

	// Add conditional headers from cache
	c.mu.RLock()
	entry, cached := c.entries[rawURL]
	c.mu.RUnlock()

	if cached {
		if entry.ETag != "" {
			req.Header.Set("If-None-Match", entry.ETag)
		}
		if entry.LastModified != "" {
			req.Header.Set("If-Modified-Since", entry.LastModified)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	// 304 Not Modified — return cached body
	if resp.StatusCode == http.StatusNotModified && cached {
		c.mu.Lock()
		c.stats.Hits++
		c.stats.NotChanged++
		c.stats.BytesSaved += len(entry.Body)
		c.mu.Unlock()
		return entry.Body, false, nil
	}

	// Retryable status codes
	if resp.StatusCode == 429 || (resp.StatusCode >= 500 && resp.StatusCode <= 504) {
		if resp.StatusCode == 429 {
			rateLimiter.backoff(rawURL)
		}
		return nil, false, &retryableError{StatusCode: resp.StatusCode}
	}

	if resp.StatusCode != 200 {
		return nil, false, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, false, err
	}

	// Update cache
	newEntry := &CacheEntry{
		Body:         body,
		ETag:         resp.Header.Get("ETag"),
		LastModified: resp.Header.Get("Last-Modified"),
		FetchedAt:    time.Now(),
		StatusCode:   resp.StatusCode,
	}

	c.mu.Lock()
	if cached {
		c.stats.Hits++
	} else {
		c.stats.Misses++
	}
	c.entries[rawURL] = newEntry
	c.mu.Unlock()

	return body, true, nil
}

// retryableError indicates an HTTP error that can be retried.
type retryableError struct {
	StatusCode int
}

func (e *retryableError) Error() string {
	return fmt.Sprintf("retryable HTTP status %d", e.StatusCode)
}

func isRetryable(err error) bool {
	if _, ok := err.(*retryableError); ok {
		return true
	}
	// Also retry on network errors
	errStr := err.Error()
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "EOF")
}

// --- Per-domain rate limiter ---

type domainLimiter struct {
	mu       sync.Mutex
	lastCall map[string]time.Time
	backoffs map[string]time.Time
}

var rateLimiter = &domainLimiter{
	lastCall: make(map[string]time.Time),
	backoffs: make(map[string]time.Time),
}

func extractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func domainDelay(domain string) time.Duration {
	institutional := []string{"inps.it", "agenziaentrate.gov.it", "mef.gov.it",
		"gazzettaufficiale.it", "normattiva.it", "regione.", "provincia."}
	for _, inst := range institutional {
		if strings.Contains(domain, inst) {
			return 5 * time.Second
		}
	}
	return 1500 * time.Millisecond
}

func (d *domainLimiter) wait(rawURL string) {
	domain := extractDomain(rawURL)
	if domain == "" {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Check backoff
	if backoffUntil, ok := d.backoffs[domain]; ok && time.Now().Before(backoffUntil) {
		wait := time.Until(backoffUntil)
		d.mu.Unlock()
		time.Sleep(wait)
		d.mu.Lock()
	}

	delay := domainDelay(domain)
	if last, ok := d.lastCall[domain]; ok {
		elapsed := time.Since(last)
		if elapsed < delay {
			d.mu.Unlock()
			time.Sleep(delay - elapsed)
			d.mu.Lock()
		}
	}

	d.lastCall[domain] = time.Now()
}

func (d *domainLimiter) backoff(rawURL string) {
	domain := extractDomain(rawURL)
	if domain == "" {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.backoffs[domain] = time.Now().Add(60 * time.Second)
}
