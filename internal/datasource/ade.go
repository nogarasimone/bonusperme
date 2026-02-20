package datasource

import (
	"bonusperme/internal/models"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

var adeURLs = []string{
	"https://www.agenziaentrate.gov.it/portale/web/guest/aree-tematiche/casa/agevolazioni",
	"https://www.agenziaentrate.gov.it/portale/web/guest/agevolazioni",
}

// AdESource scrapes bonus data from Agenzia delle Entrate.
type AdESource struct {
	client *http.Client
}

func (s *AdESource) Name() string    { return "AdE" }
func (s *AdESource) Enabled() bool   { return true }

func (s *AdESource) Fetch() ([]models.Bonus, error) {
	var all []models.Bonus
	for _, url := range adeURLs {
		body, err := fetchURL(s.client, url)
		if err != nil {
			return nil, fmt.Errorf("AdE fetch %s: %w", url, err)
		}
		bonuses := parseAdEPage(body, url)
		all = append(all, bonuses...)
		time.Sleep(2 * time.Second)
	}
	return all, nil
}

func parseAdEPage(body []byte, sourceURL string) []models.Bonus {
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
						link = "https://www.agenziaentrate.gov.it" + link
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
					Descrizione:         "Informazione da Agenzia delle Entrate. Verificare sul sito ufficiale per dettagli aggiornati.",
					Importo:             importo,
					Requisiti:           []string{"Consultare il sito ufficiale per i requisiti aggiornati"},
					ComeRichiederlo:     []string{"Visitare il sito ufficiale dell'Agenzia delle Entrate"},
					LinkUfficiale:       link,
					Ente:                "Agenzia delle Entrate",
					Fonte:               "ade",
					FonteURL:            sourceURL,
					FonteNome:           "Agenzia delle Entrate",
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
