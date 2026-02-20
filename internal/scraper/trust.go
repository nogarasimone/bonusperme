package scraper

import (
	"net/url"
	"strings"
)

// SourceTrust holds trust metadata for a domain.
type SourceTrust struct {
	TrustScore    float64
	Authoritative bool
}

// trustMap maps domains to their trust scores.
var trustMap = map[string]SourceTrust{
	"inps.it":                  {TrustScore: 1.0, Authoritative: true},
	"agenziaentrate.gov.it":    {TrustScore: 1.0, Authoritative: true},
	"mef.gov.it":               {TrustScore: 1.0, Authoritative: true},
	"gazzettaufficiale.it":     {TrustScore: 1.0, Authoritative: true},
	"normattiva.it":            {TrustScore: 0.95, Authoritative: true},
	"fiscooggi.it":             {TrustScore: 0.8, Authoritative: false},
	"ticonsiglio.com":          {TrustScore: 0.6, Authoritative: false},
	"fiscoetasse.com":          {TrustScore: 0.6, Authoritative: false},
	"money.it":                 {TrustScore: 0.55, Authoritative: false},
	"bonusx.it":                {TrustScore: 0.5, Authoritative: false},
	"brocardi.it":              {TrustScore: 0.85, Authoritative: false},
	"corriere.it":              {TrustScore: 0.7, Authoritative: false},
	"ilsole24ore.com":          {TrustScore: 0.75, Authoritative: false},
	"lavoro.gov.it":            {TrustScore: 0.95, Authoritative: true},
	"mimit.gov.it":             {TrustScore: 1.0, Authoritative: true},
	"mase.gov.it":              {TrustScore: 1.0, Authoritative: true},
	"mur.gov.it":               {TrustScore: 1.0, Authoritative: true},
	"consap.it":                {TrustScore: 0.95, Authoritative: true},
	"cartacultura.gov.it":      {TrustScore: 1.0, Authoritative: true},
	"arera.it":                 {TrustScore: 1.0, Authoritative: true},
	"poste.it":                 {TrustScore: 0.85, Authoritative: true},
	"masaf.gov.it":             {TrustScore: 1.0, Authoritative: true},
}

// GetTrust returns the trust score for a URL based on its domain.
// Returns 0.5 as default for unknown domains.
func GetTrust(rawURL string) float64 {
	u, err := url.Parse(rawURL)
	if err != nil {
		return 0.5
	}
	host := strings.ToLower(u.Hostname())

	// Try exact match first, then check if domain ends with a trusted suffix
	for domain, trust := range trustMap {
		if host == domain || host == "www."+domain || strings.HasSuffix(host, "."+domain) {
			return trust.TrustScore
		}
	}
	return 0.5
}

// IsAuthoritative returns whether the source domain is an authoritative institution.
func IsAuthoritative(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())

	for domain, trust := range trustMap {
		if host == domain || host == "www."+domain || strings.HasSuffix(host, "."+domain) {
			return trust.Authoritative
		}
	}
	return false
}
