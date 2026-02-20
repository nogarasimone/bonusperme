package handlers

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RateLimiter implements a per-IP token bucket rate limiter.
type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]*bucket
	rate    int           // tokens per interval
	burst   int           // max tokens
	interval time.Duration
}

type bucket struct {
	tokens   int
	lastTime time.Time
}

// NewRateLimiter creates a rate limiter: rate tokens per interval, with burst capacity.
func NewRateLimiter(rate int, burst int, interval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients:  make(map[string]*bucket),
		rate:     rate,
		burst:    burst,
		interval: interval,
	}
	// Cleanup stale entries every 5 minutes
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()
	return rl
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := time.Now().Add(-10 * time.Minute)
	for ip, b := range rl.clients {
		if b.lastTime.Before(cutoff) {
			delete(rl.clients, ip)
		}
	}
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, exists := rl.clients[ip]
	now := time.Now()

	if !exists {
		rl.clients[ip] = &bucket{tokens: rl.burst - 1, lastTime: now}
		return true
	}

	// Refill tokens
	elapsed := now.Sub(b.lastTime)
	refill := int(elapsed / rl.interval) * rl.rate
	b.tokens += refill
	if b.tokens > rl.burst {
		b.tokens = rl.burst
	}
	b.lastTime = now

	if b.tokens > 0 {
		b.tokens--
		return true
	}
	return false
}

// Middleware wraps an http.Handler with rate limiting.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		// Strip port from RemoteAddr
		if host, _, err := net.SplitHostPort(ip); err == nil {
			ip = host
		}
		// Use first IP from X-Forwarded-For if behind proxy
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			ip = strings.TrimSpace(strings.SplitN(fwd, ",", 2)[0])
		}

		if !rl.allow(ip) {
			http.Error(w, "Troppe richieste. Riprova tra poco.", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
