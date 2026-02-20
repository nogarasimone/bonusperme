package validity

import (
	"sync"
	"time"
)

const maxAlerts = 100

// Alert represents a validity state change event.
type Alert struct {
	BonusID   string    `json:"bonus_id"`
	BonusNome string    `json:"bonus_nome,omitempty"`
	OldStato  string    `json:"old_stato"`
	NewStato  string    `json:"new_stato"`
	Motivo    string    `json:"motivo"`
	Timestamp time.Time `json:"timestamp"`
	Urgenza   string    `json:"urgenza"`
}

var (
	alertsMu sync.Mutex
	alerts   []Alert
)

// AddAlert appends an alert to the ring buffer (max 100).
func AddAlert(a Alert) {
	alertsMu.Lock()
	defer alertsMu.Unlock()
	alerts = append(alerts, a)
	if len(alerts) > maxAlerts {
		alerts = alerts[len(alerts)-maxAlerts:]
	}
}

// GetAlerts returns a copy of recent alerts (newest first).
func GetAlerts() []Alert {
	alertsMu.Lock()
	defer alertsMu.Unlock()
	result := make([]Alert, len(alerts))
	copy(result, alerts)
	// Reverse for newest-first
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result
}
