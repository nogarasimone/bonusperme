package scraper

// Source defines a web source to scrape for bonus information.
type Source struct {
	URL      string
	Name     string
	Type     string  // "inps", "ade", "mef", "editorial", "caf"
	Priority int     // 1=primary, 2=secondary, 3=backup
	Parser   string  // "inps", "ade", "editorial", "generic"
	Trust    float64 // 0.0-1.0, derived from domain trust map
}

// GetSources returns the list of all sources to scrape.
func GetSources() []Source {
	sources := []Source{
		// Institutional (trust >= 0.9)
		{URL: "https://www.inps.it/it/it/sostegni-sussidi-indennita/per-genitori.html", Name: "INPS Genitori", Type: "inps", Priority: 1, Parser: "inps"},
		{URL: "https://www.inps.it/it/it/sostegni-sussidi-indennita/per-famiglie.html", Name: "INPS Famiglie", Type: "inps", Priority: 1, Parser: "inps"},
		{URL: "https://www.agenziaentrate.gov.it/portale/web/guest/aree-tematiche/casa/agevolazioni", Name: "AdE Casa", Type: "ade", Priority: 1, Parser: "ade"},
		{URL: "https://www.agenziaentrate.gov.it/portale/web/guest/agevolazioni", Name: "AdE Agevolazioni", Type: "ade", Priority: 1, Parser: "ade"},
		{URL: "https://www.mef.gov.it/focus/", Name: "MEF Focus", Type: "mef", Priority: 1, Parser: "generic"},
		{URL: "https://www.mef.gov.it/focus/Legge-di-Bilancio-2026/", Name: "MEF LdB 2026", Type: "mef", Priority: 1, Parser: "generic"},
		// Editorial (trust 0.5-0.8)
		{URL: "https://www.ticonsiglio.com/bonus-2026/", Name: "Ti Consiglio", Type: "editorial", Priority: 2, Parser: "editorial"},
		{URL: "https://www.fiscoetasse.com/new-rassegna-stampa/1542-legge-di-bilancio-2026-le-misure-per-le-famiglie.html", Name: "Fisco e Tasse", Type: "editorial", Priority: 2, Parser: "editorial"},
		{URL: "https://www.fiscooggi.it/", Name: "FiscoOggi", Type: "editorial", Priority: 2, Parser: "editorial"},
		{URL: "https://www.money.it/bonus-agevolazioni", Name: "Money.it", Type: "editorial", Priority: 3, Parser: "editorial"},
		{URL: "https://www.bonusx.it/", Name: "BonusX", Type: "editorial", Priority: 3, Parser: "editorial"},
		{URL: "https://www.brocardi.it/notizie-giuridiche/", Name: "Brocardi", Type: "editorial", Priority: 2, Parser: "editorial"},
		{URL: "https://www.corriere.it/economia/", Name: "Corriere Economia", Type: "editorial", Priority: 2, Parser: "editorial"},
		{URL: "https://www.ilsole24ore.com/sez/norme-e-tributi", Name: "Sole 24 Ore Norme & Tributi", Type: "editorial", Priority: 2, Parser: "editorial"},
		// Bonus Bollette — ARERA
		{URL: "https://www.arera.it/consumatori/bonus-sociale", Name: "ARERA Bonus Sociale", Type: "arera", Priority: 1, Parser: "generic"},
		// Carta Acquisti — MEF
		{URL: "https://www.mef.gov.it/focus/Carta-Acquisti/", Name: "MEF Carta Acquisti", Type: "mef", Priority: 1, Parser: "generic"},
		// Poste Italiane — erogatore carte
		{URL: "https://www.poste.it/carta-acquisti", Name: "Poste Carta Acquisti", Type: "poste", Priority: 2, Parser: "generic"},
		{URL: "https://www.poste.it/carta-dedicata-a-te", Name: "Poste Carta Dedicata", Type: "poste", Priority: 2, Parser: "generic"},
	}

	// Auto-populate Trust from domain trust map
	for i := range sources {
		sources[i].Trust = GetTrust(sources[i].URL)
	}

	return sources
}
