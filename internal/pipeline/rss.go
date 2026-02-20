package pipeline

import (
	"bonusperme/internal/config"
	"bonusperme/internal/logger"
	"bonusperme/internal/models"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// rssPipelineFeed defines an RSS feed used by the pipeline monitor.
type rssPipelineFeed struct {
	Name     string
	URL      string
	Trust    float64
	Category FeedCategory
}

// Pipeline RSS feeds (superset of validity/news.go feeds with category tags).
var pipelineFeeds = []rssPipelineFeed{
	{Name: "FiscoOggi", URL: "https://www.fiscooggi.it/rss.xml", Trust: 1.0, Category: FeedIstituzionale},
	{Name: "GazzettaUfficiale", URL: "https://www.gazzettaufficiale.it/rss/0", Trust: 1.0, Category: FeedIstituzionale},
	{Name: "Brocardi", URL: "https://www.brocardi.it/notizie-giuridiche/feed/", Trust: 0.85, Category: FeedAutorevole},
	{Name: "InformazioneFiscale", URL: "https://www.informazionefiscale.it/rss", Trust: 0.8, Category: FeedAutorevole},
	{Name: "Sole24Ore-Norme", URL: "https://www.ilsole24ore.com/rss/norme-e-tributi.xml", Trust: 0.75, Category: FeedAutorevole},
	{Name: "Sole24Ore-Economia", URL: "https://www.ilsole24ore.com/rss/economia.xml", Trust: 0.75, Category: FeedAutorevole},
	{Name: "PMI.it", URL: "https://www.pmi.it/feed", Trust: 0.7, Category: FeedEditoriale},
	{Name: "LeggiOggi", URL: "https://www.leggioggi.it/feed/", Trust: 0.7, Category: FeedEditoriale},
	{Name: "CorriereEconomia", URL: "https://xml2.corriereobjects.it/rss/economia.xml", Trust: 0.7, Category: FeedEditoriale},
	{Name: "TheWam", URL: "https://www.thewam.net/feed/", Trust: 0.6, Category: FeedEditoriale},
	{Name: "Money.it", URL: "https://www.money.it/feed", Trust: 0.55, Category: FeedEditoriale},
	{Name: "FiscoeTasse", URL: "https://www.fiscoetasse.com/rss.xml", Trust: 0.6, Category: FeedEditoriale},
}

// Expanded bonus alias map for RSS matching.
var bonusAliases = map[string][]string{
	"assegno-unico":           {"assegno unico", "assegno universale", "assegno figli", "auu"},
	"bonus-nido":              {"bonus nido", "bonus asilo nido", "rette nido", "asilo nido contributo"},
	"bonus-nascita":           {"carta nuovi nati", "bonus nascita", "bonus neonati", "nuovi nati"},
	"bonus-mamma":             {"bonus mamme", "bonus mamma", "esonero contributi madri", "lavoratrici madri"},
	"bonus-ristrutturazione":  {"bonus ristrutturazione", "detrazione ristrutturazione", "ristrutturazione edilizia"},
	"bonus-mobili":            {"bonus mobili", "bonus elettrodomestici", "mobili ed elettrodomestici"},
	"bonus-affitto-giovani":   {"bonus affitto giovani", "bonus affitto under 31", "affitto giovani"},
	"prima-casa-under36":      {"prima casa under 36", "agevolazioni prima casa giovani", "under 36 mutuo"},
	"ecobonus":                {"ecobonus", "efficientamento energetico detrazione", "detrazione energetica"},
	"bonus-verde":             {"bonus verde", "bonus giardini", "sistemazione verde"},
	"bonus-psicologo":         {"bonus psicologo", "bonus psicoterapia", "contributo psicoterapia"},
	"carta-dedicata":          {"carta dedicata a te", "social card", "carta spesa"},
	"carta-cultura":           {"carta cultura", "carta merito", "bonus cultura", "18app"},
	"borsa-studio":            {"borsa di studio", "borse studio", "diritto allo studio"},
	"adi":                     {"assegno di inclusione", "adi reddito", "assegno inclusione"},
	"sfl":                     {"supporto formazione lavoro", "sfl indennità", "formazione lavoro"},
	"bonus-animali":           {"bonus animali", "spese veterinarie detrazione", "detrazione veterinaria"},
	"bonus-colonnine":         {"bonus colonnine", "ricarica elettrica contributo", "colonnine ricarica"},
	"bonus-acqua-potabile":    {"bonus acqua potabile", "credito acqua", "filtri acqua"},
	"bonus-decoder-tv":        {"bonus tv", "bonus decoder", "dvb-t2"},
}

// Signal keyword maps.
var (
	confermaKW = []string{"confermato", "prorogato", "rinnovo", "esteso", "rifinanziato",
		"confermata", "prorogata", "rinnovato", "estesa"}
	scadenzaKW = []string{"scaduto", "eliminato", "abolito", "non rinnovato", "soppresso",
		"terminato", "scadenza superata", "non confermato", "decaduto"}
	modificaImportoKW = []string{"nuovo importo", "importo modificato", "importo aggiornato",
		"aumento importo", "riduzione importo", "cambio importo"}
	modificaRequisitiKW = []string{"nuovi requisiti", "requisiti modificati", "soglia isee modificata",
		"cambio requisiti", "nuova soglia"}
)

// Regex for extracting amounts and ISEE from article text.
var (
	importoRe = regexp.MustCompile(`€\s*([\d.,]+)`)
	iseeRe    = regexp.MustCompile(`(?i)isee\s*(?:fino\s+a\s*|[≤<]\s*)?€?\s*([\d.,]+)`)
)

// RSSMonitor watches RSS feeds and runs quorum-based corroboration.
type RSSMonitor struct {
	feeds        []rssPipelineFeed
	bonusAliases map[string][]string
	articles     map[string][]ArticleData // bonusID → articles
	mu           sync.Mutex
	client       *http.Client
}

// NewRSSMonitor creates a new RSS monitor with the given bonuses for alias matching.
func NewRSSMonitor(bonuses []models.Bonus) *RSSMonitor {
	return &RSSMonitor{
		feeds:        pipelineFeeds,
		bonusAliases: bonusAliases,
		articles:     make(map[string][]ArticleData),
		client:       &http.Client{Timeout: 15 * time.Second},
	}
}

type rssXMLDoc struct {
	XMLName xml.Name      `xml:"rss"`
	Channel rssXMLChannel `xml:"channel"`
}

type rssXMLChannel struct {
	Items []rssXMLItem `xml:"item"`
}

type rssXMLItem struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Link        string `xml:"link"`
}

func (m *RSSMonitor) matchArticleToBonuses(title, description string) []string {
	text := strings.ToLower(title + " " + description)
	var matched []string

	for bonusID, aliases := range m.bonusAliases {
		for _, alias := range aliases {
			if strings.Contains(text, alias) {
				matched = append(matched, bonusID)
				break
			}
		}
	}
	return matched
}

func (m *RSSMonitor) corroborateBonus(bonusID string, articles []ArticleData, _ *models.Bonus) CorroborationResult {
	cfg := config.Cfg

	// Per-source dedup: one vote per feed domain
	domainSeen := make(map[string]bool)
	var confermaScore, scadenzaScore, modificaScore float64
	var sources []string
	fontiIndipendenti := 0

	cutoff := time.Now().AddDate(0, 0, -30)

	for _, art := range articles {
		if art.Date.Before(cutoff) {
			continue
		}
		if domainSeen[art.Domain] {
			continue
		}
		domainSeen[art.Domain] = true
		fontiIndipendenti++
		sources = append(sources, art.Source)

		confermaScore += art.SignalScore * art.Trust
		if art.Signal == SignalScadenza {
			scadenzaScore += art.Trust
		}
		if art.Signal == SignalModificaImporto {
			modificaScore += art.Trust
		}
	}

	result := CorroborationResult{
		BonusID: bonusID,
		Action:  ActionNone,
		Sources: sources,
	}

	// Require minimum independent sources
	if fontiIndipendenti < cfg.QuorumMinFonti {
		return result
	}

	// Evaluate thresholds in priority order
	switch {
	case scadenzaScore >= cfg.QuorumScadenza:
		result.Action = ActionMarkExpired
		result.Confidence = scadenzaScore
		result.TriggerReason = "Quorum scadenza raggiunto"

	case modificaScore >= cfg.QuorumModificaImporto:
		result.Action = ActionTriggerL2
		result.Confidence = modificaScore
		result.TriggerReason = "Modifica importo segnalata da più fonti"

	case confermaScore >= cfg.QuorumTriggerL2:
		result.Action = ActionTriggerL2
		result.Confidence = confermaScore
		result.TriggerReason = "Segnale forte — verifica normativa consigliata"

	case confermaScore >= cfg.QuorumConferma:
		result.Action = ActionConfirm
		result.Confidence = confermaScore
		result.TriggerReason = "Conferma da fonti multiple"

	case scadenzaScore >= cfg.QuorumTriggerL4:
		result.Action = ActionTriggerL4
		result.Confidence = scadenzaScore
		result.TriggerReason = "Segnale scadenza — verifica sito istituzionale"
	}

	return result
}

// fetchFeedAndClassify is a higher-level method used by RunCycle to fetch+match+classify articles.
func (m *RSSMonitor) fetchAndClassifyAll() []ArticleData {
	type feedResult struct {
		articles []rawArticle
		feed     rssPipelineFeed
	}

	ch := make(chan feedResult, len(m.feeds))
	for _, feed := range m.feeds {
		go func(f rssPipelineFeed) {
			raws := m.fetchRawFeed(f)
			ch <- feedResult{articles: raws, feed: f}
		}(feed)
	}

	var all []ArticleData
	for range m.feeds {
		r := <-ch
		domain := extractFeedDomain(r.feed.URL)
		for _, raw := range r.articles {
			text := strings.ToLower(raw.Title + " " + raw.Description)

			bonusIDs := m.matchArticleToBonuses(raw.Title, raw.Description)
			if len(bonusIDs) == 0 {
				continue
			}

			signal, signalScore := classifySignalFromText(text)
			art := ArticleData{
				Source:      r.feed.Name,
				Domain:      domain,
				Trust:       r.feed.Trust,
				Date:        raw.Date,
				BonusIDs:    bonusIDs,
				Importi:     extractImporti(raw.Title + " " + raw.Description),
				SoglieISEE:  extractISEE(raw.Title + " " + raw.Description),
				Signal:      signal,
				SignalScore: signalScore,
				Category:    r.feed.Category,
			}
			all = append(all, art)
		}
	}
	return all
}

type rawArticle struct {
	Title       string
	Description string
	Date        time.Time
}

func (m *RSSMonitor) fetchRawFeed(f rssPipelineFeed) []rawArticle {
	req, err := http.NewRequest("GET", f.URL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", config.Cfg.UserAgent)

	resp, err := m.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil
	}

	var doc rssXMLDoc
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil
	}

	var articles []rawArticle
	for _, item := range doc.Channel.Items {
		articles = append(articles, rawArticle{
			Title:       item.Title,
			Description: item.Description,
			Date:        parseRSSItemDate(item.PubDate),
		})
	}
	return articles
}

// RunFullCycle is the preferred entry point: fetches, classifies, and corroborates.
func (m *RSSMonitor) RunFullCycle(bonuses []models.Bonus) []CorroborationResult {
	articles := m.fetchAndClassifyAll()
	if len(articles) == 0 {
		logger.Info("pipeline/rss: no matching articles", nil)
		return nil
	}

	// Store articles per bonus
	m.mu.Lock()
	for _, art := range articles {
		for _, bid := range art.BonusIDs {
			m.articles[bid] = append(m.articles[bid], art)
		}
	}

	// Prune old articles
	cutoff := time.Now().AddDate(0, 0, -30)
	for bid, arts := range m.articles {
		var fresh []ArticleData
		for _, a := range arts {
			if a.Date.After(cutoff) {
				fresh = append(fresh, a)
			}
		}
		m.articles[bid] = fresh
	}
	m.mu.Unlock()

	// Corroborate per bonus
	var results []CorroborationResult
	for _, b := range bonuses {
		m.mu.Lock()
		arts := m.articles[b.ID]
		m.mu.Unlock()

		if len(arts) == 0 {
			continue
		}

		result := m.corroborateBonus(b.ID, arts, &b)
		if result.Action != ActionNone {
			results = append(results, result)
		}
	}

	logger.Info("pipeline/rss: full cycle complete", map[string]interface{}{
		"articles": len(articles),
		"results":  len(results),
	})
	return results
}

func classifySignalFromText(text string) (SignalType, float64) {
	var score float64

	for _, kw := range scadenzaKW {
		if strings.Contains(text, kw) {
			return SignalScadenza, 1.0
		}
	}
	for _, kw := range modificaImportoKW {
		if strings.Contains(text, kw) {
			return SignalModificaImporto, 1.0
		}
	}
	for _, kw := range modificaRequisitiKW {
		if strings.Contains(text, kw) {
			return SignalModificaRequisiti, 0.8
		}
	}
	for _, kw := range confermaKW {
		if strings.Contains(text, kw) {
			score += 1.0
		}
	}
	if score > 0 {
		return SignalConferma, score
	}

	return SignalConferma, 0.3 // Mention without explicit signal
}

func extractImporti(text string) []string {
	matches := importoRe.FindAllStringSubmatch(text, -1)
	var result []string
	for _, m := range matches {
		result = append(result, "€"+m[1])
	}
	return result
}

func extractISEE(text string) []float64 {
	matches := iseeRe.FindAllStringSubmatch(text, -1)
	var result []float64
	for _, m := range matches {
		cleaned := strings.ReplaceAll(m[1], ".", "")
		cleaned = strings.ReplaceAll(cleaned, ",", ".")
		if v, err := strconv.ParseFloat(cleaned, 64); err == nil {
			result = append(result, v)
		}
	}
	return result
}

func extractFeedDomain(feedURL string) string {
	u, err := url.Parse(feedURL)
	if err != nil {
		return feedURL
	}
	return u.Hostname()
}

func parseRSSItemDate(s string) time.Time {
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
