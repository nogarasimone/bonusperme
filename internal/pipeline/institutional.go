package pipeline

import (
	"bonusperme/internal/linkcheck"
	"bonusperme/internal/logger"
	"bonusperme/internal/models"
	"bonusperme/internal/scraper"
	"crypto/sha256"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ScrapeURL defines a target URL for institutional scraping.
type ScrapeURL struct {
	URL  string `json:"url"`
	Ente string `json:"ente"`
}

// InstitutionalScraper checks institutional pages for bonus data.
type InstitutionalScraper struct {
	client  *http.Client
	targets map[string][]ScrapeURL    // bonusID → URLs
	results map[string]*ScrapeResult  // bonusID → latest result
	mu      sync.Mutex
}

// NewInstitutionalScraper creates a scraper with targets built from bonus data.
func NewInstitutionalScraper(bonuses []models.Bonus) *InstitutionalScraper {
	s := &InstitutionalScraper{
		client:  &http.Client{Timeout: 30 * time.Second},
		targets: make(map[string][]ScrapeURL),
		results: make(map[string]*ScrapeResult),
	}
	s.buildTargets(bonuses)
	return s
}

func (s *InstitutionalScraper) buildTargets(bonuses []models.Bonus) {
	for _, b := range bonuses {
		var urls []ScrapeURL

		if b.FonteURL != "" {
			urls = append(urls, ScrapeURL{URL: b.FonteURL, Ente: b.Ente})
		}
		if b.LinkUfficiale != "" && b.LinkUfficiale != b.FonteURL {
			urls = append(urls, ScrapeURL{URL: b.LinkUfficiale, Ente: b.Ente})
		}

		if len(urls) > 0 {
			s.targets[b.ID] = urls
		}
	}
}

// Scrape checks the institutional page(s) for a bonus and returns the result.
func (s *InstitutionalScraper) Scrape(bonusID string) (*ScrapeResult, error) {
	s.mu.Lock()
	urls, ok := s.targets[bonusID]
	s.mu.Unlock()

	if !ok || len(urls) == 0 {
		return nil, fmt.Errorf("no targets for bonus %s", bonusID)
	}

	// Try each URL until one succeeds
	for _, target := range urls {
		result, err := s.scrapeURL(bonusID, target)
		if err != nil {
			logger.Warn("pipeline/institutional: scrape failed", map[string]interface{}{
				"bonus": bonusID, "url": target.URL, "error": err.Error(),
			})
			continue
		}

		s.mu.Lock()
		s.results[bonusID] = result
		s.mu.Unlock()

		return result, nil
	}

	return nil, fmt.Errorf("all targets failed for bonus %s", bonusID)
}

func (s *InstitutionalScraper) scrapeURL(bonusID string, target ScrapeURL) (*ScrapeResult, error) {
	// Use HTTP cache with conditional GET
	body, _, err := scraper.GetHTTPCache().FetchWithCache(target.URL, s.client)
	if err != nil {
		return nil, err
	}

	// Verify link is reachable
	linkOK, _ := linkcheck.CheckLink(target.URL)

	// Compute page hash for content change detection
	hash := fmt.Sprintf("%x", sha256.Sum256(body))

	// Check if content changed from last scrape
	s.mu.Lock()
	prev := s.results[bonusID]
	s.mu.Unlock()

	if prev != nil && prev.PageHash == hash {
		// No change — return previous result with updated timestamp
		return &ScrapeResult{
			BonusID:   bonusID,
			LinkOK:    linkOK,
			Importo:   prev.Importo,
			Requisiti: prev.Requisiti,
			Scadenza:  prev.Scadenza,
			PageHash:  hash,
			FetchedAt: time.Now(),
		}, nil
	}

	// Extract data from page content
	content := string(body)
	importo := extractPageImporto(content)
	scadenza := extractPageScadenza(content)

	return &ScrapeResult{
		BonusID:   bonusID,
		LinkOK:    linkOK,
		Importo:   importo,
		Scadenza:  scadenza,
		PageHash:  hash,
		FetchedAt: time.Now(),
	}, nil
}

// GetResult returns the latest scrape result for a bonus.
func (s *InstitutionalScraper) GetResult(bonusID string) *ScrapeResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.results[bonusID]
}

// Page content extraction regex.
var (
	pageImportoRe  = regexp.MustCompile(`(?i)(?:importo|contributo|detrazione|credito)[^.]{0,80}(?:euro|€)\s*([\d.,]+)`)
	pageScadenzaRe = regexp.MustCompile(`(?i)(?:scadenza|entro il|termine)[^.]{0,60}(\d{1,2}\s+(?:gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)\s+\d{4})`)
)

func extractPageImporto(content string) string {
	m := pageImportoRe.FindStringSubmatch(content)
	if len(m) >= 2 {
		return "€" + strings.TrimSpace(m[1])
	}
	return ""
}

func extractPageScadenza(content string) string {
	m := pageScadenzaRe.FindStringSubmatch(content)
	if len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}
