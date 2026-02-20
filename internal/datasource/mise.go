package datasource

import (
	"bonusperme/internal/models"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// MISESource scrapes bonus data from Ministero delle Imprese e del Made in Italy.
type MISESource struct {
	client *http.Client
}

func (s *MISESource) Name() string    { return "MISE" }
func (s *MISESource) Enabled() bool   { return true }

func (s *MISESource) Fetch() ([]models.Bonus, error) {
	url := "https://www.mimit.gov.it/it/incentivi"
	body, err := fetchURL(s.client, url)
	if err != nil {
		return nil, fmt.Errorf("MISE fetch: %w", err)
	}
	return parseMISEPage(body, url), nil
}

func parseMISEPage(body []byte, sourceURL string) []models.Bonus {
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
						link = "https://www.mimit.gov.it" + link
					}
				}

				bonus := models.Bonus{
					ID:                  slugify(text),
					Nome:                strings.TrimSpace(text),
					Categoria:           categorize(textLower),
					Descrizione:         "Informazione da MISE/MIMIT. Verificare sul sito ufficiale per dettagli aggiornati.",
					Requisiti:           []string{"Consultare il sito ufficiale per i requisiti aggiornati"},
					ComeRichiederlo:     []string{"Visitare il sito ufficiale del Ministero"},
					LinkUfficiale:       link,
					Ente:                "MISE/MIMIT",
					Fonte:               "mise",
					FonteURL:            sourceURL,
					FonteNome:           "MISE/MIMIT",
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
