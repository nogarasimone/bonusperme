package linkcheck

import (
	"bonusperme/internal/config"
	"encoding/json"
	"net/http"
	"sync/atomic"
)

// Link health counters tracked during CheckAllLinks.
var (
	totalChecked      int64
	verifiedCount     int64
	brokenCount       int64
	swappedToFallback int64
	usingWayback      int64
)

// BrokenDetail stores info about a single broken link.
type BrokenDetail struct {
	BonusID       string `json:"bonus_id"`
	OriginalURL   string `json:"original_url"`
	FallbackURL   string `json:"fallback_url,omitempty"`
	WaybackURL    string `json:"wayback_url,omitempty"`
	StatusCode    int    `json:"status_code"`
	RecoveryState string `json:"recovery_state"` // "none", "fallback", "wayback"
}

var (
	brokenDetails   []BrokenDetail
	brokenDetailsMu = &atomicMu{}
)

// atomicMu wraps a simple mutex pattern using atomic.
type atomicMu struct {
	// Using a simple slice with append; protected by the sync.Mutex in CheckAllLinks
}

func resetCounters() {
	atomic.StoreInt64(&totalChecked, 0)
	atomic.StoreInt64(&verifiedCount, 0)
	atomic.StoreInt64(&brokenCount, 0)
	atomic.StoreInt64(&swappedToFallback, 0)
	atomic.StoreInt64(&usingWayback, 0)
	brokenDetails = nil
}

// AdminLinksHandler serves GET /api/admin/links.
// Protected by ADMIN_API_KEY (query param "key" or header "X-Admin-Key").
func AdminLinksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !checkAdminKey(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	result := map[string]interface{}{
		"total":               atomic.LoadInt64(&totalChecked),
		"verified":            atomic.LoadInt64(&verifiedCount),
		"broken":              atomic.LoadInt64(&brokenCount),
		"swapped_to_fallback": atomic.LoadInt64(&swappedToFallback),
		"using_wayback":       atomic.LoadInt64(&usingWayback),
		"broken_details":      brokenDetails,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(result)
}

func checkAdminKey(r *http.Request) bool {
	key := config.Cfg.AdminAPIKey
	if key == "" {
		return true // no key configured = open access (dev mode)
	}
	if r.URL.Query().Get("key") == key {
		return true
	}
	if r.Header.Get("X-Admin-Key") == key {
		return true
	}
	return false
}
