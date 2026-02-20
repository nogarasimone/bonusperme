package datasource

import (
	"bonusperme/internal/models"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// GURSSSource monitors Gazzetta Ufficiale RSS feed for new bonus-related legislation.
type GURSSSource struct {
	client *http.Client
}

func (s *GURSSSource) Name() string    { return "GazzettaUfficiale" }
func (s *GURSSSource) Enabled() bool   { return true }

type rssChannel struct {
	Items []rssItem `xml:"channel>item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func (s *GURSSSource) Fetch() ([]models.Bonus, error) {
	url := "https://www.gazzettaufficiale.it/rss/SG"
	body, err := fetchURL(s.client, url)
	if err != nil {
		return nil, fmt.Errorf("GU RSS fetch: %w", err)
	}

	var rss rssChannel
	if err := xml.Unmarshal(body, &rss); err != nil {
		return nil, fmt.Errorf("GU RSS parse: %w", err)
	}

	var bonuses []models.Bonus
	now := time.Now().Format("2006-01-02")

	for _, item := range rss.Items {
		titleLower := strings.ToLower(item.Title)
		descLower := strings.ToLower(item.Description)

		if containsBonusKeyword(titleLower) || containsBonusKeyword(descLower) {
			bonus := models.Bonus{
				ID:                   slugify(item.Title),
				Nome:                 strings.TrimSpace(item.Title),
				Categoria:            categorize(titleLower),
				Descrizione:          "Pubblicazione in Gazzetta Ufficiale. Verificare i dettagli sul sito ufficiale.",
				Requisiti:            []string{"Da definire — consultare il testo di legge"},
				ComeRichiederlo:      []string{"Attendere i decreti attuativi per le modalita di richiesta"},
				LinkUfficiale:        item.Link,
				Ente:                 "Gazzetta Ufficiale",
				Fonte:                "gu",
				FonteURL:             url,
				FonteNome:            "Gazzetta Ufficiale della Repubblica Italiana",
				RiferimentiNormativi: []string{item.Title},
				UltimoAggiornamento:  now,
				Stato:                "attivo",
			}
			bonuses = append(bonuses, bonus)
		}
	}

	return bonuses, nil
}
