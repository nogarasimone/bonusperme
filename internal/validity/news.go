package validity

import (
	"bonusperme/internal/config"
	"bonusperme/internal/logger"
	"bonusperme/internal/models"
	"encoding/xml"
	"io"
	"net/http"
	"strings"
	"time"
)

// RSS feed sources for bonus news monitoring.
var rssFeeds = []rssFeed{
	{Name: "FiscoOggi", URL: "https://www.fiscooggi.it/rss.xml", Trust: 1.0},
	{Name: "InformazioneFiscale", URL: "https://www.informazionefiscale.it/rss", Trust: 0.8},
	{Name: "PMI.it", URL: "https://www.pmi.it/feed", Trust: 0.7},
	{Name: "LeggiOggi", URL: "https://www.leggioggi.it/feed/", Trust: 0.7},
	{Name: "TheWam", URL: "https://www.thewam.net/feed/", Trust: 0.6},
	{Name: "GazzettaUfficiale", URL: "https://www.gazzettaufficiale.it/rss/0", Trust: 1.0},
	{Name: "Brocardi", URL: "https://www.brocardi.it/notizie-giuridiche/feed/", Trust: 0.85},
	{Name: "Corriere Economia", URL: "https://xml2.corriereobjects.it/rss/economia.xml", Trust: 0.7},
	{Name: "Sole 24 Ore - Norme e Tributi", URL: "https://www.ilsole24ore.com/rss/norme-e-tributi.xml", Trust: 0.75},
	{Name: "Sole 24 Ore - Economia", URL: "https://www.ilsole24ore.com/rss/economia.xml", Trust: 0.75},
}

type rssFeed struct {
	Name  string
	URL   string
	Trust float64
}

// Per-bonus keywords for matching RSS articles.
var bonusKeywords = map[string][]string{
	"assegno-unico":         {"assegno unico", "assegno universale", "assegno figli"},
	"bonus-nido":            {"bonus nido", "bonus asilo nido", "rette nido"},
	"bonus-nascita":         {"carta nuovi nati", "bonus nascita", "bonus neonati"},
	"bonus-mamma":           {"bonus mamme", "bonus mamma lavoratrice", "esonero contributi madri"},
	"bonus-ristrutturazione": {"bonus ristrutturazione", "detrazione ristrutturazione", "ristrutturazione edilizia"},
	"bonus-mobili":          {"bonus mobili", "bonus elettrodomestici"},
	"bonus-affitto-giovani": {"bonus affitto giovani", "bonus affitto under 31"},
	"prima-casa-under36":    {"prima casa under 36", "agevolazioni prima casa giovani"},
	"ecobonus":              {"ecobonus", "efficientamento energetico detrazione"},
	"bonus-verde":           {"bonus verde", "bonus giardini"},
	"bonus-psicologo":       {"bonus psicologo", "bonus psicoterapia"},
	"carta-dedicata":        {"carta dedicata a te", "social card"},
	"carta-cultura":         {"carta cultura", "carta merito", "bonus cultura"},
	"borsa-studio":          {"borsa di studio", "borse studio universitarie"},
	"adi":                   {"assegno di inclusione", "ADI reddito"},
	"sfl":                   {"supporto formazione lavoro", "SFL indennità"},
	"bonus-animali":         {"bonus animali", "spese veterinarie detrazione"},
	"bonus-colonnine":       {"bonus colonnine", "ricarica elettrica contributo"},
	"bonus-acqua-potabile":  {"bonus acqua potabile", "credito acqua"},
}

// Signal keywords — positive = conferma, negative = scadenza.
var confermaKeywords = []string{"confermato", "prorogato", "rinnovo", "esteso", "rifinanziato", "confermata", "prorogata"}
var scadenzaKeywords = []string{"scaduto", "eliminato", "abolito", "non rinnovato", "soppresso", "terminato", "scadenza superata"}

// RSS XML structures (independent from gu_rss.go).
type rssDocument struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Link        string `xml:"link"`
}

// RunNewsCheck fetches RSS feeds and cross-checks against bonus keywords.
func RunNewsCheck(bonuses []models.Bonus) {
	articles := fetchAllFeeds()
	if len(articles) == 0 {
		logger.Info("news check: no articles fetched", nil)
		return
	}

	cutoff := time.Now().AddDate(0, 0, -30)
	matched := 0

	for _, b := range bonuses {
		keywords, ok := bonusKeywords[b.ID]
		if !ok {
			continue
		}

		var confermaScore, scadenzaScore float64

		for _, art := range articles {
			if art.Date.Before(cutoff) {
				continue
			}

			text := strings.ToLower(art.Title + " " + art.Description)
			bonusMatch := false
			for _, kw := range keywords {
				if strings.Contains(text, kw) {
					bonusMatch = true
					break
				}
			}
			if !bonusMatch {
				continue
			}

			// Score signals
			for _, kw := range confermaKeywords {
				if strings.Contains(text, kw) {
					confermaScore += art.Trust
				}
			}
			for _, kw := range scadenzaKeywords {
				if strings.Contains(text, kw) {
					scadenzaScore += art.Trust
				}
			}
		}

		if scadenzaScore >= 1.5 {
			SetStatus(b.ID, "potenzialmente_scaduto", "Segnalazione da fonti RSS")
			AddAlert(Alert{
				BonusID:   b.ID,
				BonusNome: b.Nome,
				OldStato:  b.StatoValidita,
				NewStato:  "potenzialmente_scaduto",
				Motivo:    "Fonti RSS segnalano possibile scadenza/abolizione",
				Timestamp: time.Now(),
				Urgenza:   "alta",
			})
			matched++
		} else if confermaScore >= 1.5 {
			// Update confirmation — don't change status but add alert
			AddAlert(Alert{
				BonusID:   b.ID,
				BonusNome: b.Nome,
				OldStato:  b.StatoValidita,
				NewStato:  "attivo",
				Motivo:    "Conferma/proroga rilevata da fonti RSS",
				Timestamp: time.Now(),
				Urgenza:   "bassa",
			})
			matched++
		} else if scadenzaScore > 0 || confermaScore > 0 {
			// Weak signal — alert only
			AddAlert(Alert{
				BonusID:   b.ID,
				BonusNome: b.Nome,
				OldStato:  b.StatoValidita,
				NewStato:  b.StatoValidita,
				Motivo:    "Segnale debole da RSS — verificare manualmente",
				Timestamp: time.Now(),
				Urgenza:   "bassa",
			})
		}
	}

	logger.Info("news check completed", map[string]interface{}{
		"articles_fetched": len(articles),
		"bonuses_matched":  matched,
	})
}

type fetchedArticle struct {
	Title       string
	Description string
	Date        time.Time
	Source      string
	Trust       float64
}

func fetchAllFeeds() []fetchedArticle {
	type result struct {
		articles []fetchedArticle
	}

	ch := make(chan result, len(rssFeeds))

	for _, feed := range rssFeeds {
		go func(f rssFeed) {
			arts := fetchFeed(f)
			ch <- result{articles: arts}
		}(feed)
	}

	var all []fetchedArticle
	for range rssFeeds {
		r := <-ch
		all = append(all, r.articles...)
	}
	return all
}

func fetchFeed(f rssFeed) []fetchedArticle {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", f.URL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", config.Cfg.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		logger.Warn("news: feed fetch failed", map[string]interface{}{"feed": f.Name, "error": err.Error()})
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB max
	if err != nil {
		return nil
	}

	var doc rssDocument
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil
	}

	var articles []fetchedArticle
	for _, item := range doc.Channel.Items {
		pubDate := parseRSSDate(item.PubDate)
		articles = append(articles, fetchedArticle{
			Title:       item.Title,
			Description: item.Description,
			Date:        pubDate,
			Source:      f.Name,
			Trust:       f.Trust,
		})
	}
	return articles
}

func parseRSSDate(s string) time.Time {
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
