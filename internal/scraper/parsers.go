package scraper

import (
	"bonusperme/internal/config"
	"bonusperme/internal/models"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	},
}

var bonusKeywords = []string{"bonus", "assegno", "detrazione", "agevolazione", "contributo", "carta", "esonero"}

var (
	amountRegex = regexp.MustCompile(`(?:€|euro)\s*([0-9][0-9.,]*)`)
	iseeRegex   = regexp.MustCompile(`(?i)isee\s*(?:fino a|entro|non superiore a|massimo)?\s*(?:€|euro)?\s*([0-9][0-9.,]*)`)
	dateRegex   = regexp.MustCompile(`(\d{1,2})\s+(gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)\s+(\d{4})`)
)

func fetchURL(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", config.Cfg.UserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
}

// ParseSource fetches a source URL and parses the HTML for bonus information.
// It never panics — panics are recovered and logged.
// Uses FetchWithCache for conditional HTTP requests and retries.
func ParseSource(src Source) (bonuses []models.Bonus) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[scraper] panic in parser %s: %v", src.Name, r)
			bonuses = nil
		}
	}()

	body, _, err := GetHTTPCache().FetchWithCache(src.URL, httpClient)
	if err != nil {
		log.Printf("[scraper] fetch error %s: %v", src.Name, err)
		return nil
	}

	return ParseSourceFromBody(src, body)
}

// ParseSourceFromBody parses pre-fetched HTML body for bonus information.
func ParseSourceFromBody(src Source, body []byte) []models.Bonus {
	switch src.Parser {
	case "inps":
		return parseINPS(body, src)
	case "ade":
		return parseADE(body, src)
	case "editorial":
		return parseEditorial(body, src)
	default:
		return parseGeneric(body, src)
	}
}

// parseINPS extracts bonus data from INPS pages.
func parseINPS(body []byte, src Source) []models.Bonus {
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		log.Printf("[scraper] HTML parse error %s: %v", src.Name, err)
		return nil
	}

	var bonuses []models.Bonus
	now := time.Now().Format("2 January 2006")

	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "h2" || n.Data == "h3" || n.Data == "h4" || n.Data == "a") {
			text := getTextContent(n)
			textLower := strings.ToLower(text)

			if containsBonusKeyword(textLower) && len(text) > 10 && len(text) < 200 {
				link := ""
				if n.Data == "a" {
					link = getAttr(n, "href")
					if link != "" && !strings.HasPrefix(link, "http") {
						link = "https://www.inps.it" + link
					}
				}

				// Look for amount in nearby text
				importo := ""
				if parent := n.Parent; parent != nil {
					parentText := getTextContent(parent)
					if matches := amountRegex.FindStringSubmatch(parentText); len(matches) > 1 {
						importo = "\u20ac" + matches[1]
					}
				}

				bonus := models.Bonus{
					ID:                  slugify(text),
					Nome:                strings.TrimSpace(text),
					Categoria:           categorize(textLower),
					Descrizione:         fmt.Sprintf("Informazione trovata su %s. Verificare sul sito ufficiale per dettagli aggiornati.", src.Name),
					Importo:             importo,
					Scadenza:            "",
					Requisiti:           []string{"Consultare il sito ufficiale per i requisiti aggiornati"},
					ComeRichiederlo:     []string{"Visitare il sito ufficiale dell'ente erogatore"},
					LinkUfficiale:       link,
					Ente:                "INPS",
					Fonte:               src.Type,
					FonteURL:            src.URL,
					FonteNome:           src.Name,
					UltimoAggiornamento: now,
					Stato:               "attivo",
				}
				bonuses = append(bonuses, bonus)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)

	return bonuses
}

// parseADE extracts bonus data from Agenzia delle Entrate pages.
func parseADE(body []byte, src Source) []models.Bonus {
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		log.Printf("[scraper] HTML parse error %s: %v", src.Name, err)
		return nil
	}

	var bonuses []models.Bonus
	now := time.Now().Format("2 January 2006")

	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "h2" || n.Data == "h3" || n.Data == "h4" || n.Data == "a") {
			text := getTextContent(n)
			textLower := strings.ToLower(text)

			if containsBonusKeyword(textLower) && len(text) > 10 && len(text) < 200 {
				link := ""
				if n.Data == "a" {
					link = getAttr(n, "href")
					if link != "" && !strings.HasPrefix(link, "http") {
						link = "https://www.agenziaentrate.gov.it" + link
					}
				}

				// Look for percentage deductions typical of AdE
				importo := ""
				if parent := n.Parent; parent != nil {
					parentText := getTextContent(parent)
					if matches := amountRegex.FindStringSubmatch(parentText); len(matches) > 1 {
						importo = "\u20ac" + matches[1]
					}
				}

				bonus := models.Bonus{
					ID:                  slugify(text),
					Nome:                strings.TrimSpace(text),
					Categoria:           categorize(textLower),
					Descrizione:         fmt.Sprintf("Informazione trovata su %s. Verificare sul sito ufficiale per dettagli aggiornati.", src.Name),
					Importo:             importo,
					Scadenza:            "",
					Requisiti:           []string{"Consultare il sito ufficiale per i requisiti aggiornati"},
					ComeRichiederlo:     []string{"Visitare il sito ufficiale dell'ente erogatore"},
					LinkUfficiale:       link,
					Ente:                "Agenzia delle Entrate",
					Fonte:               src.Type,
					FonteURL:            src.URL,
					FonteNome:           src.Name,
					UltimoAggiornamento: now,
					Stato:               "attivo",
				}
				bonuses = append(bonuses, bonus)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)

	return bonuses
}

// parseEditorial extracts bonus data from editorial/news sites using aggressive regex matching.
func parseEditorial(body []byte, src Source) []models.Bonus {
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		log.Printf("[scraper] HTML parse error %s: %v", src.Name, err)
		return nil
	}

	var bonuses []models.Bonus
	now := time.Now().Format("2 January 2006")
	seen := make(map[string]bool)

	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "h2" || n.Data == "h3" || n.Data == "h4") {
			text := getTextContent(n)
			textLower := strings.ToLower(text)

			if containsBonusKeyword(textLower) && len(text) > 10 && len(text) < 200 {
				slug := slugify(text)
				if seen[slug] {
					goto nextChildren
				}
				seen[slug] = true

				// Aggressively extract amounts, ISEE, and dates from surrounding context
				contextText := ""
				if n.Parent != nil {
					contextText = getTextContent(n.Parent)
				}

				importo := ""
				if matches := amountRegex.FindStringSubmatch(contextText); len(matches) > 1 {
					importo = "\u20ac" + matches[1]
				}

				scadenza := ""
				if matches := dateRegex.FindStringSubmatch(contextText); len(matches) > 3 {
					scadenza = matches[1] + " " + matches[2] + " " + matches[3]
				}

				requisiti := []string{"Consultare il sito ufficiale per i requisiti aggiornati"}
				if matches := iseeRegex.FindStringSubmatch(contextText); len(matches) > 1 {
					requisiti = append(requisiti, fmt.Sprintf("ISEE fino a \u20ac%s", matches[1]))
				}

				bonus := models.Bonus{
					ID:                  slug,
					Nome:                strings.TrimSpace(text),
					Categoria:           categorize(textLower),
					Descrizione:         fmt.Sprintf("Informazione trovata su %s. Verificare sul sito ufficiale per dettagli aggiornati.", src.Name),
					Importo:             importo,
					Scadenza:            scadenza,
					Requisiti:           requisiti,
					ComeRichiederlo:     []string{"Visitare il sito ufficiale dell'ente erogatore"},
					Ente:                "",
					Fonte:               src.Type,
					FonteURL:            src.URL,
					FonteNome:           src.Name,
					UltimoAggiornamento: now,
					Stato:               "attivo",
				}
				bonuses = append(bonuses, bonus)
			}
		}

		// Also check links for bonus references
		if n.Type == html.ElementNode && n.Data == "a" {
			text := getTextContent(n)
			textLower := strings.ToLower(text)

			if containsBonusKeyword(textLower) && len(text) > 10 && len(text) < 200 {
				slug := slugify(text)
				if seen[slug] {
					goto nextChildren
				}
				seen[slug] = true

				link := getAttr(n, "href")

				bonus := models.Bonus{
					ID:                  slug,
					Nome:                strings.TrimSpace(text),
					Categoria:           categorize(textLower),
					Descrizione:         fmt.Sprintf("Informazione trovata su %s. Verificare sul sito ufficiale per dettagli aggiornati.", src.Name),
					LinkUfficiale:       link,
					Fonte:               src.Type,
					FonteURL:            src.URL,
					FonteNome:           src.Name,
					UltimoAggiornamento: now,
					Stato:               "attivo",
				}
				bonuses = append(bonuses, bonus)
			}
		}

	nextChildren:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)

	return bonuses
}

// parseGeneric extracts bonus data from any HTML page using heading and link scanning.
func parseGeneric(body []byte, src Source) []models.Bonus {
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		log.Printf("[scraper] HTML parse error %s: %v", src.Name, err)
		return nil
	}

	var bonuses []models.Bonus
	now := time.Now().Format("2 January 2006")

	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "h2" || n.Data == "h3" || n.Data == "h4" || n.Data == "a") {
			text := getTextContent(n)
			textLower := strings.ToLower(text)

			if containsBonusKeyword(textLower) && len(text) > 10 && len(text) < 200 {
				link := ""
				if n.Data == "a" {
					link = getAttr(n, "href")
				}

				bonus := models.Bonus{
					ID:                  slugify(text),
					Nome:                strings.TrimSpace(text),
					Categoria:           categorize(textLower),
					Descrizione:         fmt.Sprintf("Informazione trovata su %s. Verificare sul sito ufficiale per dettagli aggiornati.", src.Name),
					Requisiti:           []string{"Consultare il sito ufficiale per i requisiti aggiornati"},
					ComeRichiederlo:     []string{"Visitare il sito ufficiale dell'ente erogatore"},
					LinkUfficiale:       link,
					Fonte:               src.Type,
					FonteURL:            src.URL,
					FonteNome:           src.Name,
					UltimoAggiornamento: now,
					Stato:               "attivo",
				}
				bonuses = append(bonuses, bonus)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)

	return bonuses
}

// --- Helper functions ---

func getTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(getTextContent(c))
	}
	return sb.String()
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, s)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	if len(s) > 60 {
		s = s[:60]
	}
	return s
}

func categorize(text string) string {
	switch {
	case strings.Contains(text, "famiglia") || strings.Contains(text, "figlio") || strings.Contains(text, "nido") || strings.Contains(text, "nascita") || strings.Contains(text, "mamma"):
		return "famiglia"
	case strings.Contains(text, "casa") || strings.Contains(text, "ristruttur") || strings.Contains(text, "affitto") || strings.Contains(text, "abitazione"):
		return "casa"
	case strings.Contains(text, "salute") || strings.Contains(text, "psicolog"):
		return "salute"
	case strings.Contains(text, "studio") || strings.Contains(text, "cultura") || strings.Contains(text, "istruzione"):
		return "istruzione"
	case strings.Contains(text, "spesa") || strings.Contains(text, "alimentar"):
		return "spesa"
	case strings.Contains(text, "lavoro") || strings.Contains(text, "formazione"):
		return "lavoro"
	default:
		return "altro"
	}
}

func containsBonusKeyword(textLower string) bool {
	for _, kw := range bonusKeywords {
		if strings.Contains(textLower, kw) {
			return true
		}
	}
	return false
}
