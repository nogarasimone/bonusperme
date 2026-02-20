package datasource

import (
	"strings"

	"golang.org/x/net/html"
)

var bonusKeywords = []string{"bonus", "assegno", "detrazione", "agevolazione", "contributo", "carta", "esonero", "incentivo"}

func containsBonusKeyword(textLower string) bool {
	for _, kw := range bonusKeywords {
		if strings.Contains(textLower, kw) {
			return true
		}
	}
	return false
}

func categorize(text string) string {
	switch {
	case strings.Contains(text, "famiglia") || strings.Contains(text, "figlio") || strings.Contains(text, "nido") || strings.Contains(text, "nascita") || strings.Contains(text, "mamma"):
		return "famiglia"
	case strings.Contains(text, "casa") || strings.Contains(text, "ristruttur") || strings.Contains(text, "affitto") || strings.Contains(text, "abitazione"):
		return "casa"
	case strings.Contains(text, "salute") || strings.Contains(text, "psicolog"):
		return "salute"
	case strings.Contains(text, "studio") || strings.Contains(text, "cultura") || strings.Contains(text, "istruzione"):
		return "istruzione"
	case strings.Contains(text, "spesa") || strings.Contains(text, "alimentar"):
		return "spesa"
	case strings.Contains(text, "lavoro") || strings.Contains(text, "formazione"):
		return "lavoro"
	default:
		return "altro"
	}
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, s)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	if len(s) > 60 {
		s = s[:60]
	}
	return s
}

func getTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(getTextContent(c))
	}
	return sb.String()
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}
