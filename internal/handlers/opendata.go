package handlers

import (
	"bonusperme/internal/linkcheck"
	"bonusperme/internal/matcher"
	"bonusperme/internal/scraper"
	"bonusperme/internal/validity"
	"encoding/json"
	"net/http"
	"strings"
)

func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Max-Age", "86400")
}

// BonusListHandler returns the full list of all bonuses (national + regional) as JSON.
// GET /api/bonus
func BonusListHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Try cached (enriched) bonuses first, fallback to hardcoded
	allBonus := scraper.GetCachedBonus()
	if len(allBonus) == 0 {
		allBonus = matcher.GetAllBonusWithRegional()
	} else {
		// Append regional bonuses if not already present
		regionals := matcher.GetRegionalBonus()
		allBonus = append(allBonus, regionals...)
	}

	linkcheck.ApplyStatus(allBonus)
	validity.ApplyStatus(allBonus)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	json.NewEncoder(w).Encode(allBonus)
}

// BonusDetailHandler returns a single bonus by ID.
// GET /api/bonus/{id}
func BonusDetailHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path: /api/bonus/xxx
	path := strings.TrimPrefix(r.URL.Path, "/api/bonus/")
	bonusID := strings.TrimSuffix(path, "/")
	if bonusID == "" {
		http.Error(w, "Bonus ID richiesto", http.StatusBadRequest)
		return
	}

	allBonus := matcher.GetAllBonusWithRegional()
	linkcheck.ApplyStatus(allBonus)
	validity.ApplyStatus(allBonus)
	for _, b := range allBonus {
		if b.ID == bonusID {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "public, max-age=3600")
			json.NewEncoder(w).Encode(b)
			return
		}
	}

	http.Error(w, "Bonus non trovato", http.StatusNotFound)
}
