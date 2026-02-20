package datasource

import (
	"bonusperme/internal/models"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

var inpsURLs = []string{
	"https://www.inps.it/it/it/sostegni-sussidi-indennita/per-genitori.html",
	"https://www.inps.it/it/it/sostegni-sussidi-indennita/per-famiglie.html",
}

// INPSSource scrapes bonus data from INPS official pages.
type INPSSource struct {
	client *http.Client
}

func (s *INPSSource) Name() string    { return "INPS" }
func (s *INPSSource) Enabled() bool   { return true }

func (s *INPSSource) Fetch() ([]models.Bonus, error) {
	var all []models.Bonus
	for _, url := range inpsURLs {
		body, err := fetchURL(s.client, url)
		if err != nil {
			return nil, fmt.Errorf("INPS fetch %s: %w", url, err)
		}
		bonuses := parseINPSPage(body, url)
		all = append(all, bonuses...)
		time.Sleep(2 * time.Second) // polite delay
	}
	return all, nil
}

var amountRegex = regexp.MustCompile(`(?:€|euro)\s*([0-9][0-9.,]*)`)

func parseINPSPage(body []byte, sourceURL string) []models.Bonus {
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil
	}

	var bonuses []models.Bonus
	now := time.Now().Format("2006-01-02")

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
					Descrizione:         "Informazione da INPS. Verificare sul sito ufficiale per dettagli aggiornati.",
					Importo:             importo,
					Requisiti:           []string{"Consultare il sito ufficiale per i requisiti aggiornati"},
					ComeRichiederlo:     []string{"Visitare il sito ufficiale INPS"},
					LinkUfficiale:       link,
					Ente:                "INPS",
					Fonte:               "inps",
					FonteURL:            sourceURL,
					FonteNome:           "INPS",
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
