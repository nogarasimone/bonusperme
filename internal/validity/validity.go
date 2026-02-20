package validity

import (
	"bonusperme/internal/models"
	"fmt"
	"math"
	"sync"
	"time"
)

var statusCache sync.Map // map[string]validityStatus

type validityStatus struct {
	StatoValidita string
	MotivoStato   string
	UpdatedAt     time.Time
}

// ApplyStatus patches StatoValidita and MotivoStato from cache onto bonuses.
// Also syncs the Scaduto bool based on validity state.
func ApplyStatus(bonuses []models.Bonus) {
	for i := range bonuses {
		if v, ok := statusCache.Load(bonuses[i].ID); ok {
			vs := v.(validityStatus)
			bonuses[i].StatoValidita = vs.StatoValidita
			bonuses[i].MotivoStato = vs.MotivoStato
			// Sync Scaduto with validity
			if vs.StatoValidita == "scaduto" || vs.StatoValidita == "potenzialmente_scaduto" {
				bonuses[i].Scaduto = true
			}
		}
	}
}

// GenerateAvvisi builds warnings for non-"attivo" bonuses.
func GenerateAvvisi(bonuses []models.Bonus) []models.Avviso {
	var avvisi []models.Avviso
	for _, b := range bonuses {
		if b.Scaduto {
			continue // already shown as expired
		}
		switch b.StatoValidita {
		case "in_scadenza":
			days := 0
			if !b.ScadenzaDomanda.IsZero() {
				days = int(math.Ceil(time.Until(b.ScadenzaDomanda).Hours() / 24))
				if days < 0 {
					days = 0
				}
			}
			msg := "Questo bonus scade a breve"
			if days > 0 {
				msg = fmt.Sprintf("Scade tra %d giorni — Fai domanda subito", days)
			}
			avvisi = append(avvisi, models.Avviso{
				BonusID:   b.ID,
				Tipo:      "warning",
				Messaggio: msg,
			})
		case "da_verificare":
			avvisi = append(avvisi, models.Avviso{
				BonusID:   b.ID,
				Tipo:      "info",
				Messaggio: "Verifica disponibilità sul sito ufficiale",
			})
		case "potenzialmente_scaduto":
			avvisi = append(avvisi, models.Avviso{
				BonusID:   b.ID,
				Tipo:      "danger",
				Messaggio: "Questo bonus potrebbe non essere più disponibile",
			})
		}
	}
	return avvisi
}

// SetStatus stores a validity status in cache (used by checker and news).
func SetStatus(bonusID, stato, motivo string) {
	old := ""
	if v, ok := statusCache.Load(bonusID); ok {
		old = v.(validityStatus).StatoValidita
	}
	statusCache.Store(bonusID, validityStatus{
		StatoValidita: stato,
		MotivoStato:   motivo,
		UpdatedAt:     time.Now(),
	})
	// Generate alert if status changed
	if old != "" && old != stato {
		AddAlert(Alert{
			BonusID:   bonusID,
			OldStato:  old,
			NewStato:  stato,
			Motivo:    motivo,
			Timestamp: time.Now(),
			Urgenza:   alertUrgency(stato),
		})
	}
}

func alertUrgency(stato string) string {
	switch stato {
	case "scaduto", "potenzialmente_scaduto":
		return "alta"
	case "in_scadenza":
		return "media"
	default:
		return "bassa"
	}
}
