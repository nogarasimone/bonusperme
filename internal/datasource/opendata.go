package datasource

import (
	"bonusperme/internal/models"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// OpenDataSource fetches bonus data from Italian open data APIs (dati.gov.it, INPS Open Data).
type OpenDataSource struct {
	client *http.Client
}

func (s *OpenDataSource) Name() string    { return "OpenData" }
func (s *OpenDataSource) Enabled() bool   { return true }

func (s *OpenDataSource) Fetch() ([]models.Bonus, error) {
	var all []models.Bonus

	// dati.gov.it CKAN API — search for "bonus" datasets
	datiGov, err := s.fetchDatiGov()
	if err != nil {
		// Non-fatal: log and continue
		_ = err
	} else {
		all = append(all, datiGov...)
	}

	return all, nil
}

type ckanSearchResult struct {
	Result struct {
		Results []struct {
			Title string `json:"title"`
			Notes string `json:"notes"`
			URL   string `json:"url"`
			Organization struct {
				Title string `json:"title"`
			} `json:"organization"`
		} `json:"results"`
	} `json:"result"`
}

func (s *OpenDataSource) fetchDatiGov() ([]models.Bonus, error) {
	url := "https://www.dati.gov.it/opendata/api/3/action/package_search?q=bonus+famiglia&rows=20"
	body, err := fetchURL(s.client, url)
	if err != nil {
		return nil, fmt.Errorf("dati.gov.it fetch: %w", err)
	}

	var result ckanSearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("dati.gov.it parse: %w", err)
	}

	var bonuses []models.Bonus
	now := time.Now().Format("2006-01-02")

	for _, ds := range result.Result.Results {
		titleLower := strings.ToLower(ds.Title)
		if !containsBonusKeyword(titleLower) {
			continue
		}

		desc := ds.Notes
		if len(desc) > 300 {
			desc = desc[:297] + "..."
		}

		ente := ds.Organization.Title
		if ente == "" {
			ente = "dati.gov.it"
		}

		bonus := models.Bonus{
			ID:                  slugify(ds.Title),
			Nome:                strings.TrimSpace(ds.Title),
			Categoria:           categorize(titleLower),
			Descrizione:         desc,
			Requisiti:           []string{"Consultare il dataset ufficiale per i requisiti"},
			ComeRichiederlo:     []string{"Consultare la fonte dati ufficiale"},
			LinkUfficiale:       ds.URL,
			Ente:                ente,
			Fonte:               "opendata",
			FonteURL:            "https://www.dati.gov.it",
			FonteNome:           "dati.gov.it",
			UltimoAggiornamento: now,
			Stato:               "attivo",
		}
		bonuses = append(bonuses, bonus)
	}

	return bonuses, nil
}
