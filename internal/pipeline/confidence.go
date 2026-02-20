package pipeline

import (
	"bonusperme/internal/models"
	"time"
)

// CalculateConfidence computes a [0,1] confidence score for a bonus based on
// pipeline verification status and data completeness.
func CalculateConfidence(bonus *models.Bonus) float64 {
	score := 0.0

	// Base score: hardcoded data quality (0.15 – 0.40)
	if bonus.Importo != "" {
		score += 0.10
	}
	if len(bonus.Requisiti) > 0 {
		score += 0.05
	}
	if bonus.LinkUfficiale != "" {
		score += 0.05
	}
	if len(bonus.RiferimentiNormativi) > 0 {
		score += 0.10
	}
	if bonus.FonteURL != "" {
		score += 0.05
	}
	if bonus.Ente != "" {
		score += 0.05
	}

	// L1: GU verification (+0.20)
	if bonus.UltimaVerificaGU != nil {
		score += 0.20
	}

	// L2: Normattiva confirmed importo (+0.15)
	if bonus.ImportoConfermato != "" {
		score += 0.15
	}

	// L3: RSS corroboration (+0.05 per source, max +0.15)
	corrobBonus := float64(bonus.FontiCorroborate) * 0.05
	if corrobBonus > 0.15 {
		corrobBonus = 0.15
	}
	score += corrobBonus

	// L4: Link verified (+0.05)
	if bonus.LinkVerificato {
		score += 0.05
	}

	// Penalties
	now := time.Now()

	// Staleness penalty
	if !bonus.UltimaVerifica.IsZero() {
		age := now.Sub(bonus.UltimaVerifica)
		switch {
		case age > 120*24*time.Hour:
			score -= 0.20
		case age > 60*24*time.Hour:
			score -= 0.10
		}
	}

	// Year confirmation penalty
	if bonus.AnnoConferma > 0 && bonus.AnnoConferma < now.Year() {
		score -= 0.15
	}

	// Clamp to [0, 1]
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}
