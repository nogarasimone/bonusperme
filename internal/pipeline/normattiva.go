package pipeline

import (
	"bonusperme/internal/config"
	"bonusperme/internal/logger"
	"bonusperme/internal/scraper"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const normattivaBaseURL = "https://www.normattiva.it/uri-res/N2Ls?urn:nir:stato:"

// NormattivaFetcher fetches consolidated law text from normattiva.it.
// Disabled by default (PipelineL2Enabled=false) — needs manual testing.
type NormattivaFetcher struct {
	client *http.Client
	cache  map[string]*NormText
	mu     sync.Mutex
}

// NewNormattivaFetcher creates a new fetcher instance.
func NewNormattivaFetcher() *NormattivaFetcher {
	return &NormattivaFetcher{
		client: &http.Client{Timeout: 30 * time.Second},
		cache:  make(map[string]*NormText),
	}
}

// FetchConsolidated fetches the consolidated text for a normalized norm reference.
// Uses the HTTP cache to respect rate limits (5s for normattiva.it).
func (f *NormattivaFetcher) FetchConsolidated(normRef string) (*NormText, error) {
	if !config.Cfg.PipelineL2Enabled {
		return nil, fmt.Errorf("L2 (Normattiva) is disabled")
	}

	f.mu.Lock()
	if cached, ok := f.cache[normRef]; ok {
		// Return cache if fetched within 24h
		if time.Since(cached.FetchedAt) < 24*time.Hour {
			f.mu.Unlock()
			return cached, nil
		}
	}
	f.mu.Unlock()

	searchURL := buildNormattivaURL(normRef)
	if searchURL == "" {
		return nil, fmt.Errorf("cannot build URL for norm ref: %s", normRef)
	}

	body, _, err := scraper.GetHTTPCache().FetchWithCache(searchURL, f.client)
	if err != nil {
		logger.Warn("pipeline/normattiva: fetch failed", map[string]interface{}{
			"normRef": normRef, "error": err.Error(),
		})
		return nil, err
	}

	result := parseNormattivaHTML(normRef, string(body))
	result.FetchedAt = time.Now()

	f.mu.Lock()
	f.cache[normRef] = result
	f.mu.Unlock()

	logger.Info("pipeline/normattiva: fetched", map[string]interface{}{
		"normRef":  normRef,
		"importi":  len(result.ImportiTrovati),
		"isee":     len(result.ISEETrovate),
	})

	return result, nil
}

// buildNormattivaURL constructs a normattiva.it search URL from a canonical norm ref.
// Example: "DLgs 230/2021" → "https://www.normattiva.it/uri-res/N2Ls?urn:nir:stato:decreto.legislativo:2021-12-29;230"
func buildNormattivaURL(normRef string) string {
	parts := strings.Fields(normRef)
	if len(parts) < 2 {
		return ""
	}

	tipologia := strings.ToLower(parts[0])
	numYear := parts[1]

	numParts := strings.Split(numYear, "/")
	if len(numParts) != 2 {
		return ""
	}
	number := numParts[0]
	year := numParts[1]

	var urnType string
	switch tipologia {
	case "dlgs":
		urnType = "decreto.legislativo"
	case "dl":
		urnType = "decreto.legge"
	case "l":
		urnType = "legge"
	case "dpcm":
		urnType = "decreto.del.presidente.del.consiglio.dei.ministri"
	case "dpr":
		urnType = "decreto.del.presidente.della.repubblica"
	case "dm":
		urnType = "decreto.ministeriale"
	default:
		return ""
	}

	urn := fmt.Sprintf("urn:nir:stato:%s:%s;%s", urnType, year, number)
	return normattivaBaseURL[:len(normattivaBaseURL)-len("urn:nir:stato:")] + url.PathEscape(urn)
}

// Legislative extraction regex patterns.
var (
	normImportoRe = regexp.MustCompile(`(?i)(?:euro|€)\s*([\d.,]+)`)
	normISEERe    = regexp.MustCompile(`(?i)(?:ISEE|indicatore.+situazione.+economica)\s*(?:pari\s+a|non\s+superiore\s+a|fino\s+a|inferiore\s+a)?\s*(?:euro|€)?\s*([\d.,]+)`)
	normDateRe    = regexp.MustCompile(`(\d{1,2})\s+(gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)\s+(\d{4})`)
)

var monthMap = map[string]time.Month{
	"gennaio": time.January, "febbraio": time.February, "marzo": time.March,
	"aprile": time.April, "maggio": time.May, "giugno": time.June,
	"luglio": time.July, "agosto": time.August, "settembre": time.September,
	"ottobre": time.October, "novembre": time.November, "dicembre": time.December,
}

func parseNormattivaHTML(normRef, html string) *NormText {
	result := &NormText{
		NormRef: normRef,
	}

	// Extract importi
	for _, m := range normImportoRe.FindAllStringSubmatch(html, -1) {
		cleaned := strings.ReplaceAll(m[1], ".", "")
		cleaned = strings.ReplaceAll(cleaned, ",", ".")
		result.ImportiTrovati = append(result.ImportiTrovati, ImportoNormativo{
			Valore:   "€" + m[1],
			Contesto: extractContext(html, m[0], 100),
		})
		_ = cleaned // parsed value available if needed
	}

	// Extract ISEE thresholds
	for _, m := range normISEERe.FindAllStringSubmatch(html, -1) {
		cleaned := strings.ReplaceAll(m[1], ".", "")
		cleaned = strings.ReplaceAll(cleaned, ",", ".")
		if v, err := strconv.ParseFloat(cleaned, 64); err == nil {
			result.ISEETrovate = append(result.ISEETrovate, SogliaISEE{
				Valore:   v,
				Contesto: extractContext(html, m[0], 100),
			})
		}
	}

	// Extract dates
	for _, m := range normDateRe.FindAllStringSubmatch(html, -1) {
		day, _ := strconv.Atoi(m[1])
		month := monthMap[strings.ToLower(m[2])]
		year, _ := strconv.Atoi(m[3])
		if month > 0 {
			result.DateTrovate = append(result.DateTrovate, DataNormativa{
				Data:     time.Date(year, month, day, 0, 0, 0, 0, time.UTC),
				Contesto: extractContext(html, m[0], 100),
			})
		}
	}

	return result
}

// extractContext returns surrounding text around a match for context.
func extractContext(text, match string, radius int) string {
	idx := strings.Index(text, match)
	if idx < 0 {
		return ""
	}
	start := idx - radius
	if start < 0 {
		start = 0
	}
	end := idx + len(match) + radius
	if end > len(text) {
		end = len(text)
	}
	return strings.TrimSpace(text[start:end])
}
