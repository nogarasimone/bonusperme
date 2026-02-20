package handlers

import (
	"bonusperme/internal/blog"
	"bonusperme/internal/config"
	"bonusperme/internal/matcher"
	"bonusperme/internal/scraper"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ---------- Startup time for uptime ----------

var startTime = time.Now()

// ---------- Last scrape time tracking ----------

var (
	lastScrapeTime time.Time
	lastScrapeMu   sync.RWMutex
)

// SetLastScrape records the time of the most recent data update.
func SetLastScrape(t time.Time) {
	lastScrapeMu.Lock()
	lastScrapeTime = t
	lastScrapeMu.Unlock()
}

// getLastScrape returns the last scrape time, falling back to server start time.
func getLastScrape() time.Time {
	lastScrapeMu.RLock()
	defer lastScrapeMu.RUnlock()
	if lastScrapeTime.IsZero() {
		return startTime
	}
	return lastScrapeTime
}

// StatusHandler returns the last data update time in Italian timezone.
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	t := getLastScrape()

	loc, err := time.LoadLocation("Europe/Rome")
	if err == nil {
		t = t.In(loc)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"last_update":         t.Format(time.RFC3339),
		"last_update_display": t.Format("02/01/2006 alle 15:04"),
	})
}

// ---------- Privacy-first analytics ----------

type analyticsStore struct {
	mu         sync.Mutex
	pageViews  int64
	apiCalls   int64
	matchCalls int64
	dailyViews map[string]int64 // date -> count
}

var analytics = &analyticsStore{
	dailyViews: make(map[string]int64),
}

// TrackPageView increments page view counter.
func TrackPageView() {
	atomic.AddInt64(&analytics.pageViews, 1)
	analytics.mu.Lock()
	today := time.Now().Format("2006-01-02")
	analytics.dailyViews[today]++
	analytics.mu.Unlock()
}

// TrackAPICall increments API call counter.
func TrackAPICall() {
	atomic.AddInt64(&analytics.apiCalls, 1)
}

// TrackMatchCall increments match-specific counter.
func TrackMatchCall() {
	atomic.AddInt64(&analytics.matchCalls, 1)
}

// AnalyticsHandler records a page view (POST) or returns basic stats (GET).
func AnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		TrackPageView()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{
		"page_views":  atomic.LoadInt64(&analytics.pageViews),
		"api_calls":   atomic.LoadInt64(&analytics.apiCalls),
		"match_calls": atomic.LoadInt64(&analytics.matchCalls),
	})
}

// AnalyticsSummaryHandler returns aggregated analytics.
func AnalyticsSummaryHandler(w http.ResponseWriter, r *http.Request) {
	analytics.mu.Lock()
	dailyCopy := make(map[string]int64)
	for k, v := range analytics.dailyViews {
		dailyCopy[k] = v
	}
	analytics.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"page_views":  atomic.LoadInt64(&analytics.pageViews),
		"api_calls":   atomic.LoadInt64(&analytics.apiCalls),
		"match_calls": atomic.LoadInt64(&analytics.matchCalls),
		"daily_views": dailyCopy,
		"uptime_sec":  int(time.Since(startTime).Seconds()),
	})
}

// ---------- Improved Health ----------

// HealthDetailedHandler returns detailed health info.
func HealthDetailedHandler(w http.ResponseWriter, r *http.Request) {
	scraperStatus := scraper.GetScraperStatus()
	uptime := time.Since(startTime)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "ok",
		"uptime_seconds": int(uptime.Seconds()),
		"uptime_human":   formatDuration(uptime),
		"scraper":        scraperStatus,
		"scansioni":      GetCounter(),
		"bonus_count":    len(scraper.GetCachedBonus()),
	})
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// ScraperStatusHandler returns scraper status.
func ScraperStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scraper.GetScraperStatus())
}

// ---------- SEO Pages ----------

// BonusPageHandler serves an HTML page for a single bonus (SEO-friendly).
func BonusPageHandler(w http.ResponseWriter, r *http.Request) {
	// Extract bonus ID from path: /bonus/{id}
	path := strings.TrimPrefix(r.URL.Path, "/bonus/")
	bonusID := strings.TrimSuffix(path, "/")
	if bonusID == "" {
		http.Error(w, "Bonus non trovato", http.StatusNotFound)
		return
	}

	allBonuses := scraper.GetCachedBonus()
	for _, b := range allBonuses {
		if b.ID == bonusID {
			serveBonusPage(w, b)
			return
		}
	}

	http.Error(w, "Bonus non trovato", http.StatusNotFound)
}

func serveBonusPage(w http.ResponseWriter, b interface{}) {
	type bonusLike struct {
		ID                  string
		Nome                string
		Descrizione         string
		Importo             string
		Scadenza            string
		Requisiti           []string
		ComeRichiederlo     []string
		Documenti           []string
		LinkUfficiale       string
		Ente                string
		FonteURL            string
		FonteNome           string
		RiferimentiNormativi []string
		UltimoAggiornamento string
		Stato               string
	}

	// Marshal and unmarshal to get structured access
	data, _ := json.Marshal(b)
	var bonus bonusLike
	json.Unmarshal(data, &bonus)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html lang="it">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + htmlEscape(bonus.Nome) + ` - BonusPerMe</title>
<meta name="description" content="` + htmlEscape(truncate(bonus.Descrizione, 160)) + `">
<meta property="og:title" content="` + htmlEscape(bonus.Nome) + ` - BonusPerMe">
<meta property="og:description" content="` + htmlEscape(truncate(bonus.Descrizione, 160)) + `">
<link rel="canonical" href="/bonus/` + htmlEscape(bonus.ID) + `">
<style>
body{font-family:system-ui,sans-serif;max-width:800px;margin:0 auto;padding:20px;color:#333;line-height:1.6}
h1{color:#003366;border-bottom:2px solid #0066cc;padding-bottom:10px}
.meta{color:#666;font-size:0.9em;margin-bottom:20px}
.section{margin:20px 0}
.section h2{color:#004488;font-size:1.2em}
ul{padding-left:20px}
.badge{display:inline-block;padding:3px 10px;border-radius:12px;font-size:0.85em;font-weight:600}
.badge-attivo{background:#d4edda;color:#155724}
.importo{font-size:1.3em;color:#006600;font-weight:bold}
.back{display:inline-block;margin-bottom:20px;color:#0066cc;text-decoration:none}
.back:hover{text-decoration:underline}
.fonte{background:#f8f9fa;padding:15px;border-radius:8px;margin:20px 0;font-size:0.9em}
footer{margin-top:40px;padding-top:20px;border-top:1px solid #ddd;color:#999;font-size:0.85em;text-align:center}
</style>
</head>
<body>
<a href="/" class="back">← Torna a BonusPerMe</a>
<h1>` + htmlEscape(bonus.Nome) + `</h1>
<div class="meta">`)

	if bonus.Ente != "" {
		sb.WriteString(`<strong>Ente:</strong> ` + htmlEscape(bonus.Ente))
	}
	if bonus.Stato != "" {
		sb.WriteString(` <span class="badge badge-` + htmlEscape(bonus.Stato) + `">` + htmlEscape(strings.ToUpper(bonus.Stato[:1])+bonus.Stato[1:]) + `</span>`)
	}
	if bonus.UltimoAggiornamento != "" {
		sb.WriteString(`<br><strong>Aggiornato:</strong> ` + htmlEscape(bonus.UltimoAggiornamento))
	}
	sb.WriteString(`</div>`)

	if bonus.Importo != "" {
		sb.WriteString(`<p class="importo">Importo: ` + htmlEscape(bonus.Importo) + `</p>`)
	}

	sb.WriteString(`<div class="section"><p>` + htmlEscape(bonus.Descrizione) + `</p></div>`)

	if bonus.Scadenza != "" {
		sb.WriteString(`<div class="section"><h2>Scadenza</h2><p>` + htmlEscape(bonus.Scadenza) + `</p></div>`)
	}

	if len(bonus.Requisiti) > 0 {
		sb.WriteString(`<div class="section"><h2>Requisiti</h2><ul>`)
		for _, r := range bonus.Requisiti {
			sb.WriteString(`<li>` + htmlEscape(r) + `</li>`)
		}
		sb.WriteString(`</ul></div>`)
	}

	if len(bonus.ComeRichiederlo) > 0 {
		sb.WriteString(`<div class="section"><h2>Come Richiederlo</h2><ol>`)
		for _, s := range bonus.ComeRichiederlo {
			sb.WriteString(`<li>` + htmlEscape(s) + `</li>`)
		}
		sb.WriteString(`</ol></div>`)
	}

	if len(bonus.Documenti) > 0 {
		sb.WriteString(`<div class="section"><h2>Documenti Necessari</h2><ul>`)
		for _, d := range bonus.Documenti {
			sb.WriteString(`<li>` + htmlEscape(d) + `</li>`)
		}
		sb.WriteString(`</ul></div>`)
	}

	if bonus.FonteURL != "" || bonus.FonteNome != "" || len(bonus.RiferimentiNormativi) > 0 {
		sb.WriteString(`<div class="fonte"><strong>Fonti e riferimenti</strong><br>`)
		if bonus.FonteNome != "" {
			sb.WriteString(`Fonte: ` + htmlEscape(bonus.FonteNome))
		}
		if bonus.FonteURL != "" {
			sb.WriteString(` — <a href="` + htmlEscape(bonus.FonteURL) + `" target="_blank" rel="noopener">Sito ufficiale</a>`)
		}
		if len(bonus.RiferimentiNormativi) > 0 {
			sb.WriteString(`<br>Riferimenti: ` + htmlEscape(strings.Join(bonus.RiferimentiNormativi, "; ")))
		}
		sb.WriteString(`</div>`)
	}

	if bonus.LinkUfficiale != "" {
		sb.WriteString(`<p><a href="` + htmlEscape(bonus.LinkUfficiale) + `" target="_blank" rel="noopener">Vai al sito ufficiale →</a></p>`)
	}

	sb.WriteString(`
<footer>
<p>BonusPerMe — Servizio gratuito e indipendente. Le informazioni sono a scopo orientativo.</p>
<p><a href="/">Verifica i tuoi bonus →</a></p>
</footer>
</body>
</html>`)

	w.Write([]byte(sb.String()))
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// ---------- Sitemap ----------

type xhtmlLink struct {
	XMLName xml.Name `xml:"xhtml:link"`
	Rel     string   `xml:"rel,attr"`
	Hreflang string  `xml:"hreflang,attr"`
	Href    string   `xml:"href,attr"`
}

type siteURL struct {
	Loc        string      `xml:"loc"`
	LastMod    string      `xml:"lastmod,omitempty"`
	ChangeFreq string      `xml:"changefreq,omitempty"`
	Priority   string      `xml:"priority,omitempty"`
	Links      []xhtmlLink `xml:",omitempty"`
}

type urlSet struct {
	XMLName    xml.Name  `xml:"urlset"`
	XMLNS      string    `xml:"xmlns,attr"`
	XHTMLns    string    `xml:"xmlns:xhtml,attr,omitempty"`
	URLs       []siteURL `xml:"url"`
}

var sitemapLangs = []string{"it", "en", "fr", "es", "ro", "ar", "sq"}

func hreflangLinks(baseURL, path string) []xhtmlLink {
	links := make([]xhtmlLink, 0, len(sitemapLangs)+1)
	// x-default points to Italian (no lang param)
	links = append(links, xhtmlLink{Rel: "alternate", Hreflang: "x-default", Href: baseURL + path})
	for _, lang := range sitemapLangs {
		href := baseURL + path
		if lang != "it" {
			if path == "/" {
				href += "?lang=" + lang
			} else {
				href += "?lang=" + lang
			}
		}
		links = append(links, xhtmlLink{Rel: "alternate", Hreflang: lang, Href: href})
	}
	return links
}

func SitemapHandler(w http.ResponseWriter, r *http.Request) {
	baseURL := config.Cfg.BaseURL
	today := time.Now().Format("2006-01-02")

	urls := []siteURL{
		{Loc: baseURL + "/", LastMod: today, ChangeFreq: "daily", Priority: "1.0", Links: hreflangLinks(baseURL, "/")},
		{Loc: baseURL + "/per-caf", LastMod: today, ChangeFreq: "monthly", Priority: "0.7"},
		{Loc: baseURL + "/contatti", ChangeFreq: "monthly", Priority: "0.5"},
		{Loc: baseURL + "/privacy", ChangeFreq: "yearly", Priority: "0.3"},
		{Loc: baseURL + "/cookie-policy", ChangeFreq: "yearly", Priority: "0.3"},
	}

	allBonuses := matcher.GetAllBonusWithRegional()
	for _, b := range allBonuses {
		urls = append(urls, siteURL{
			Loc:        baseURL + "/bonus/" + b.ID,
			LastMod:    today,
			ChangeFreq: "weekly",
			Priority:   "0.8",
		})
	}

	// Blog guide pages
	urls = append(urls, siteURL{
		Loc:        baseURL + "/guide",
		LastMod:    today,
		ChangeFreq: "weekly",
		Priority:   "0.7",
	})
	for _, p := range blog.GetAll() {
		urls = append(urls, siteURL{
			Loc:        baseURL + "/guide/" + p.Slug,
			LastMod:    p.Date.Format("2006-01-02"),
			ChangeFreq: "weekly",
			Priority:   "0.7",
		})
	}

	sitemap := urlSet{
		XMLNS:   "http://www.sitemaps.org/schemas/sitemap/0.9",
		XHTMLns: "http://www.w3.org/1999/xhtml",
		URLs:    urls,
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	enc.Encode(sitemap)
}

// RobotsTxtHandler serves robots.txt with sitemap link.
func RobotsTxtHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write([]byte("User-agent: *\nAllow: /\nAllow: /bonus/\nAllow: /guide\nAllow: /guide/\nAllow: /per-caf\nAllow: /contatti\nDisallow: /api/\nDisallow: /static/\nDisallow: /.env\nDisallow: /.git\n\n# Crawl-delay for polite bots\nCrawl-delay: 1\n\nSitemap: " + config.Cfg.BaseURL + "/sitemap.xml\n"))
}

// ---------- Translations ----------

// TranslationsHandler serves i18n translations.
// Will be connected to the i18n package once it's available.
func TranslationsHandler(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = "it"
	}

	// Try to load from i18n package dynamically
	translations := getTranslations(lang)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	json.NewEncoder(w).Encode(translations)
}

// getTranslations will be replaced once i18n package is ready.
// For now returns the language code so the frontend knows it was received.
var translationLoader func(string) map[string]string

// SetTranslationLoader sets the function used to load translations.
func SetTranslationLoader(loader func(string) map[string]string) {
	translationLoader = loader
}

func getTranslations(lang string) map[string]string {
	if translationLoader != nil {
		return translationLoader(lang)
	}
	return map[string]string{"_lang": lang, "_status": "translations_not_loaded"}
}
