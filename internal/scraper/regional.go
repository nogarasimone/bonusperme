package scraper

import (
	"bonusperme/internal/config"
	"bonusperme/internal/logger"
	"bonusperme/internal/matcher"
	"bonusperme/internal/models"
	sentryutil "bonusperme/internal/sentry"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// RegionalSource describes a source for regional bonus data.
type RegionalSource struct {
	Regione string
	URL     string
	Tipo    string // "regionale" | "aggregatore"
	Parser  func(body string, regione string) ([]models.Bonus, error)
}

// RegionalSources lists the sources for regional data.
var RegionalSources = []RegionalSource{
	{Regione: "*", URL: "https://www.ticonsiglio.com/bonus-regionali/", Tipo: "aggregatore", Parser: parseRegionalAggregator},
	{Regione: "Piemonte", URL: "https://www.regione.piemonte.it/web/temi/diritti-politiche-sociali", Tipo: "regionale", Parser: parseRegionalGeneric},
	{Regione: "Lombardia", URL: "https://www.regione.lombardia.it/wps/portal/istituzionale/HP/servizi-e-informazioni/cittadini/scuola-formazione-e-lavoro/dote-scuola", Tipo: "regionale", Parser: parseRegionalGeneric},
	{Regione: "Emilia-Romagna", URL: "https://www.regione.emilia-romagna.it/", Tipo: "regionale", Parser: parseRegionalGeneric},
}

var regionalClient = &http.Client{
	Timeout: 15 * time.Second,
}

// FetchRegionalBonuses tries scraping, falls back to hardcoded data.
func FetchRegionalBonuses(regione string, hardcoded []models.Bonus) []models.Bonus {
	var scraped []models.Bonus
	scrapingOK := false

	for _, src := range RegionalSources {
		if src.Regione != regione && src.Regione != "*" {
			continue
		}

		result, err := tryRegionalSource(src, regione)
		if err != nil {
			logger.Warn("scraper_regional: source failed", map[string]interface{}{
				"regione": regione, "source": src.URL, "error": err.Error(),
			})
			sentryutil.CaptureError(err, map[string]string{
				"component": "scraper_regional",
				"regione":   regione,
				"source":    src.URL,
			})
			continue
		}
		if len(result) > 0 {
			scraped = append(scraped, result...)
			scrapingOK = true
			logger.Info("scraper_regional: OK", map[string]interface{}{
				"regione": regione, "count": len(result), "source": src.URL,
			})
			break
		}
	}

	if scrapingOK && len(scraped) > 0 {
		for i := range scraped {
			scraped[i].FonteAggiornamento = "scraping"
			scraped[i].UltimoAggiornamento = time.Now().Format("2006-01-02")
		}
		return scraped
	}

	// Fallback: hardcoded with disclaimer
	logger.Info("scraper_regional: fallback to hardcoded", map[string]interface{}{"regione": regione})
	for i := range hardcoded {
		hardcoded[i].FonteAggiornamento = "manuale"
		hardcoded[i].VerificaManualeNecessaria = true
		hardcoded[i].NotaVerifica = fmt.Sprintf(
			"Dato non aggiornato automaticamente per la regione %s. "+
				"Verificare sul sito ufficiale della regione. "+
				"Ultimo controllo manuale: dati inseriti nel codice sorgente.",
			regione,
		)
	}
	return hardcoded
}

func tryRegionalSource(src RegionalSource, regione string) ([]models.Bonus, error) {
	req, err := http.NewRequest("GET", src.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", config.Cfg.UserAgent)
	req.Header.Set("Accept-Language", "it-IT,it;q=0.9")

	resp, err := regionalClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, err
	}

	return src.Parser(string(body), regione)
}

// Parsers — placeholder implementations that trigger fallback.
// When a real parser is implemented, it returns actual bonuses.

func parseRegionalAggregator(body string, regione string) ([]models.Bonus, error) {
	// TiConsiglio.com aggregated page — not yet implemented
	_ = body
	_ = regione
	return nil, fmt.Errorf("parser aggregatore non ancora implementato")
}

func parseRegionalGeneric(body string, regione string) ([]models.Bonus, error) {
	// Generic regional page parser — not yet implemented
	_ = body
	_ = regione
	return nil, fmt.Errorf("parser regionale non ancora implementato per %s", regione)
}

// GetHardcodedRegionalForRegione returns hardcoded regionals for a specific region.
func GetHardcodedRegionalForRegione(regione string) []models.Bonus {
	all := matcher.GetRegionalBonus()
	var result []models.Bonus
	regioneLower := strings.ToLower(regione)
	for _, b := range all {
		for _, r := range b.RegioniApplicabili {
			if strings.ToLower(r) == regioneLower {
				result = append(result, b)
				break
			}
		}
	}
	return result
}
