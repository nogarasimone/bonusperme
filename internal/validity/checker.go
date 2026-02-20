package validity

import (
	"bonusperme/internal/logger"
	"bonusperme/internal/models"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// RunCheck evaluates all bonuses and stores results in statusCache.
func RunCheck(bonuses []models.Bonus) {
	now := time.Now()
	currentYear := now.Year()
	checked := 0

	for _, b := range bonuses {
		stato, motivo := evaluate(b, now, currentYear)
		SetStatus(b.ID, stato, motivo)
		checked++
	}

	logger.Info("validity check completed", map[string]interface{}{
		"checked": checked,
	})
}

// evaluate applies rules in priority order and returns (stato, motivo).
func evaluate(b models.Bonus, now time.Time, currentYear int) (string, string) {
	tipo := b.TipoScadenza

	// Rule 0: UltimoAggiornamento parseable and > 90 days old → da_verificare
	if b.UltimoAggiornamento != "" {
		if aggiornamento, ok := parseItalianDate(b.UltimoAggiornamento); ok {
			daysSince := int(now.Sub(aggiornamento).Hours() / 24)
			if daysSince > 90 {
				return "da_verificare", "Dati non aggiornati da " + itoa(daysSince) + " giorni"
			}
		}
	}

	// Rule 1: permanente + AnnoConferma >= current year → attivo
	if tipo == "permanente" && b.AnnoConferma >= currentYear {
		return "attivo", "Bonus permanente confermato per " + itoa(b.AnnoConferma)
	}

	// Rule 2: data_fissa and past → scaduto
	if tipo == "data_fissa" && !b.ScadenzaDomanda.IsZero() && now.After(b.ScadenzaDomanda) {
		return "scaduto", "Scadenza superata: " + b.ScadenzaDomanda.Format("02/01/2006")
	}

	// Rule 3: data_fissa within 30 days → in_scadenza
	if tipo == "data_fissa" && !b.ScadenzaDomanda.IsZero() {
		daysLeft := int(b.ScadenzaDomanda.Sub(now).Hours() / 24)
		if daysLeft <= 30 && daysLeft >= 0 {
			return "in_scadenza", "Scade tra " + itoa(daysLeft) + " giorni"
		}
	}

	// Rule 4: AnnoConferma < current year → da_verificare
	if b.AnnoConferma > 0 && b.AnnoConferma < currentYear {
		return "da_verificare", "Ultima conferma: " + itoa(b.AnnoConferma)
	}

	// Rule 5: bando_annuale + AnnoConferma < current year → da_verificare
	if tipo == "bando_annuale" && b.AnnoConferma > 0 && b.AnnoConferma < currentYear {
		return "da_verificare", "Bando annuale — verifica nuova edizione"
	}

	// Rule 6: UltimaVerifica > 60 days → da_verificare (if would be attivo)
	if !b.UltimaVerifica.IsZero() {
		daysSince := int(now.Sub(b.UltimaVerifica).Hours() / 24)
		if daysSince > 60 {
			return "da_verificare", "Ultima verifica " + itoa(daysSince) + " giorni fa"
		}
	}

	return "attivo", ""
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		return "-" + s
	}
	return s
}

var italianMonths = map[string]time.Month{
	"gennaio": time.January, "febbraio": time.February, "marzo": time.March,
	"aprile": time.April, "maggio": time.May, "giugno": time.June,
	"luglio": time.July, "agosto": time.August, "settembre": time.September,
	"ottobre": time.October, "novembre": time.November, "dicembre": time.December,
}

var itDatePattern = regexp.MustCompile(`(\d{1,2})\s+(gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)\s+(\d{4})`)

// parseItalianDate parses dates like "15 febbraio 2026".
func parseItalianDate(s string) (time.Time, bool) {
	lower := strings.ToLower(strings.TrimSpace(s))
	m := itDatePattern.FindStringSubmatch(lower)
	if len(m) != 4 {
		return time.Time{}, false
	}
	day, err := strconv.Atoi(m[1])
	if err != nil {
		return time.Time{}, false
	}
	month, ok := italianMonths[m[2]]
	if !ok {
		return time.Time{}, false
	}
	year, err := strconv.Atoi(m[3])
	if err != nil {
		return time.Time{}, false
	}
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), true
}
