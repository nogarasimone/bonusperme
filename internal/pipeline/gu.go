package pipeline

import (
	"bonusperme/internal/config"
	"bonusperme/internal/logger"
	"bonusperme/internal/models"
	"encoding/xml"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	guRSSURL     = "https://www.gazzettaufficiale.it/rss/SG"
	maxSeenItems = 1000
)

// GUWatcher monitors the Gazzetta Ufficiale RSS for legislative changes.
type GUWatcher struct {
	client    *http.Client
	bonusRefs map[string][]string // bonusID → normalized norm refs
	seen      map[string]bool
	mu        sync.Mutex
}

// NewGUWatcher creates a watcher that matches GU items against the given bonuses.
func NewGUWatcher(bonuses []models.Bonus) *GUWatcher {
	refs := make(map[string][]string)
	for _, b := range bonuses {
		if len(b.RiferimentiNormativi) == 0 {
			continue
		}
		var normalized []string
		for _, raw := range b.RiferimentiNormativi {
			n := NormalizeNormRef(raw)
			if n != "" {
				normalized = append(normalized, n)
			}
		}
		if len(normalized) > 0 {
			refs[b.ID] = normalized
		}
	}

	return &GUWatcher{
		client: &http.Client{Timeout: 30 * time.Second},
		bonusRefs: refs,
		seen:      make(map[string]bool),
	}
}

// Check fetches the GU RSS feed and returns events matching known bonus norm refs.
func (w *GUWatcher) Check() []GUEvent {
	items, err := w.fetchGURSS()
	if err != nil {
		logger.Error("pipeline/gu: fetch failed", map[string]interface{}{"error": err.Error()})
		return nil
	}

	var events []GUEvent
	for _, item := range items {
		// Dedup
		w.mu.Lock()
		if w.seen[item.Link] {
			w.mu.Unlock()
			continue
		}
		w.seen[item.Link] = true
		w.mu.Unlock()

		// Match against each bonus
		for bonusID, refs := range w.bonusRefs {
			matched, ref := MatchNormRef(item.Title, refs)
			if !matched {
				continue
			}

			evtType := classifyGUEvent(item.Title)
			pubDate := parseGUDate(item.PubDate)

			events = append(events, GUEvent{
				BonusID:    bonusID,
				Type:       evtType,
				NormRef:    ref,
				GUTitle:    item.Title,
				GULink:     item.Link,
				GUDate:     pubDate,
				Confidence: 1.0,
			})

			logger.Info("pipeline/gu: event detected", map[string]interface{}{
				"bonus": bonusID, "type": string(evtType), "ref": ref,
			})
		}
	}

	// Prune seen map
	w.mu.Lock()
	if len(w.seen) > maxSeenItems {
		w.seen = make(map[string]bool)
	}
	w.mu.Unlock()

	return events
}

// RSS XML structures for GU feed.
type guRSSDoc struct {
	XMLName xml.Name    `xml:"rss"`
	Channel guRSSChannel `xml:"channel"`
}

type guRSSChannel struct {
	Items []guRSSItem `xml:"item"`
}

type guRSSItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	PubDate string `xml:"pubDate"`
}

func (w *GUWatcher) fetchGURSS() ([]guRSSItem, error) {
	req, err := http.NewRequest("GET", guRSSURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", config.Cfg.UserAgent)

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &retryableHTTPError{StatusCode: resp.StatusCode}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	var doc guRSSDoc
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, err
	}

	return doc.Channel.Items, nil
}

type retryableHTTPError struct {
	StatusCode int
}

func (e *retryableHTTPError) Error() string {
	return "GU RSS returned status " + http.StatusText(e.StatusCode)
}

// Event type classification regex patterns.
var (
	prorogaRe         = regexp.MustCompile(`(?i)\b(prorog[ao]|proroga(?:to|ta)?)\b`)
	abrogazioneRe     = regexp.MustCompile(`(?i)\b(abrogaz|abrogat[oa]|soppres[so]|eliminat[oa])\b`)
	rifinanziamentoRe = regexp.MustCompile(`(?i)\b(rifinanziament|rifinanzia[to])\b`)
	conversioneRe     = regexp.MustCompile(`(?i)\b(conversion[ei]|convertit[oa])\b`)
	attuazioneRe      = regexp.MustCompile(`(?i)\b(attuazion[ei]|attuativ[oa]|regolament[oi])\b`)
)

func classifyGUEvent(title string) GUEventType {
	lower := strings.ToLower(title)

	switch {
	case abrogazioneRe.MatchString(lower):
		return GUAbrogazione
	case prorogaRe.MatchString(lower):
		return GUProroga
	case rifinanziamentoRe.MatchString(lower):
		return GURifinanziamento
	case conversioneRe.MatchString(lower):
		return GUConversione
	case attuazioneRe.MatchString(lower):
		return GUAttuazione
	default:
		return GUModifica
	}
}

func parseGUDate(s string) time.Time {
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"2006-01-02T15:04:05Z",
		"2006-01-02",
	}
	for _, fmt := range formats {
		if t, err := time.Parse(fmt, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
