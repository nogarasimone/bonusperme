package pipeline

import (
	"regexp"
	"strings"
)

// normRefAliases maps common legislative type names to their canonical abbreviation.
var normRefAliases = map[string]string{
	"decreto-legge":       "DL",
	"decreto legge":       "DL",
	"d.l.":                "DL",
	"dl":                  "DL",
	"legge":               "L",
	"l.":                  "L",
	"decreto legislativo": "DLgs",
	"d.lgs.":              "DLgs",
	"d.lgs":               "DLgs",
	"dlgs":                "DLgs",
	"dpcm":                "DPCM",
	"d.p.c.m.":            "DPCM",
	"dpr":                 "DPR",
	"d.p.r.":              "DPR",
	"dm":                  "DM",
	"d.m.":                "DM",
	"decreto ministeriale":  "DM",
	"decreto direttoriale":  "DD",
	"dd":                    "DD",
}

// Regex patterns for extracting norm references.
var (
	// Matches patterns like "D.Lgs. 29 dicembre 2021, n. 230" or "DL 48/2023"
	normRefFullRe = regexp.MustCompile(
		`(?i)(D\.?L(?:gs)?\.?|decreto[- ]?legge|decreto legislativo|legge|DPCM|D\.?P\.?C\.?M\.?|DPR|D\.?P\.?R\.?|DM|D\.?M\.?|decreto ministeriale|decreto direttoriale|DD)` +
			`[\s,.]*` +
			`(?:\d{1,2}\s+(?:gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)\s+)?` +
			`(\d{4})?` +
			`[\s,.]*` +
			`(?:n\.?\s*)?(\d+)` +
			`(?:[/\s-](\d{4}))?`,
	)

)

// NormalizeNormRef normalizes a raw legislative reference to canonical form.
// Example: "D.Lgs. 29 dicembre 2021, n. 230" → "DLgs 230/2021"
func NormalizeNormRef(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	m := normRefFullRe.FindStringSubmatch(raw)
	if m == nil {
		return raw
	}

	typeStr := strings.ToLower(strings.TrimRight(m[1], ". "))
	canonical, ok := normRefAliases[typeStr]
	if !ok {
		// Try without trailing dots
		cleaned := strings.ReplaceAll(typeStr, ".", "")
		cleaned = strings.ReplaceAll(cleaned, " ", "")
		canonical, ok = normRefAliases[cleaned]
		if !ok {
			canonical = m[1]
		}
	}

	number := m[3]
	year := m[4]
	if year == "" {
		year = m[2]
	}

	if year != "" && number != "" {
		return canonical + " " + number + "/" + year
	}
	if number != "" {
		return canonical + " " + number
	}
	return raw
}

// ExtractNormRefs finds all norm references in free text and returns them normalized.
func ExtractNormRefs(text string) []string {
	matches := normRefFullRe.FindAllString(text, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	var result []string
	for _, m := range matches {
		norm := NormalizeNormRef(m)
		if norm != "" && !seen[norm] {
			seen[norm] = true
			result = append(result, norm)
		}
	}
	return result
}

// MatchNormRef checks if a GU title matches any of the bonus's norm references.
// Returns whether a match was found and which ref matched.
func MatchNormRef(guTitle string, bonusRefs []string) (bool, string) {
	guNorms := ExtractNormRefs(guTitle)
	if len(guNorms) == 0 {
		return false, ""
	}

	for _, guNorm := range guNorms {
		for _, bonusRef := range bonusRefs {
			if guNorm == bonusRef {
				return true, guNorm
			}
			// Partial match: same type and number (ignore year differences for amendments)
			if partialNormMatch(guNorm, bonusRef) {
				return true, guNorm
			}
		}
	}
	return false, ""
}

// partialNormMatch checks if two norm refs share the same type and number.
func partialNormMatch(a, b string) bool {
	aParts := strings.Fields(a)
	bParts := strings.Fields(b)
	if len(aParts) < 2 || len(bParts) < 2 {
		return false
	}
	// Same type
	if !strings.EqualFold(aParts[0], bParts[0]) {
		return false
	}
	// Extract number before /
	aNum := strings.Split(aParts[1], "/")[0]
	bNum := strings.Split(bParts[1], "/")[0]
	return aNum == bNum
}
