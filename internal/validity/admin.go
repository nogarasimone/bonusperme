package validity

import (
	"bonusperme/internal/config"
	"encoding/json"
	"net/http"
)

// AdminAlertsHandler serves GET /api/admin/alerts.
// Protected by ADMIN_API_KEY (query param "key" or header "X-Admin-Key").
func AdminAlertsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !checkAdminKey(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	alerts := GetAlerts()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(alerts)
}

// AdminBonusStatusHandler serves GET /api/admin/bonus-status.
// Returns validity status of all bonuses from cache.
func AdminBonusStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !checkAdminKey(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	type statusEntry struct {
		BonusID       string `json:"bonus_id"`
		StatoValidita string `json:"stato_validita"`
		MotivoStato   string `json:"motivo_stato"`
		UpdatedAt     string `json:"updated_at"`
	}

	var entries []statusEntry
	statusCache.Range(func(key, value interface{}) bool {
		vs := value.(validityStatus)
		entries = append(entries, statusEntry{
			BonusID:       key.(string),
			StatoValidita: vs.StatoValidita,
			MotivoStato:   vs.MotivoStato,
			UpdatedAt:     vs.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
		return true
	})

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(entries)
}

func checkAdminKey(r *http.Request) bool {
	key := config.Cfg.AdminAPIKey
	if key == "" {
		return true // no key configured = open access (dev mode)
	}
	// Check query param
	if r.URL.Query().Get("key") == key {
		return true
	}
	// Check header
	if r.Header.Get("X-Admin-Key") == key {
		return true
	}
	return false
}
