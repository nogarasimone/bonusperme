package scraper

import (
	"bonusperme/internal/config"
	"bonusperme/internal/logger"
	"bonusperme/internal/models"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxChangeEvents = 500

// ChangeEvent records a detected change in bonus data.
type ChangeEvent struct {
	BonusID   string    `json:"bonus_id"`
	BonusNome string    `json:"bonus_nome"`
	Field     string    `json:"field"`
	OldValue  string    `json:"old_value"`
	NewValue  string    `json:"new_value"`
	Source    string    `json:"source,omitempty"`
	Trust     float64   `json:"trust,omitempty"`
	Severity  string    `json:"severity"` // "critical", "high", "medium", "low"
	Timestamp time.Time `json:"timestamp"`
}

var (
	changesMu sync.Mutex
	changes   []ChangeEvent
)

// DetectChanges compares old and new bonus lists, recording significant changes.
func DetectChanges(old, new []models.Bonus) {
	oldMap := make(map[string]models.Bonus)
	for _, b := range old {
		oldMap[b.ID] = b
	}

	newMap := make(map[string]models.Bonus)
	for _, b := range new {
		newMap[b.ID] = b
	}

	now := time.Now()

	// Check for changed/disappeared bonuses
	for id, oldB := range oldMap {
		newB, exists := newMap[id]
		if !exists {
			// Bonus disappeared
			addChange(ChangeEvent{
				BonusID:   id,
				BonusNome: oldB.Nome,
				Field:     "esistenza",
				OldValue:  "presente",
				NewValue:  "rimosso",
				Severity:  "critical",
				Timestamp: now,
			})
			continue
		}

		// Compare fields
		if oldB.Importo != newB.Importo && oldB.Importo != "" && newB.Importo != "" {
			sev := importoSeverity(oldB.Importo, newB.Importo)
			addChange(ChangeEvent{
				BonusID:   id,
				BonusNome: newB.Nome,
				Field:     "importo",
				OldValue:  oldB.Importo,
				NewValue:  newB.Importo,
				Source:    newB.FonteNome,
				Severity:  sev,
				Timestamp: now,
			})
		}

		if oldB.Scadenza != newB.Scadenza && oldB.Scadenza != "" && newB.Scadenza != "" {
			addChange(ChangeEvent{
				BonusID:   id,
				BonusNome: newB.Nome,
				Field:     "scadenza",
				OldValue:  oldB.Scadenza,
				NewValue:  newB.Scadenza,
				Source:    newB.FonteNome,
				Severity:  "high",
				Timestamp: now,
			})
		}

		if oldB.Stato != newB.Stato && oldB.Stato != "" && newB.Stato != "" {
			addChange(ChangeEvent{
				BonusID:   id,
				BonusNome: newB.Nome,
				Field:     "stato",
				OldValue:  oldB.Stato,
				NewValue:  newB.Stato,
				Source:    newB.FonteNome,
				Severity:  "high",
				Timestamp: now,
			})
		}

		if oldB.LinkUfficiale != newB.LinkUfficiale && oldB.LinkUfficiale != "" && newB.LinkUfficiale != "" {
			addChange(ChangeEvent{
				BonusID:   id,
				BonusNome: newB.Nome,
				Field:     "link_ufficiale",
				OldValue:  oldB.LinkUfficiale,
				NewValue:  newB.LinkUfficiale,
				Source:    newB.FonteNome,
				Severity:  "medium",
				Timestamp: now,
			})
		}

		if oldB.Descrizione != newB.Descrizione && oldB.Descrizione != "" && newB.Descrizione != "" {
			addChange(ChangeEvent{
				BonusID:   id,
				BonusNome: newB.Nome,
				Field:     "descrizione",
				OldValue:  truncate(oldB.Descrizione, 100),
				NewValue:  truncate(newB.Descrizione, 100),
				Source:    newB.FonteNome,
				Severity:  "low",
				Timestamp: now,
			})
		}
	}

	// Check for new bonuses
	for id, newB := range newMap {
		if _, exists := oldMap[id]; !exists {
			addChange(ChangeEvent{
				BonusID:   id,
				BonusNome: newB.Nome,
				Field:     "esistenza",
				OldValue:  "",
				NewValue:  "nuovo",
				Source:    newB.FonteNome,
				Severity:  "medium",
				Timestamp: now,
			})
		}
	}
}

func addChange(ev ChangeEvent) {
	changesMu.Lock()
	defer changesMu.Unlock()
	changes = append(changes, ev)
	if len(changes) > maxChangeEvents {
		changes = changes[len(changes)-maxChangeEvents:]
	}
	logger.Info("change detected", map[string]interface{}{
		"bonus_id": ev.BonusID, "field": ev.Field,
		"severity": ev.Severity, "old": ev.OldValue, "new": ev.NewValue,
	})
}

// GetChanges returns a copy of recent change events (newest first).
func GetChanges() []ChangeEvent {
	changesMu.Lock()
	defer changesMu.Unlock()
	result := make([]ChangeEvent, len(changes))
	copy(result, changes)
	// Reverse for newest-first
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result
}

// AdminChangesHandler serves GET /api/admin/changes.
// Protected by ADMIN_API_KEY.
func AdminChangesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !checkAdminKey(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	result := GetChanges()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(result)
}

func checkAdminKey(r *http.Request) bool {
	key := config.Cfg.AdminAPIKey
	if key == "" {
		return true
	}
	if r.URL.Query().Get("key") == key {
		return true
	}
	if r.Header.Get("X-Admin-Key") == key {
		return true
	}
	return false
}

// importoSeverity determines severity based on how much the amount changed.
func importoSeverity(oldVal, newVal string) string {
	oldNum := extractNumber(oldVal)
	newNum := extractNumber(newVal)
	if oldNum == 0 || newNum == 0 {
		return "medium"
	}
	change := math.Abs(newNum-oldNum) / oldNum
	if change > 0.5 {
		return "critical"
	}
	return "medium"
}

func extractNumber(s string) float64 {
	s = strings.ReplaceAll(s, "€", "")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", ".")
	s = strings.TrimSpace(s)
	// Try to find a number
	for _, part := range strings.Fields(s) {
		part = strings.TrimLeft(part, "€ ")
		if n, err := strconv.ParseFloat(part, 64); err == nil {
			return n
		}
	}
	return 0
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
