package scraper

import (
	"bonusperme/internal/logger"
	"bonusperme/internal/matcher"
	"bonusperme/internal/models"
	"strings"
	"time"
)

// ensureHardcodedBase is a compile-time check that matcher.GetAllBonus is accessible.
var _ = matcher.GetAllBonus

// bonusAliases maps common alternative names to their canonical normalized name.
var bonusAliases = map[string]string{
	// Famiglia
	"carta nuovi nati":                     "carta per i nuovi nati",
	"bonus nascita":                        "carta per i nuovi nati",
	"bonus bebè":                           "carta per i nuovi nati",
	"bonus bebe":                           "carta per i nuovi nati",
	"bonus nascita 2026":                   "carta per i nuovi nati",
	"bonus mamma":                          "bonus mamme lavoratrici",
	"esonero contributivo mamme":           "bonus mamme lavoratrici",
	"esonero contributi madri lavoratrici": "bonus mamme lavoratrici",
	"assegno unico figli":                  "assegno unico universale",
	"assegno unico":                        "assegno unico universale",
	"auu":                                  "assegno unico universale",
	"bonus nido":                           "bonus asilo nido",
	"bonus asilo":                          "bonus asilo nido",
	// Casa
	"bonus casa":              "bonus ristrutturazione",
	"bonus ristrutturazioni":  "bonus ristrutturazione",
	"detrazione ristrutturaz": "bonus ristrutturazione",
	"bonus mobili 2026":       "bonus mobili ed elettrodomestici",
	"bonus elettrodomestici":  "bonus mobili ed elettrodomestici",
	"bonus affitto giovani":   "bonus affitto giovani under 31",
	"bonus affitto under 31":  "bonus affitto giovani under 31",
	"prima casa giovani":      "agevolazioni prima casa under 36",
	"prima casa under 36":     "agevolazioni prima casa under 36",
	"bonus verde giardini":    "bonus verde",
	// Altro
	"bonus psicologo 2026":   "bonus psicologo",
	"carta dedicata a te":    "carta dedicata a te",
	"social card":            "carta dedicata a te",
	"carta cultura giovani":  "carta della cultura / merito",
	"carta merito":           "carta della cultura / merito",
	"18app":                  "carta della cultura / merito",
	"reddito di inclusione":  "assegno di inclusione (adi)",
	"adi":                    "assegno di inclusione (adi)",
	"assegno di inclusione":  "assegno di inclusione (adi)",
	"sfl":                    "supporto formazione e lavoro",
	"bonus acqua":            "bonus acqua potabile",
	"bonus colonnine":        "bonus colonnine ricarica elettrica",
	"bonus tv":               "bonus tv / decoder",
	"bonus decoder":          "bonus tv / decoder",
}

// sourceEvidence tracks data from a single source for cross-validation.
type sourceEvidence struct {
	SourceName string
	Trust      float64
	Importo    string
	Scadenza   string
}

// EnrichBonusData merges scraped bonus data with the hardcoded bonus list.
// Hardcoded bonuses serve as the authoritative base; scraped data supplements them.
// Now includes fuzzy matching, multi-source tracking, and cross-validation.
func EnrichBonusData(scraped []models.Bonus, hardcoded []models.Bonus) []models.Bonus {
	// 1. Start with hardcoded as base (they have complete data + proper IDs for scoring)
	result := make([]models.Bonus, len(hardcoded))
	copy(result, hardcoded)

	// 2. Set UltimoAggiornamento on hardcoded if not set
	now := time.Now().Format("2 January 2006")
	for i := range result {
		if result[i].UltimoAggiornamento == "" {
			result[i].UltimoAggiornamento = now
		}
		if result[i].Stato == "" {
			result[i].Stato = "attivo"
		}
	}

	// 3. Build lookup by normalized name
	existing := make(map[string]int) // normalized name -> index in result
	for i, b := range result {
		existing[normalizeName(b.Nome)] = i
	}

	// 4. Track source evidence per bonus
	evidence := make(map[int][]sourceEvidence) // result index -> evidence list

	// Add hardcoded as initial evidence
	for i, b := range result {
		trust := 0.9 // hardcoded data is high trust
		if b.FonteURL != "" {
			trust = GetTrust(b.FonteURL)
		}
		evidence[i] = []sourceEvidence{{
			SourceName: "hardcoded",
			Trust:      trust,
			Importo:    b.Importo,
			Scadenza:   b.Scadenza,
		}}
	}

	// 5. Merge scraped data with fuzzy matching
	for _, s := range scraped {
		idx := findMatch(s.Nome, existing)
		srcTrust := 0.5
		if s.FonteURL != "" {
			srcTrust = GetTrust(s.FonteURL)
		}

		if idx >= 0 {
			// Match found — merge and track evidence
			mergeBonus(&result[idx], &s, srcTrust)
			evidence[idx] = append(evidence[idx], sourceEvidence{
				SourceName: s.FonteNome,
				Trust:      srcTrust,
				Importo:    s.Importo,
				Scadenza:   s.Scadenza,
			})
		} else {
			// New bonus from scraper
			s.UltimoAggiornamento = now
			if s.Stato == "" {
				s.Stato = "attivo"
			}
			if s.Categoria == "" {
				s.Categoria = "altro"
			}
			result = append(result, s)
			newIdx := len(result) - 1
			existing[normalizeName(s.Nome)] = newIdx
			evidence[newIdx] = []sourceEvidence{{
				SourceName: s.FonteNome,
				Trust:      srcTrust,
				Importo:    s.Importo,
				Scadenza:   s.Scadenza,
			}}
		}
	}

	// 6. Compute confidence scores and cross-validation
	for i := range result {
		ev := evidence[i]
		if len(ev) == 0 {
			continue
		}

		result[i].SourcesCount = len(ev)
		result[i].Corroborated = len(ev) >= 2

		// Cross-validate and detect conflicts
		conflicts := crossValidate(&result[i], ev)
		if len(conflicts) > 0 {
			result[i].ConflictFields = conflicts
		}

		// Compute confidence score
		result[i].ConfidenceScore = computeConfidence(ev, result[i].Corroborated, len(conflicts) > 0)
	}

	// 7. Validate
	var valid []models.Bonus
	for _, b := range result {
		if b.Nome != "" && b.ID != "" {
			valid = append(valid, b)
		}
	}

	logger.Info("[enricher] Result", map[string]interface{}{
		"hardcoded": len(hardcoded), "scraped": len(scraped), "unique": len(valid),
	})
	return valid
}

// findMatch looks up a bonus name using exact, alias, and fuzzy matching.
// Returns the index in the result slice, or -1 if no match.
func findMatch(name string, existing map[string]int) int {
	norm := normalizeName(name)

	// 1. Exact normalized match
	if idx, ok := existing[norm]; ok {
		return idx
	}

	// 2. Alias lookup
	if canonical, ok := bonusAliases[norm]; ok {
		if idx, ok := existing[canonical]; ok {
			return idx
		}
	}

	// 3. Fuzzy match (Levenshtein distance/maxLen <= 0.2)
	bestIdx := -1
	bestRatio := 1.0
	for existingName, idx := range existing {
		maxLen := len(norm)
		if len(existingName) > maxLen {
			maxLen = len(existingName)
		}
		if maxLen == 0 {
			continue
		}
		dist := levenshtein(norm, existingName)
		ratio := float64(dist) / float64(maxLen)
		if ratio <= 0.2 && ratio < bestRatio {
			bestRatio = ratio
			bestIdx = idx
		}
	}

	return bestIdx
}

// levenshtein computes the Levenshtein edit distance between two strings.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min3(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}

	return prev[lb]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func normalizeName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func mergeBonus(dst *models.Bonus, src *models.Bonus, srcTrust float64) {
	// Trust-based merge: higher trust sources can override
	if src.Importo != "" && src.Importo != "Vedi sito ufficiale" {
		if dst.Importo == "" || srcTrust >= 0.9 {
			dst.Importo = src.Importo
		}
	}
	if src.Scadenza != "" && src.Scadenza != "Verificare sul sito ufficiale" {
		if dst.Scadenza == "" || srcTrust >= 0.9 {
			dst.Scadenza = src.Scadenza
		}
	}
	if src.FonteURL != "" && dst.FonteURL == "" {
		dst.FonteURL = src.FonteURL
	}
	if src.UltimoAggiornamento != "" {
		dst.UltimoAggiornamento = src.UltimoAggiornamento
	}
}

// crossValidate checks for conflicts between source evidence.
// Returns a list of conflicting field names.
func crossValidate(bonus *models.Bonus, ev []sourceEvidence) []string {
	if len(ev) < 2 {
		return nil
	}

	var conflicts []string

	// Check importo conflicts
	importoValues := make(map[string]float64) // value -> max trust
	for _, e := range ev {
		if e.Importo == "" {
			continue
		}
		norm := normalizeName(e.Importo)
		if existing, ok := importoValues[norm]; ok {
			if e.Trust > existing {
				importoValues[norm] = e.Trust
			}
		} else {
			importoValues[norm] = e.Trust
		}
	}
	if len(importoValues) > 1 {
		conflicts = append(conflicts, "importo")
		// Apply cross-validation rules: highest trust wins
		applyTrustResolution(bonus, ev, "importo")
	}

	// Check scadenza conflicts
	scadenzaValues := make(map[string]float64)
	for _, e := range ev {
		if e.Scadenza == "" {
			continue
		}
		norm := normalizeName(e.Scadenza)
		if existing, ok := scadenzaValues[norm]; ok {
			if e.Trust > existing {
				scadenzaValues[norm] = e.Trust
			}
		} else {
			scadenzaValues[norm] = e.Trust
		}
	}
	if len(scadenzaValues) > 1 {
		conflicts = append(conflicts, "scadenza")
		applyTrustResolution(bonus, ev, "scadenza")
	}

	return conflicts
}

// applyTrustResolution resolves conflicts by preferring the highest-trust source value.
func applyTrustResolution(bonus *models.Bonus, ev []sourceEvidence, field string) {
	var bestValue string
	var bestTrust float64

	for _, e := range ev {
		var val string
		switch field {
		case "importo":
			val = e.Importo
		case "scadenza":
			val = e.Scadenza
		}
		if val == "" {
			continue
		}
		if e.Trust > bestTrust {
			bestTrust = e.Trust
			bestValue = val
		}
	}

	if bestValue == "" {
		return
	}

	switch field {
	case "importo":
		if bestTrust >= 0.9 {
			bonus.Importo = bestValue
		}
	case "scadenza":
		if bestTrust >= 0.9 {
			bonus.Scadenza = bestValue
		}
	}
}

// computeConfidence calculates confidence score from evidence.
func computeConfidence(ev []sourceEvidence, corroborated bool, hasConflicts bool) float64 {
	if len(ev) == 0 {
		return 0
	}

	// Weighted average of trust scores
	totalWeight := 0.0
	weightedSum := 0.0
	for _, e := range ev {
		weightedSum += e.Trust
		totalWeight++
	}
	avgTrust := weightedSum / totalWeight

	// Apply factor based on corroboration/conflict status
	var factor float64
	switch {
	case hasConflicts:
		factor = 0.4
	case corroborated:
		factor = 1.0
	default:
		factor = 0.7
	}

	score := avgTrust * factor
	// Clamp to [0, 1]
	if score > 1.0 {
		score = 1.0
	}
	if score < 0 {
		score = 0
	}
	return score
}
