package matcher

import (
	"bonusperme/internal/models"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var italianMonthsMap = map[string]time.Month{
	"gennaio": time.January, "febbraio": time.February, "marzo": time.March,
	"aprile": time.April, "maggio": time.May, "giugno": time.June,
	"luglio": time.July, "agosto": time.August, "settembre": time.September,
	"ottobre": time.October, "novembre": time.November, "dicembre": time.December,
}

var itDateRe = regexp.MustCompile(`(\d{1,2})\s+(gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)\s+(\d{4})`)
var yearOnlyRe = regexp.MustCompile(`\b(20\d{2})\b`)

// formatEuro formats a float as "€1.234" with dot as thousands separator, no decimals.
func formatEuro(v float64) string {
	n := int64(v)
	if n < 0 {
		return "-" + formatEuro(-v)
	}
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return "€" + s
	}
	rem := len(s) % 3
	if rem == 0 {
		rem = 3
	}
	var b strings.Builder
	b.WriteString("€")
	b.WriteString(s[:rem])
	for i := rem; i < len(s); i += 3 {
		b.WriteByte('.')
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// isScaduto determines whether a bonus deadline has passed.
func isScaduto(scadenza string) bool {
	if scadenza == "" {
		return false
	}
	lower := strings.ToLower(strings.TrimSpace(scadenza))

	// Never-expiring patterns
	neverExpired := []string{"in vigore", "permanente", "annuale", "esaurimento fondi", "erogazione automatica", "entro 60 giorni"}
	for _, pat := range neverExpired {
		if strings.Contains(lower, pat) {
			return false
		}
	}
	// "Bando regionale" / "Bando annuale" patterns
	if strings.Contains(lower, "bando") {
		return false
	}
	// "Domanda entro il 28 febbraio per arretrati" — recurring deadline, not expired
	if strings.Contains(lower, "per arretrati") {
		return false
	}

	now := time.Now()

	// Try Italian date pattern: "31 dicembre 2025"
	if m := itDateRe.FindStringSubmatch(lower); len(m) == 4 {
		day, _ := strconv.Atoi(m[1])
		month := italianMonthsMap[m[2]]
		year, _ := strconv.Atoi(m[3])
		deadline := time.Date(year, month, day, 23, 59, 59, 0, time.UTC)
		return now.After(deadline)
	}

	// Try dd/mm/yyyy pattern
	slashRe := regexp.MustCompile(`(\d{2})/(\d{2})/(\d{4})`)
	if m := slashRe.FindStringSubmatch(scadenza); len(m) == 4 {
		day, _ := strconv.Atoi(m[1])
		month, _ := strconv.Atoi(m[2])
		year, _ := strconv.Atoi(m[3])
		deadline := time.Date(year, time.Month(month), day, 23, 59, 59, 0, time.UTC)
		return now.After(deadline)
	}

	// If just a past year is mentioned (e.g. "2024"), consider expired
	if m := yearOnlyRe.FindStringSubmatch(scadenza); len(m) == 2 {
		year, _ := strconv.Atoi(m[1])
		if year < now.Year() {
			return true
		}
	}

	return false
}

// LINK STABILITY STRATEGY (v2 — 19 febbraio 2026)
// =============================================
// Italian institutional sites frequently reorganize URLs. To prevent link rot:
//
// LinkUfficiale → General thematic section (not deep link to specific bonus page)
//   e.g. INPS → /schede/prestazioni-e-servizi.html (stable infrastructure)
//   e.g. AdE  → /aree-tematiche/casa/agevolazioni (stable section)
//
// LinkRicerca → Search endpoint with keywords (search URLs never change)
//   e.g. INPS → /ricerca.html?q=assegno+unico+universale+figli
//   e.g. AdE  → /ricerca?keywords=bonus+mobili+elettrodomestici
//
// FonteURL → Same as LinkUfficiale (deep links unreliable for citations)
//
// The pipeline L4 link checker will detect when specific pages move.
// Users can navigate from the general section or use the search link.

func GetAllBonus() []models.Bonus {
	bonuses := []models.Bonus{

		// ═══════════════════════════════════════════════════════
		// FAMIGLIA
		// ═══════════════════════════════════════════════════════
		{
			ID: "assegno-unico", Nome: "Assegno Unico Universale", Categoria: "famiglia",
			Descrizione: "Assegno mensile per ogni figlio a carico fino a 21 anni. Importo da €57 a €199,4/mese per figlio in base all'ISEE, con maggiorazioni per famiglie numerose e figli piccoli.",
			Importo: "da €57 a €199,4/mese per figlio", Scadenza: "Domanda entro il 28 febbraio per arretrati",
			Requisiti:       []string{"Figli a carico sotto i 21 anni", "Residenza in Italia", "ISEE valido (facoltativo)"},
			ComeRichiederlo: []string{"Portale INPS con SPID/CIE", "Sezione 'Assegno Unico'", "Compilare domanda online"},
			Documenti:       []string{"SPID o CIE", "ISEE in corso di validità", "Codici fiscali di tutti i figli", "Coordinate bancarie/postali (IBAN)"},
			FAQ: []models.FAQ{
				{Domanda: "Posso richiederlo se sono separato/a?", Risposta: "Sì, l'assegno spetta al genitore che ha i figli a carico. In caso di affido condiviso, può essere diviso al 50%."},
				{Domanda: "Serve il commercialista?", Risposta: "No, la domanda si fa online sul portale INPS con SPID o CIE. In alternativa puoi rivolgerti a un patronato gratuitamente."},
				{Domanda: "Quanto tempo ci vuole per ricevere i soldi?", Risposta: "Generalmente 30-60 giorni dalla domanda. Il pagamento avviene mensilmente tramite bonifico."},
			},
			LinkUfficiale:        "https://www.inps.it/it/it/dettaglio-scheda.it.schede-servizio-strumento.schede-servizi.assegno-unico-e-universale-per-i-figli-a-carico-55984.assegno-unico-e-universale-per-i-figli-a-carico.html",
			LinkRicerca:          "https://www.inps.it/it/it/dettaglio-scheda.it.schede-servizio-strumento.schede-servizi.assegno-unico-e-universale-per-i-figli-a-carico-55984.assegno-unico-e-universale-per-i-figli-a-carico.html",
			Ente:                 "INPS",
			FonteURL:             "https://www.inps.it/it/it/dettaglio-scheda.it.schede-servizio-strumento.schede-servizi.assegno-unico-e-universale-per-i-figli-a-carico-55984.assegno-unico-e-universale-per-i-figli-a-carico.html",
			FonteNome:            "INPS",
			RiferimentiNormativi: []string{"D.Lgs. 29 dicembre 2021, n. 230", "Circolare INPS n. 33 del 4 febbraio 2025 — Aggiornamento importi"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "attivo",
		},
		{
			ID: "bonus-nido", Nome: "Bonus Asilo Nido", Categoria: "famiglia",
			Descrizione: "Contributo per rette asilo nido pubblico/privato o supporto domiciliare per bimbi sotto 3 anni con patologie croniche.",
			Importo: "fino a €3.600/anno (ISEE ≤ €25.000)", Scadenza: "31 dicembre 2026",
			Requisiti:       []string{"Figli sotto i 3 anni", "Iscrizione asilo nido", "ISEE in corso di validità"},
			ComeRichiederlo: []string{"Portale INPS con SPID/CIE", "Sezione 'Bonus Nido'", "Allegare ricevute rette + ISEE"},
			Documenti:       []string{"SPID o CIE", "ISEE in corso di validità", "Ricevute di pagamento rette asilo", "Iscrizione/frequenza del minore"},
			FAQ: []models.FAQ{
				{Domanda: "Vale anche per asili nido privati?", Risposta: "Sì, il bonus copre sia asili nido pubblici che privati autorizzati, con importi diversi in base all'ISEE."},
				{Domanda: "Posso cumularlo con l'Assegno Unico?", Risposta: "Sì, bonus nido e Assegno Unico sono pienamente cumulabili."},
			},
			LinkUfficiale:        "https://www.inps.it/it/it/dettaglio-scheda.it.schede-servizio-strumento.schede-servizi.bonus-asilo-nido-e-forme-di-supporto-presso-la-propria-abitazione-51105.bonus-asilo-nido-e-forme-di-supporto-presso-la-propria-abitazione.html",
			LinkRicerca:          "https://www.inps.it/it/it/dettaglio-scheda.it.schede-servizio-strumento.schede-servizi.bonus-asilo-nido-e-forme-di-supporto-presso-la-propria-abitazione-51105.bonus-asilo-nido-e-forme-di-supporto-presso-la-propria-abitazione.html",
			Ente:                 "INPS",
			FonteURL:             "https://www.inps.it/it/it/dettaglio-scheda.it.schede-servizio-strumento.schede-servizi.bonus-asilo-nido-e-forme-di-supporto-presso-la-propria-abitazione-51105.bonus-asilo-nido-e-forme-di-supporto-presso-la-propria-abitazione.html",
			FonteNome:            "INPS",
			RiferimentiNormativi: []string{"Legge di Bilancio 2025, art. 1 comma 177", "Circolare INPS n. 27/2025", "Legge di Bilancio 2026 (L. 198/2025) — Conferma"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "attivo",
		},
		{
			ID: "bonus-nascita", Nome: "Bonus nuovi nati", Categoria: "famiglia",
			Descrizione: "Contributo una tantum di €1.000 per ogni figlio nato o adottato dal 2025, confermato per il 2026, per nuclei con ISEE fino a €40.000.",
			Importo: "€1.000 una tantum", Scadenza: "Entro 60 giorni dalla nascita",
			Requisiti:       []string{"Figlio nato/adottato dal 2025", "ISEE fino a €40.000", "Residenza in Italia"},
			ComeRichiederlo: []string{"Portale INPS con SPID/CIE", "Sezione 'Carta nuovi nati'", "Domanda online entro 60 giorni"},
			Documenti:       []string{"SPID o CIE", "ISEE in corso di validità", "Certificato di nascita o adozione", "Coordinate bancarie (IBAN)"},
			FAQ: []models.FAQ{
				{Domanda: "Vale per adozioni internazionali?", Risposta: "Sì, il bonus spetta anche per adozioni nazionali e internazionali perfezionate dal 2025."},
				{Domanda: "Entro quando devo fare domanda?", Risposta: "La domanda va presentata entro 60 giorni dalla nascita o dall'ingresso in famiglia del minore adottato."},
			},
			LinkUfficiale:        "https://www.inps.it/it/it/dettaglio-scheda.it.schede-servizio-strumento.schede-servizi.bonus-nuovi-nati.html",
			LinkRicerca:          "https://www.inps.it/it/it/dettaglio-scheda.it.schede-servizio-strumento.schede-servizi.bonus-nuovi-nati.html",
			Ente:                 "INPS",
			FonteURL:             "https://www.inps.it/it/it/dettaglio-scheda.it.schede-servizio-strumento.schede-servizi.bonus-nuovi-nati.html",
			FonteNome:            "INPS",
			RiferimentiNormativi: []string{"Legge di Bilancio 2025, art. 1 commi 206-208", "Legge di Bilancio 2026 (L. 198/2025) — Conferma e rifinanziamento"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "attivo",
		},
		{
			ID: "bonus-mamma", Nome: "Nuovo Bonus mamme", Categoria: "famiglia",
			Descrizione: "Contributo di €60/mese (€720/anno) per madri lavoratrici con almeno 2 figli, erogato in unica soluzione a dicembre 2026. Vale per dipendenti e autonome con reddito fino a €40.000. Per madri con 3+ figli e contratto a tempo indeterminato resta l'esonero contributivo IVS (fino a €3.000/anno) fino al 31/12/2026.",
			Importo: "€60/mese (€720/anno) oppure esonero IVS fino a €3.000/anno", Scadenza: "31 dicembre 2026",
			Requisiti: []string{
				"Madre lavoratrice dipendente o autonoma",
				"Almeno 2 figli a carico",
				"Figlio più piccolo sotto i 10 anni (sotto i 18 se 3+ figli)",
				"Reddito da lavoro ≤ €40.000/anno",
			},
			ComeRichiederlo: []string{
				"Bonus €60/mese: domanda INPS online con SPID/CIE",
				"Esonero IVS (3+ figli, tempo indeterminato): comunicare al datore di lavoro i CF dei figli",
			},
			Documenti: []string{"SPID o CIE", "Codici fiscali dei figli", "ISEE in corso di validità (per bonus €60)", "Comunicazione al datore di lavoro (per esonero IVS)"},
			FAQ: []models.FAQ{
				{Domanda: "Vale per le lavoratrici autonome?", Risposta: "Sì, dal 2026 il bonus €60/mese è esteso anche a lavoratrici autonome con partita IVA iscritte a gestioni INPS o casse professionali."},
				{Domanda: "Bonus e esonero contributivo sono cumulabili?", Risposta: "No, sono alternativi. Se hai 3+ figli e contratto a tempo indeterminato, hai diritto solo all'esonero IVS (più vantaggioso). Se hai 2 figli o lavoro a termine/autonomo, hai diritto al bonus €60/mese."},
				{Domanda: "Quando ricevo i soldi?", Risposta: "Le mensilità da gennaio a novembre 2026 vengono erogate in unica soluzione a dicembre 2026."},
			},
			LinkUfficiale:        "https://www.inps.it/it/it/dettaglio-scheda.it.schede-servizio-strumento.schede-servizi.nuovo-bonus-mamme.html",
			LinkRicerca:          "https://www.inps.it/it/it/dettaglio-scheda.it.schede-servizio-strumento.schede-servizi.nuovo-bonus-mamme.html",
			Ente:                 "INPS",
			FonteURL:             "https://www.inps.it/it/it/dettaglio-scheda.it.schede-servizio-strumento.schede-servizi.nuovo-bonus-mamme.html",
			FonteNome:            "INPS",
			RiferimentiNormativi: []string{"Legge di Bilancio 2024, art. 1 commi 180-182 (esonero IVS)", "DL 95/2025, art. 6 (bonus mamme)", "Legge di Bilancio 2026 (L. 198/2025) — Aumento a €60/mese"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "attivo",
		},

		// ═══════════════════════════════════════════════════════
		// CASA
		// ═══════════════════════════════════════════════════════
		{
			ID: "bonus-ristrutturazione", Nome: "Bonus Ristrutturazione", Categoria: "casa",
			Descrizione: "Detrazione IRPEF per spese di ristrutturazione edilizia fino a €96.000 per unità immobiliare. Aliquota 50% per abitazione principale, 36% per altri immobili (seconde case). Recupero in 10 rate annuali.",
			Importo: "detrazione 50% prima casa / 36% altre, fino a €96.000", Scadenza: "31 dicembre 2026",
			Requisiti:       []string{"Proprietario/titolare diritto reale", "Lavori manutenzione straordinaria", "Pagamento con bonifico parlante"},
			ComeRichiederlo: []string{"Pagare con bonifico parlante", "Conservare fatture", "Indicare in dichiarazione dei redditi"},
			Documenti:       []string{"Fatture e ricevute dei lavori", "Bonifici parlanti", "Titoli abilitativi (CILA/SCIA)", "Dati catastali dell'immobile"},
			FAQ: []models.FAQ{
				{Domanda: "Posso cedere il credito?", Risposta: "Dal 2025 la cessione del credito e lo sconto in fattura non sono più disponibili per le nuove pratiche, salvo eccezioni residuali."},
				{Domanda: "Devo fare la pratica prima di iniziare i lavori?", Risposta: "Per la manutenzione straordinaria serve la CILA prima dell'inizio lavori. Per la manutenzione ordinaria su parti condominiali basta la delibera assembleare."},
				{Domanda: "In quanti anni si recupera?", Risposta: "La detrazione si recupera in 10 rate annuali di pari importo nella dichiarazione dei redditi."},
			},
			LinkUfficiale:        "https://www.agenziaentrate.gov.it/portale/aree-tematiche/casa/agevolazioni/agevolazioni-per-le-ristrutturazioni-edilizie",
			LinkRicerca:          "https://www.agenziaentrate.gov.it/portale/aree-tematiche/casa/agevolazioni/agevolazioni-per-le-ristrutturazioni-edilizie",
			Ente:                 "Agenzia delle Entrate",
			FonteURL:             "https://www.agenziaentrate.gov.it/portale/aree-tematiche/casa/agevolazioni/agevolazioni-per-le-ristrutturazioni-edilizie",
			FonteNome:            "Agenzia delle Entrate",
			RiferimentiNormativi: []string{"Art. 16-bis DPR 917/1986 (TUIR)", "Legge di Bilancio 2026 (L. 198/2025) — Aliquote 50% prima casa, 36% altre"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "attivo",
		},
		{
			ID: "bonus-mobili", Nome: "Bonus Mobili ed Elettrodomestici", Categoria: "casa",
			Descrizione: "Detrazione 50% su acquisto mobili e grandi elettrodomestici per immobile in ristrutturazione, fino a €5.000. Nessuna distinzione tra prima e seconda casa: l'aliquota è 50% per tutti, purché legato a ristrutturazione.",
			Importo: "detrazione 50% fino a €5.000", Scadenza: "31 dicembre 2026",
			Requisiti:       []string{"Lavori di ristrutturazione avviati", "Elettrodomestici classe A+ (A per forni)", "Pagamento tracciabile"},
			ComeRichiederlo: []string{"Pagamenti tracciabili", "Conservare ricevute", "Indicare in dichiarazione dei redditi"},
			Documenti:       []string{"Fatture di acquisto mobili/elettrodomestici", "Ricevute bonifico o carta", "Documentazione ristrutturazione in corso"},
			FAQ: []models.FAQ{
				{Domanda: "Posso comprare mobili anche prima della fine dei lavori?", Risposta: "Sì, basta che la ristrutturazione sia iniziata. I mobili possono essere acquistati anche prima della conclusione dei lavori."},
				{Domanda: "Quali elettrodomestici sono inclusi?", Risposta: "Grandi elettrodomestici di classe energetica A+ (A per forni): frigoriferi, lavatrici, lavastoviglie, forni, condizionatori, ecc."},
			},
			LinkUfficiale:        "https://www.agenziaentrate.gov.it/portale/aree-tematiche/casa/agevolazioni/bonus-mobili-ed-elettrodomestici",
			LinkRicerca:          "https://www.agenziaentrate.gov.it/portale/aree-tematiche/casa/agevolazioni/bonus-mobili-ed-elettrodomestici",
			Ente:                 "Agenzia delle Entrate",
			FonteURL:             "https://www.agenziaentrate.gov.it/portale/aree-tematiche/casa/agevolazioni/bonus-mobili-ed-elettrodomestici",
			FonteNome:            "Agenzia delle Entrate",
			RiferimentiNormativi: []string{"Art. 16, comma 2, DL 63/2013", "Legge di Bilancio 2026 (L. 198/2025) — Conferma"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "attivo",
		},
		{
			ID: "bonus-affitto-giovani", Nome: "Bonus Affitto Giovani Under 31", Categoria: "casa",
			Descrizione: "Detrazione IRPEF pari al 20% del canone annuo per giovani tra 20 e 31 anni non compiuti che affittano un'abitazione principale diversa da quella dei genitori. Importo minimo garantito €991,60, massimo €2.000/anno, per i primi 4 anni di contratto.",
			Importo: "da €991,60 a €2.000/anno per 4 anni", Scadenza: "In vigore (misura strutturale)",
			Requisiti:       []string{"Età 20-31 anni non compiuti alla firma del contratto", "Reddito complessivo ≤ €15.493,71", "Contratto di locazione registrato", "Residenza nell'immobile, diversa da quella dei genitori"},
			ComeRichiederlo: []string{"Indicare in dichiarazione dei redditi (730 o Redditi PF)", "Quadro E — codice 4 detrazioni canoni di locazione", "Conservare contratto registrato e ricevute pagamento"},
			Documenti:       []string{"Contratto di locazione registrato", "Ricevuta di registrazione", "Certificato di residenza o autocertificazione", "Prove di pagamento canoni"},
			FAQ: []models.FAQ{
				{Domanda: "Vale se convivo con il mio partner?", Risposta: "Sì, purché il contratto sia intestato a te e l'immobile sia la tua abitazione principale, diversa da quella dei genitori."},
				{Domanda: "Posso usarlo se sono studente fuori sede?", Risposta: "Sì, ma attenzione: esiste anche la detrazione specifica per studenti fuori sede (19% su max €2.633). Le due detrazioni non sono cumulabili, scegli la più vantaggiosa."},
				{Domanda: "Se compio 31 anni durante il contratto?", Risposta: "La detrazione resta valida per i primi 4 anni dal contratto, anche se nel frattempo compi 31 anni."},
			},
			LinkUfficiale:        "https://www.agenziaentrate.gov.it/portale/aree-tematiche/casa/agevolazioni",
			LinkRicerca:          "https://www.agenziaentrate.gov.it/portale/ricerca?keywords=bonus+affitto+giovani+under+31",
			Ente:                 "Agenzia delle Entrate",
			FonteURL:             "https://www.agenziaentrate.gov.it/portale/aree-tematiche/casa/agevolazioni",
			FonteNome:            "Agenzia delle Entrate",
			RiferimentiNormativi: []string{"Art. 16, comma 1-quinquies, TUIR", "Decreto Sostegni-bis (DL 73/2021), art. 31"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "attivo",
		},
		{
			ID: "prima-casa-under36", Nome: "Fondo Garanzia Prima Casa Under 36", Categoria: "casa",
			Descrizione: "Fondo di garanzia statale (Consap) fino all'80% del mutuo per under 36 con ISEE ≤ €40.000. Le agevolazioni fiscali potenziate (esenzione imposte registro, ipotecaria, catastale e credito IVA) sono scadute il 31/12/2023 (transitorio fino al 31/12/2024 per chi aveva preliminare registrato entro il 31/12/2023). Resta attivo solo il fondo garanzia fino al 31/12/2027.",
			Importo: "garanzia statale 80% mutuo (esenzioni fiscali scadute)", Scadenza: "Fondo garanzia fino al 31 dicembre 2027",
			Requisiti:       []string{"Età < 36 anni al rogito", "ISEE ≤ €40.000", "Acquisto prima casa non di lusso", "Residenza nel comune entro 18 mesi"},
			ComeRichiederlo: []string{"Richiedere mutuo presso banca aderente al Fondo Consap", "Dichiarare requisiti nell'atto notarile", "ISEE aggiornato al momento del rogito"},
			Documenti:       []string{"ISEE in corso di validità", "Atto notarile di acquisto", "Documento d'identità", "Autocertificazione requisiti"},
			FAQ: []models.FAQ{
				{Domanda: "Le esenzioni fiscali sono ancora attive?", Risposta: "No, le esenzioni imposte (registro, ipotecaria, catastale) e il credito IVA sono scaduti il 31/12/2023. Il transitorio per chi aveva preliminare registrato entro il 31/12/2023 è terminato il 31/12/2024. Resta attivo solo il fondo di garanzia Consap per il mutuo."},
				{Domanda: "Posso comprare con il mio partner?", Risposta: "Sì, ma entrambi gli acquirenti devono avere meno di 36 anni e rispettare il limite ISEE per il fondo garanzia."},
			},
			LinkUfficiale:        "https://www.consap.it/fondo-prima-casa/",
			LinkRicerca:          "https://www.consap.it/fondo-prima-casa/",
			Ente:                 "Consap / MEF",
			FonteURL:             "https://www.consap.it/fondo-prima-casa/",
			FonteNome:            "Consap",
			RiferimentiNormativi: []string{"DL 73/2021, art. 64, commi 6-10", "Legge di Bilancio 2025 — Proroga fondo garanzia al 31/12/2027"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "attivo",
		},
		{
			ID: "ecobonus", Nome: "Ecobonus", Categoria: "casa",
			Descrizione: "Detrazione fiscale per interventi di efficientamento energetico: infissi, pompe di calore, cappotto termico, pannelli solari. Aliquota 50% per abitazione principale, 36% per altri immobili. Caldaie a combustibili fossili escluse dal 2025. Recupero in 10 rate annuali.",
			Importo: "detrazione 50% prima casa / 36% altre, massimali variabili per intervento", Scadenza: "31 dicembre 2026",
			Requisiti:       []string{"Immobile esistente con impianto di riscaldamento", "Interventi di efficientamento energetico", "Asseverazione tecnica", "Comunicazione ENEA entro 90 giorni"},
			ComeRichiederlo: []string{"Comunicazione ENEA entro 90 giorni da fine lavori", "Bonifico parlante", "Dichiarazione dei redditi"},
			Documenti:       []string{"Asseverazione tecnica", "APE pre e post intervento", "Fatture e bonifici parlanti", "Comunicazione ENEA"},
			FAQ: []models.FAQ{
				{Domanda: "Serve un tecnico per la pratica ENEA?", Risposta: "Sì, per la maggior parte degli interventi serve un tecnico abilitato per l'asseverazione e la comunicazione ENEA."},
				{Domanda: "Posso combinare ecobonus e bonus ristrutturazione?", Risposta: "No, per lo stesso intervento non puoi cumulare le due detrazioni. Devi scegliere quella più conveniente."},
				{Domanda: "Le caldaie a gas rientrano ancora?", Risposta: "No, dal 2025 gli interventi di sostituzione con caldaie a combustibili fossili sono esclusi dall'ecobonus, anche se dotate di valvole termostatiche."},
			},
			LinkUfficiale:        "https://ecobonus.mimit.gov.it/",
			LinkRicerca:          "https://ecobonus.mimit.gov.it/",
			Ente:                 "Ministero delle Imprese e del Made in Italy",
			FonteURL:             "https://ecobonus.mimit.gov.it/",
			FonteNome:            "Ministero delle Imprese e del Made in Italy",
			RiferimentiNormativi: []string{"Art. 14, DL 63/2013", "Legge di Bilancio 2026 (L. 198/2025) — Aliquote 50% prima casa, 36% altre"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "attivo",
		},
		{
			ID: "sismabonus", Nome: "Sismabonus", Categoria: "casa",
			Descrizione: "Detrazione IRPEF per interventi di messa in sicurezza antisismica su edifici esistenti in zone sismiche 1, 2 e 3. Aliquota 50% per abitazione principale, 36% per altri immobili, su un massimale di €96.000 per unità immobiliare. Recupero in 10 rate annuali. Include anche il 'Sismabonus Acquisti' per chi compra immobili in edifici demoliti e ricostruiti antisismicamente.",
			Importo: "detrazione 50% prima casa / 36% altre, fino a €96.000", Scadenza: "31 dicembre 2026",
			Requisiti:       []string{"Immobile in zona sismica 1, 2 o 3", "Interventi di consolidamento strutturale documentati", "Asseverazione tecnica da professionista abilitato", "Pagamento con bonifico parlante"},
			ComeRichiederlo: []string{"Asseverazione tecnica pre-intervento", "Pagamento con bonifico parlante", "Conservare documentazione lavori", "Indicare in dichiarazione dei redditi"},
			Documenti:       []string{"Asseverazione tecnica (classificazione rischio ante/post)", "Fatture e bonifici parlanti", "Titoli abilitativi (SCIA/permesso di costruire)", "Dati catastali dell'immobile"},
			FAQ: []models.FAQ{
				{Domanda: "È cumulabile con il Bonus Ristrutturazione?", Risposta: "Il massimale di €96.000 è spesso condiviso con il Bonus Ristrutturazioni se i lavori sono contestuali, a meno che non siano contabilizzati separatamente come interventi di messa in sicurezza statica."},
				{Domanda: "Cos'è il Sismabonus Acquisti?", Risposta: "Se acquisti un immobile in un edificio demolito e ricostruito antisismicamente da un'impresa (zone 1-2-3), puoi detrarre il 50% (prima casa) o 36% (altre) sul prezzo di vendita, purché l'acquisto avvenga entro 30 mesi dalla fine lavori."},
				{Domanda: "Le aliquote cambieranno?", Risposta: "Sì, dal 2027 scenderanno a 36% (prima casa) e 30% (altre). Conviene pianificare i lavori entro il 2026."},
			},
			LinkUfficiale:        "https://www.agenziaentrate.gov.it/portale/aree-tematiche/casa/agevolazioni",
			LinkRicerca:          "https://www.agenziaentrate.gov.it/portale/ricerca?keywords=sismabonus+detrazione+antisismica",
			Ente:                 "Agenzia delle Entrate",
			FonteURL:             "https://www.agenziaentrate.gov.it/portale/aree-tematiche/casa/agevolazioni",
			FonteNome:            "Agenzia delle Entrate",
			RiferimentiNormativi: []string{"Art. 16, commi 1-bis a 1-septies, DL 63/2013", "Legge di Bilancio 2026 (L. 199/2025), art. 1 comma 22 — Aliquote 50%/36% prorogate"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "attivo",
		},
		{
			ID: "detrazione-mutuo", Nome: "Detrazione Interessi Mutuo Prima Casa", Categoria: "casa",
			Descrizione: "Detrazione IRPEF del 19% sugli interessi passivi e oneri accessori del mutuo ipotecario per l'acquisto dell'abitazione principale, fino a €4.000 annui. Misura strutturale TUIR.",
			Importo: "detrazione 19% fino a €4.000/anno di interessi (max €760/anno)", Scadenza: "In vigore (misura strutturale TUIR)",
			Requisiti:       []string{"Mutuo ipotecario per acquisto abitazione principale", "Immobile adibito ad abitazione principale entro 12 mesi dall'acquisto", "Intestatario del mutuo"},
			ComeRichiederlo: []string{"Indicare in dichiarazione dei redditi (730 o Redditi PF)", "La banca comunica i dati al Sistema TS (precompilato)"},
			Documenti:       []string{"Certificazione interessi passivi dalla banca (inviata annualmente)", "Atto di acquisto", "Contratto di mutuo"},
			FAQ: []models.FAQ{
				{Domanda: "Vale anche se il mutuo è cointestato?", Risposta: "Sì, ciascun cointestatario detrae la propria quota di interessi fino a €4.000 ciascuno, purché entrambi abbiano la residenza nell'immobile."},
				{Domanda: "Se cambio residenza perdo la detrazione?", Risposta: "Sì, se l'immobile non è più la tua abitazione principale perdi il diritto alla detrazione per gli anni successivi."},
			},
			LinkUfficiale:        "https://www.agenziaentrate.gov.it/portale/aree-tematiche/casa/agevolazioni",
			LinkRicerca:          "https://www.agenziaentrate.gov.it/portale/ricerca?keywords=detrazione+interessi+mutuo+prima+casa",
			Ente:                 "Agenzia delle Entrate",
			FonteURL:             "https://www.agenziaentrate.gov.it/portale/aree-tematiche/casa/agevolazioni",
			FonteNome:            "Agenzia delle Entrate",
			RiferimentiNormativi: []string{"Art. 15, comma 1, lett. b), TUIR (DPR 917/1986)"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "attivo",
		},
		{
			ID: "bonus-verde", Nome: "Bonus Verde", Categoria: "casa",
			Descrizione: "Detrazione Irpef del 36% sulle spese per sistemazione a verde di aree scoperte, giardini e terrazzi, fino a €5.000. SCADUTO il 31/12/2024, non prorogato dalla Legge di Bilancio 2025 né dalla Legge di Bilancio 2026. Chi ha sostenuto spese entro il 2024 può ancora detrarre in dichiarazione dei redditi (10 rate annuali).",
			Importo: "detrazione 36% fino a €5.000", Scadenza: "31 dicembre 2024 (non prorogato)",
			Scaduto: true,
			Requisiti:       []string{"Proprietario o nudo proprietario", "Interventi di sistemazione a verde", "Pagamento tracciabile"},
			ComeRichiederlo: []string{"Pagamento tracciabile", "Conservare fatture", "Dichiarazione dei redditi"},
			Documenti:       []string{"Fatture dei lavori", "Ricevute pagamento tracciabile", "Autocertificazione proprietà"},
			FAQ: []models.FAQ{
				{Domanda: "Posso ancora detrarre le spese del 2024?", Risposta: "Sì, le spese sostenute entro il 31/12/2024 si possono detrarre in 10 rate annuali nella dichiarazione dei redditi dal 2025 in poi."},
				{Domanda: "Vale per i balconi?", Risposta: "Rientravano anche giardini pensili e coperture a verde su balconi e terrazzi, ma solo per spese sostenute entro il 2024."},
			},
			LinkUfficiale:        "https://www.agenziaentrate.gov.it/portale/bonus-verde/infogen-bonus-verde",
			LinkRicerca:          "https://www.agenziaentrate.gov.it/portale/bonus-verde/infogen-bonus-verde",
			Ente:                 "Agenzia delle Entrate",
			FonteURL:             "https://www.agenziaentrate.gov.it/portale/bonus-verde/infogen-bonus-verde",
			FonteNome:            "Agenzia delle Entrate",
			RiferimentiNormativi: []string{"Legge 205/2017, art. 1 commi 12-15", "Non prorogato da L. 207/2024 né da L. 198/2025"},
			Stato:                "scaduto",
			UltimoAggiornamento:  "19 febbraio 2026",
		},
		{
			ID: "bonus-colonnine", Nome: "Bonus Colonnine Ricarica Elettrica", Categoria: "casa",
			Descrizione: "Contributo fino all'80% (max €1.500 per privati) per installazione di infrastrutture di ricarica per veicoli elettrici in ambito domestico. SCADUTO: la misura autonoma non è stata rinnovata dalla Legge di Bilancio 2026. Eventuali installazioni possono rientrare nel Bonus Ristrutturazione (50%/36%).",
			Importo: "fino a €1.500 (80% delle spese)", Scadenza: "Fondi esauriti / non rinnovato",
			Scaduto: true,
			Requisiti:       []string{"Persona fisica residente in Italia", "Installazione in ambito domestico", "Installatore qualificato"},
			ComeRichiederlo: []string{"Misura non più attiva come bonus autonomo", "L'installazione può rientrare nel Bonus Ristrutturazione"},
			Documenti:       []string{"Fattura installazione", "Certificato installatore qualificato", "Documentazione immobile"},
			FAQ: []models.FAQ{
				{Domanda: "Il bonus colonnine esiste ancora?", Risposta: "Come misura autonoma al 80% no, non è stato rinnovato. L'installazione può però rientrare nel Bonus Ristrutturazione al 50% (prima casa) o 36% (altre)."},
				{Domanda: "Vale per le colonnine condominiali?", Risposta: "Le installazioni condominiali possono rientrare nel Bonus Ristrutturazione per le parti comuni."},
			},
			LinkUfficiale:        "https://www.mimit.gov.it/it/incentivi/bonus-colonnine-domestiche",
			LinkRicerca:          "https://www.mimit.gov.it/it/incentivi/bonus-colonnine-domestiche",
			Ente:                 "Ministero delle imprese e del Made in Italy",
			FonteURL:             "https://www.mimit.gov.it/it/incentivi/bonus-colonnine-domestiche",
			FonteNome:            "Ministero delle imprese e del Made in Italy",
			RiferimentiNormativi: []string{"DM 25 agosto 2021, n. 358", "Non rinnovato dalla Legge di Bilancio 2026"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "scaduto",
		},
		{
			ID: "bonus-acqua-potabile", Nome: "Bonus Acqua Potabile", Categoria: "casa",
			Descrizione: "Credito d'imposta del 50% sulle spese per sistemi di filtraggio e mineralizzazione dell'acqua potabile, fino a €1.000. SCADUTO il 31/12/2023, non prorogato. Chi ha sostenuto spese entro il 2023 può ancora recuperare il credito residuo in dichiarazione dei redditi.",
			Importo: "credito d'imposta 50% fino a €1.000", Scadenza: "31 dicembre 2023 (non prorogato)",
			Scaduto: true,
			Requisiti:       []string{"Acquisto sistemi filtraggio/mineralizzazione entro il 2023", "Comunicazione spese all'Agenzia delle Entrate"},
			ComeRichiederlo: []string{"Comunicazione spese su sito Agenzia Entrate (scaduta per nuove spese)", "Recupero credito residuo in dichiarazione dei redditi"},
			Documenti:       []string{"Fattura acquisto sistema filtraggio", "Comunicazione all'Agenzia delle Entrate"},
			FAQ: []models.FAQ{
				{Domanda: "Posso ancora comprare un depuratore e avere il bonus?", Risposta: "No, il bonus è scaduto il 31/12/2023 e non è stato prorogato. Puoi solo recuperare il credito residuo per spese sostenute entro il 2023."},
				{Domanda: "Quali sistemi erano ammessi?", Risposta: "Sistemi di filtraggio, mineralizzazione, raffreddamento e addizione di anidride carbonica alimentare, acquistati entro il 2023."},
			},
			LinkUfficiale:        "https://www.agenziaentrate.gov.it/portale/bonus-acqua-potabile",
			LinkRicerca:          "https://www.agenziaentrate.gov.it/portale/ricerca?keywords=bonus+acqua+potabile",
			Ente:                 "Agenzia delle Entrate",
			FonteURL:             "https://www.agenziaentrate.gov.it/portale/bonus-acqua-potabile",
			FonteNome:            "Agenzia delle Entrate",
			RiferimentiNormativi: []string{"Legge di Bilancio 2021, art. 1 commi 1087-1089", "Non prorogato da L. 213/2023 né successive"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "scaduto",
		},

		{
			ID: "bonus-barriere", Nome: "Bonus Barriere Architettoniche 75%", Categoria: "casa",
			Descrizione: "Detrazione del 75% per interventi di superamento ed eliminazione delle barriere architettoniche (ascensori, rampe, automazione porte, servoscala). SCADUTO il 31/12/2025, non prorogato dalla Legge di Bilancio 2026. Chi ha sostenuto spese entro il 2025 può ancora detrarre in dichiarazione dei redditi (5 rate annuali).",
			Importo: "detrazione 75% (massimali da €30.000 a €50.000)", Scadenza: "31 dicembre 2025 (non prorogato)",
			Scaduto: true,
			Requisiti:       []string{"Spese sostenute entro il 31/12/2025", "Interventi conformi ai requisiti DM 236/1989"},
			ComeRichiederlo: []string{"Misura non più attiva per nuove spese", "Spese 2025 detraibili in dichiarazione dei redditi (5 rate annuali)"},
			Documenti:       []string{"Fatture e bonifici parlanti", "Asseverazione conformità DM 236/1989", "Titoli abilitativi"},
			FAQ: []models.FAQ{
				{Domanda: "Posso ancora usufruirne?", Risposta: "Solo per spese sostenute entro il 31/12/2025. La detrazione si recupera in 5 rate annuali in dichiarazione dei redditi."},
				{Domanda: "L'installazione di un ascensore rientra ancora in qualche agevolazione?", Risposta: "Può rientrare nel Bonus Ristrutturazione al 50% (prima casa) o 36% (altre), con massimale di €96.000."},
			},
			LinkUfficiale:        "https://www.agenziaentrate.gov.it/portale/aree-tematiche/casa/agevolazioni",
			LinkRicerca:          "https://www.agenziaentrate.gov.it/portale/ricerca?keywords=bonus+barriere+architettoniche+75",
			Ente:                 "Agenzia delle Entrate",
			FonteURL:             "https://www.agenziaentrate.gov.it/portale/aree-tematiche/casa/agevolazioni",
			FonteNome:            "Agenzia delle Entrate",
			RiferimentiNormativi: []string{"Art. 119-ter DL 34/2020", "Non prorogato da L. 199/2025"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "scaduto",
		},

		// ═══════════════════════════════════════════════════════
		// SALUTE
		// ═══════════════════════════════════════════════════════
		{
			ID: "bonus-psicologo", Nome: "Bonus Psicologo", Categoria: "salute",
			Descrizione: "Contributo per sessioni di psicoterapia con professionisti iscritti all'albo. Importo variabile in base all'ISEE: fino a €1.500 (ISEE ≤ €15.000), €1.000 (ISEE ≤ €30.000), €500 (ISEE ≤ €50.000).",
			Importo: "da €500 a €1.500 in base all'ISEE", Scadenza: "Bando annuale (2025)",
			Scaduto: true,
			Requisiti:       []string{"ISEE ≤ €50.000", "Residenza in Italia", "Psicoterapeuta iscritto all'albo e aderente al bonus"},
			ComeRichiederlo: []string{"Portale INPS con SPID/CIE", "Sezione 'Bonus Psicologo'", "Domanda nel periodo di apertura del bando"},
			Documenti:       []string{"SPID o CIE", "ISEE in corso di validità", "Dati dello psicoterapeuta (nome, cognome, codice albo)"},
			FAQ: []models.FAQ{
				{Domanda: "Quanto ricevo per seduta?", Risposta: "Il bonus copre fino a €50 per seduta, fino al raggiungimento dell'importo totale assegnato in base al tuo ISEE."},
				{Domanda: "Posso scegliere qualsiasi psicologo?", Risposta: "Deve essere uno psicoterapeuta iscritto nell'elenco degli aderenti al bonus psicologo sul portale INPS."},
			},
			LinkUfficiale:        "https://www.inps.it/it/it/dettaglio-scheda.it.schede-servizio-strumento.schede-servizi.contributo-per-sostenere-le-spese-relative-a-sessioni-di-psicoterapia-bonus-psicologo.html",
			LinkRicerca:          "https://www.inps.it/it/it/dettaglio-scheda.it.schede-servizio-strumento.schede-servizi.contributo-per-sostenere-le-spese-relative-a-sessioni-di-psicoterapia-bonus-psicologo.html",
			Ente:                 "INPS",
			FonteURL:             "https://www.inps.it/it/it/dettaglio-scheda.it.schede-servizio-strumento.schede-servizi.contributo-per-sostenere-le-spese-relative-a-sessioni-di-psicoterapia-bonus-psicologo.html",
			FonteNome:            "INPS",
			RiferimentiNormativi: []string{"DL 228/2021, art. 1-quater", "DM 24 novembre 2023"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "scaduto",
		},

		{
			ID: "detrazione-spese-mediche", Nome: "Detrazione Spese Mediche e Sanitarie", Categoria: "salute",
			Descrizione: "Detrazione IRPEF del 19% sulle spese mediche e sanitarie (visite specialistiche, farmaci, analisi, interventi chirurgici, dispositivi medici) per la parte eccedente la franchigia di €129,11. Nessun tetto massimo. Misura strutturale, non ha scadenza.",
			Importo: "detrazione 19% sopra franchigia €129,11 (nessun tetto)", Scadenza: "In vigore (misura strutturale TUIR)",
			Requisiti:       []string{"Spese mediche/sanitarie documentate", "Pagamento tracciabile per visite e prestazioni (esclusi farmaci e dispositivi medici)", "Franchigia di €129,11"},
			ComeRichiederlo: []string{"Conservare scontrini, fatture e ricevute", "Indicare in dichiarazione dei redditi (730 o Redditi PF)", "Le spese del Sistema Tessera Sanitaria sono pre-caricate nel 730 precompilato"},
			Documenti:       []string{"Scontrini parlanti farmacia (con codice fiscale)", "Fatture mediche/specialistiche", "Ricevute pagamento tracciabile"},
			FAQ: []models.FAQ{
				{Domanda: "Quali spese rientrano?", Risposta: "Visite mediche, specialistiche, analisi, farmaci, interventi, protesi, dispositivi medici, occhiali, lenti a contatto, cure dentistiche, fisioterapia, psicoterapia, ticket SSN."},
				{Domanda: "Devo pagare con carta?", Risposta: "Sì, per visite e prestazioni il pagamento deve essere tracciabile (carta, bonifico, assegno). Per farmaci e dispositivi medici è ammesso anche il contante."},
				{Domanda: "Come funziona la franchigia?", Risposta: "Si detraggono solo le spese che superano €129,11. Se spendi €1.000, la detrazione è il 19% di €870,89 = circa €165."},
			},
			LinkUfficiale:        "https://infoprecompilata.agenziaentrate.gov.it/portale/spese-sanitarie",
			LinkRicerca:          "https://www.agenziaentrate.gov.it/portale/ricerca?keywords=detrazione+spese+mediche+sanitarie",
			Ente:                 "Agenzia delle Entrate",
			FonteURL:             "https://infoprecompilata.agenziaentrate.gov.it/portale/spese-sanitarie",
			FonteNome:            "Agenzia delle Entrate",
			RiferimentiNormativi: []string{"Art. 15, comma 1, lett. c), TUIR (DPR 917/1986)"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "attivo",
		},

		// ═══════════════════════════════════════════════════════
		// SPESA / SOSTEGNO AL REDDITO
		// ═══════════════════════════════════════════════════════
		{
			ID: "carta-dedicata", Nome: "Carta Dedicata a Te", Categoria: "spesa",
			Descrizione: "Carta prepagata €500 per acquisto beni alimentari di prima necessità per nuclei con ISEE fino a €15.000 e almeno 3 componenti. Assegnazione automatica senza domanda, erogazione tramite Poste Italiane. Confermata per 2026 e 2027. Scadenza utilizzo saldo 2025: entro il 28 febbraio 2026. Nuova erogazione 2026 prevista nella seconda metà dell'anno, in attesa del decreto attuativo.",
			Importo: "€500 su carta prepagata", Scadenza: "Erogazione automatica",
			Requisiti:       []string{"ISEE fino a €15.000", "Nucleo ≥ 3 componenti", "Nessun altro sostegno al reddito (ADI, Naspi, SFL, ecc.)", "Iscrizione anagrafe popolazione residente"},
			ComeRichiederlo: []string{"Erogazione automatica: nessuna domanda necessaria", "INPS stila graduatoria e la invia ai Comuni", "Ritiro carta presso uffici postali su comunicazione del Comune"},
			Documenti:       []string{"Documento d'identità", "Codice fiscale", "ISEE in corso di validità (presentato autonomamente)"},
			FAQ: []models.FAQ{
				{Domanda: "Come faccio a sapere se mi spetta?", Risposta: "L'assegnazione è automatica: il Comune seleziona i beneficiari dalla graduatoria INPS in base all'ISEE. Riceverai una comunicazione (SMS o lettera) per il ritiro."},
				{Domanda: "Dove posso usare la carta?", Risposta: "Solo per acquisto di beni alimentari di prima necessità nei supermercati e negozi convenzionati. Dal 2025 sono esclusi carburante e alcolici."},
				{Domanda: "Quando arriva?", Risposta: "Le tempistiche dipendono dal decreto attuativo annuale. Nel 2025 le ricariche sono arrivate a novembre. Per il 2026 si attende il decreto nei prossimi mesi."},
			},
			LinkUfficiale:        "https://www.poste.it/carta-dedicata-a-te",
			LinkRicerca:          "https://www.poste.it/carta-dedicata-a-te",
			Ente:                 "Poste Italiane",
			FonteURL:             "https://www.masaf.gov.it/Carta_Dedicata_a_te_2025-info",
			FonteNome:            "MASAF / Poste Italiane",
			RiferimentiNormativi: []string{"DL 48/2023, art. 1 comma 450", "Legge di Bilancio 2026 (L. 198/2025) — Rifinanziamento 500M per 2026 e 2027"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "attivo",
		},
		{
			ID: "carta-acquisti", Nome: "Carta Acquisti", Categoria: "spesa",
			Descrizione: "Carta prepagata da €40/mese (€80 ogni 2 mesi, €480/anno) per over 65 e genitori di bambini sotto 3 anni. Utilizzabile per spese alimentari, farmaci e bollette luce/gas. Include accesso alla tariffa elettrica agevolata e sconti in negozi convenzionati.",
			Importo: "€40/mese (€480/anno)", Scadenza: "In vigore (permanente)",
			Requisiti:       []string{"Over 65: ISEE ≤ €8.230,81 e reddito ≤ €8.230,81 (≤ €10.974,42 se over 70)", "Genitori bimbi under 3: ISEE ≤ €8.230,81", "Non intestatari di più di 1 utenza elettrica domestica, 1 non domestica, 2 gas", "Patrimonio mobiliare ≤ €15.000", "Cittadinanza italiana/UE o permesso di soggiorno lungo periodo"},
			ComeRichiederlo: []string{"Domanda gratuita presso qualsiasi Ufficio Postale", "Compilare il modulo (over 65 o genitori under 3)", "INPS verifica requisiti e, in caso positivo, attiva la carta", "Chi già la riceve e mantiene i requisiti non deve ripresentare domanda"},
			Documenti:       []string{"Modulo domanda (disponibile su mef.gov.it, poste.it, INPS)", "Documento d'identità valido", "ISEE in corso di validità (aggiornare entro 31 gennaio)", "Codice fiscale"},
			FAQ: []models.FAQ{
				{Domanda: "È la stessa cosa della Carta Dedicata a Te?", Risposta: "No, sono due misure diverse. La Carta Acquisti è €40/mese per over 65 e genitori bimbi under 3 (ISEE ≤ €8.230). La Carta Dedicata a Te è €500 una tantum per nuclei ≥3 componenti (ISEE ≤ €15.000)."},
				{Domanda: "Dove posso usarla?", Risposta: "Nei negozi alimentari e farmacie abilitate al circuito Mastercard, e per pagare bollette luce/gas agli Uffici Postali."},
				{Domanda: "Quando arrivano gli accrediti?", Risposta: "Ogni 2 mesi (gennaio, marzo, maggio, luglio, settembre, novembre) con €80 per bimestre."},
			},
			LinkUfficiale:        "https://www.mef.gov.it/focus/Carta-Acquisti/",
			LinkRicerca:          "https://www.mef.gov.it/focus/Carta-Acquisti/",
			Ente:                 "MEF / Poste Italiane",
			FonteURL:             "https://www.mef.gov.it/focus/Carta-Acquisti/",
			FonteNome:            "MEF",
			RiferimentiNormativi: []string{"DL 112/2008, art. 81 comma 32", "DM 16 settembre 2008", "Aggiornamento ISTAT 2026 — ISEE €8.230,81"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "attivo",
		},
		{
			ID: "adi", Nome: "Assegno di Inclusione (ADI)", Categoria: "sostegno",
			Descrizione: "Sostegno economico per nuclei con minori, disabili, over 60 o in condizione di svantaggio. Sostituisce il Reddito di Cittadinanza.",
			Importo: "fino a €6.000/anno (+ integrazione affitto fino a €3.360)", Scadenza: "In vigore",
			Requisiti:       []string{"ISEE ≤ €9.360", "Nucleo con minori, disabili, over 60", "Residenza in Italia da almeno 5 anni", "Patrimonio mobiliare ≤ €6.000"},
			ComeRichiederlo: []string{"Portale INPS o patronato", "Iscrizione al SIISL", "Colloquio presso servizi sociali"},
			Documenti:       []string{"SPID o CIE", "ISEE in corso di validità", "Documento d'identità", "Attestazione disabilità (se applicabile)"},
			FAQ: []models.FAQ{
				{Domanda: "È compatibile con un lavoro part-time?", Risposta: "Sì, fino a un certo reddito da lavoro. L'importo dell'ADI viene ricalcolato in base al reddito percepito."},
				{Domanda: "Quanto dura?", Risposta: "L'ADI dura 18 mesi, rinnovabili per periodi di 12 mesi previo aggiornamento dei requisiti."},
			},
			LinkUfficiale:        "https://www.inps.it/it/it/dettaglio-scheda.it.schede-servizio-strumento.schede-servizi.assegno-di-inclusione-adi.html",
			LinkRicerca:          "https://www.inps.it/it/it/dettaglio-scheda.it.schede-servizio-strumento.schede-servizi.assegno-di-inclusione-adi.html",
			Ente:                 "INPS",
			FonteURL:             "https://www.inps.it/it/it/dettaglio-scheda.it.schede-servizio-strumento.schede-servizi.assegno-di-inclusione-adi.html",
			FonteNome:            "INPS",
			RiferimentiNormativi: []string{"DL 48/2023, convertito in L. 85/2023", "Circolare INPS n. 105/2023"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "attivo",
		},

		// ═══════════════════════════════════════════════════════
		// LAVORO
		// ═══════════════════════════════════════════════════════
		{
			ID: "sfl", Nome: "Supporto Formazione e Lavoro", Categoria: "lavoro",
			Descrizione: "Indennità di €350/mese per 12 mesi per persone tra 18 e 59 anni occupabili che partecipano a percorsi di formazione o lavoro.",
			Importo: "€350/mese per 12 mesi", Scadenza: "In vigore",
			Requisiti:       []string{"Età 18-59 anni", "ISEE ≤ €6.000", "Non beneficiario ADI", "Partecipazione a percorsi formativi"},
			ComeRichiederlo: []string{"Portale INPS o patronato", "Iscrizione al SIISL", "Adesione a percorso formativo/lavorativo"},
			Documenti:       []string{"SPID o CIE", "ISEE in corso di validità", "Curriculum vitae", "Iscrizione centro per l'impiego"},
			FAQ: []models.FAQ{
				{Domanda: "Devo frequentare un corso di formazione?", Risposta: "Sì, l'indennità è condizionata alla partecipazione attiva a percorsi formativi o di riqualificazione professionale."},
				{Domanda: "Posso rifiutare offerte di lavoro?", Risposta: "Il rifiuto di un'offerta di lavoro congrua comporta la decadenza dal beneficio."},
			},
			LinkUfficiale:        "https://www.inps.it/it/it/dettaglio-scheda.it.schede-servizio-strumento.schede-servizi.supporto-per-la-formazione-e-il-lavoro-sfl-.html",
			LinkRicerca:          "https://www.inps.it/it/it/dettaglio-scheda.it.schede-servizio-strumento.schede-servizi.supporto-per-la-formazione-e-il-lavoro-sfl-.html",
			Ente:                 "INPS",
			FonteURL:             "https://www.inps.it/it/it/dettaglio-scheda.it.schede-servizio-strumento.schede-servizi.supporto-per-la-formazione-e-il-lavoro-sfl-.html",
			FonteNome:            "INPS",
			RiferimentiNormativi: []string{"DL 48/2023, art. 12", "Circolare INPS n. 77/2023"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "attivo",
		},

		// ═══════════════════════════════════════════════════════
		// UTENZE
		// ═══════════════════════════════════════════════════════
		{
			ID: "bonus-bollette", Nome: "Bonus Sociale Bollette (Luce, Gas, Acqua, TARI)", Categoria: "utenze",
			Descrizione: "Sconto automatico in bolletta per famiglie con disagio economico. Comprende 4 agevolazioni: sconto 30% sulla bolletta elettrica, 15% sul gas, 50 litri/giorno/abitante per l'acqua, e dal 2026 anche 25% sulla TARI (tassa rifiuti). Applicato automaticamente con ISEE valido, senza bisogno di domanda.",
			Importo: "sconto 30% luce + 15% gas + acqua gratuita (50L/giorno) + 25% TARI", Scadenza: "In vigore (annuale, automatico con ISEE)",
			Requisiti:       []string{"ISEE ≤ €9.796 (aggiornato dal 2026, era €9.530)", "oppure ISEE ≤ €20.000 per nuclei con almeno 4 figli a carico", "oppure percettori di Assegno di Inclusione (ADI) indipendentemente dall'ISEE", "DSU/ISEE in corso di validità presentata all'INPS"},
			ComeRichiederlo: []string{"Nessuna domanda necessaria: il bonus è automatico", "Presentare la DSU per ottenere l'ISEE aggiornato (online su inps.it o tramite CAF)", "L'incrocio dati INPS-ARERA-SII attiva lo sconto in bolletta", "Lo sconto appare in bolletta come 'Compensazione Bonus Sociale' o 'Bonus Sociale'"},
			Documenti:       []string{"ISEE in corso di validità (presentare DSU)", "Nessun altro documento richiesto"},
			FAQ: []models.FAQ{
				{Domanda: "Devo fare domanda al mio fornitore?", Risposta: "No, il bonus è completamente automatico. Basta avere un ISEE valido e lo sconto viene applicato direttamente in bolletta dal tuo fornitore."},
				{Domanda: "Se presento l'ISEE in ritardo perdo i mesi precedenti?", Risposta: "No, il bonus è retroattivo: se presenti l'ISEE a giugno, ricevi lo sconto anche per i mesi da gennaio a maggio in un'unica soluzione."},
				{Domanda: "Cos'è il nuovo bonus TARI 2026?", Risposta: "Dal 2026 si aggiunge uno sconto del 25% sulla tassa rifiuti (TARI), con gli stessi requisiti ISEE degli altri bonus sociali. Anche questo è automatico."},
			},
			LinkUfficiale:        "https://www.arera.it/consumatori/bonus-sociale",
			LinkRicerca:          "https://www.arera.it/consumatori/bonus-sociale",
			Ente:                 "ARERA",
			FonteURL:             "https://www.arera.it/consumatori/bonus-sociale",
			FonteNome:            "ARERA",
			RiferimentiNormativi: []string{"Delibera ARERA 2/2026/R/com del 24 gennaio 2026", "DM 29 dicembre 2016 (meccanismo adeguamento ISEE)"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "attivo",
		},

		// ═══════════════════════════════════════════════════════
		// ISTRUZIONE
		// ═══════════════════════════════════════════════════════
		{
			ID: "carta-cultura", Nome: "Carta della Cultura / Merito", Categoria: "istruzione",
			Descrizione: "€500 Carta Cultura per neodiciottenni (ISEE ≤ €35.000) + €500 Carta Merito (diploma con 100). Cumulabili fino a €1.000.",
			Importo: "€500 (fino a €1.000 cumulate)", Scadenza: "Entro 30 giugno dell'anno successivo ai 18 anni",
			Requisiti:       []string{"18 anni compiuti nell'anno precedente", "ISEE ≤ €35.000 (Carta Cultura)", "Diploma con 100 (Carta Merito)"},
			ComeRichiederlo: []string{"Registrarsi su cartacultura.gov.it", "Accesso con SPID", "Generare buoni per acquisti culturali"},
			Documenti:       []string{"SPID", "Diploma di maturità (per Carta Merito)", "ISEE in corso di validità"},
			FAQ: []models.FAQ{
				{Domanda: "Cosa posso comprare?", Risposta: "Libri, musica, biglietti cinema/teatro/concerti/musei, corsi di formazione, abbonamenti a quotidiani digitali."},
				{Domanda: "Posso averle entrambe?", Risposta: "Sì, se hai sia ISEE ≤ €35.000 sia diploma con 100, puoi cumulare Carta Cultura e Carta Merito per un totale di €1.000."},
			},
			LinkUfficiale:        "https://www.cartacultura.gov.it",
			LinkRicerca:          "https://www.cartacultura.gov.it",
			Ente:                 "Ministero della Cultura",
			FonteURL:             "https://www.cartacultura.gov.it",
			FonteNome:            "Ministero della Cultura",
			RiferimentiNormativi: []string{"DL 230/2023, art. 1", "DPCM 20 luglio 2023"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "attivo",
		},
		{
			ID: "borsa-studio", Nome: "Borse di Studio Universitarie", Categoria: "istruzione",
			Descrizione: "Borsa di studio regionale per studenti universitari meritevoli e con basso ISEE. Copre tasse, vitto e alloggio.",
			Importo: "da €2.000 a €6.000/anno + esenzione tasse", Scadenza: "Bando regionale (luglio-settembre)",
			Requisiti:       []string{"Iscrizione università/AFAM", "ISEE universitario ≤ €23.000-€26.000", "Requisiti di merito (CFU minimi)"},
			ComeRichiederlo: []string{"Portale ente regionale diritto allo studio", "Domanda online nel periodo del bando", "Allegare ISEE universitario"},
			Documenti:       []string{"ISEE universitario", "Iscrizione universitaria", "Piano di studi", "Certificato esami sostenuti"},
			FAQ: []models.FAQ{
				{Domanda: "Devo ripresentare domanda ogni anno?", Risposta: "Sì, la domanda va rinnovata ogni anno accademico, verificando il possesso dei requisiti di reddito e merito."},
				{Domanda: "Se perdo i requisiti di merito devo restituire i soldi?", Risposta: "Non devi restituire quanto già ricevuto, ma perdi il diritto alla borsa per l'anno successivo."},
			},
			LinkUfficiale:        "https://www.inps.it/it/it/risultati-ricerca.html",
			LinkRicerca:          "https://www.inps.it/it/it/risultati-ricerca.html",
			Ente:                 "MUR / Ente DSU Regionale",
			FonteURL:             "https://www.mur.gov.it/it/aree-tematiche/diritto-allo-studio",
			FonteNome:            "Ministero dell'Università e della Ricerca",
			RiferimentiNormativi: []string{"D.Lgs. 68/2012", "DPCM annuale soglie ISEE"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "attivo",
		},

		// ═══════════════════════════════════════════════════════
		// ALTRO
		// ═══════════════════════════════════════════════════════
		{
			ID: "bonus-decoder-tv", Nome: "Bonus TV / Decoder", Categoria: "altro",
			Descrizione: "Contributo per acquisto TV e decoder compatibili con il nuovo digitale terrestre DVB-T2 per famiglie con ISEE fino a €20.000. SCADUTO: fondi esauriti nel 2024.",
			Importo: "fino a €50 (decoder) / €100 (TV)", Scadenza: "Fondi esauriti (2024)",
			Scaduto: true,
			Requisiti:       []string{"ISEE ≤ €20.000", "Residenza in Italia", "Rottamazione vecchio apparecchio (per bonus TV)"},
			ComeRichiederlo: []string{"Misura non più attiva: fondi esauriti"},
			Documenti:       []string{"Documento d'identità", "Autocertificazione ISEE", "Vecchio apparecchio da rottamare"},
			FAQ: []models.FAQ{
				{Domanda: "Posso ancora richiederlo?", Risposta: "No, i fondi sono esauriti e la misura non è stata rifinanziata."},
				{Domanda: "Serve rottamare la vecchia TV?", Risposta: "Per il bonus TV serviva la rottamazione. La misura non è più attiva."},
			},
			LinkUfficiale:        "https://www.mimit.gov.it/it/incentivi",
			LinkRicerca:          "https://www.mimit.gov.it/it/incentivi",
			Ente:                 "MIMIT",
			FonteURL:             "https://www.mimit.gov.it/it/incentivi",
			FonteNome:            "Ministero delle Imprese e del Made in Italy",
			RiferimentiNormativi: []string{"DM 18 ottobre 2021"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "scaduto",
		},
		{
			ID: "bonus-animali", Nome: "Bonus Animali Domestici", Categoria: "altro",
			Descrizione: "Detrazione del 19% sulle spese veterinarie per animali domestici legalmente detenuti, fino a €550 (con franchigia di €129,11).",
			Importo: "detrazione 19% fino a €550", Scadenza: "In vigore (annuale)",
			Requisiti:       []string{"Possesso legale di animale domestico", "Spese veterinarie documentate", "Franchigia di €129,11"},
			ComeRichiederlo: []string{"Conservare fatture/scontrini veterinario", "Indicare in dichiarazione dei redditi"},
			Documenti:       []string{"Fatture/scontrini veterinario", "Documentazione possesso animale"},
			FAQ: []models.FAQ{
				{Domanda: "Vale per tutti gli animali?", Risposta: "Solo per animali domestici legalmente detenuti (cani, gatti, ecc.). Non si applica ad animali da reddito o allevamento."},
				{Domanda: "Come funziona la franchigia?", Risposta: "La detrazione si applica sulle spese che superano €129,11, fino a un massimo di €550."},
			},
			LinkUfficiale:        "https://infoprecompilata.agenziaentrate.gov.it/portale/spese-sanitarie",
			LinkRicerca:          "https://www.agenziaentrate.gov.it/portale/ricerca?keywords=spese+veterinarie+animali+detrazione",
			Ente:                 "Agenzia delle Entrate",
			FonteURL:             "https://infoprecompilata.agenziaentrate.gov.it/portale/spese-sanitarie",
			FonteNome:            "Agenzia delle Entrate",
			RiferimentiNormativi: []string{"Art. 15, comma 1, lett. c-bis, TUIR"},
			UltimoAggiornamento:  "19 febbraio 2026",
			Stato:                "attivo",
		},
	}
	populateValidity(bonuses)
	return bonuses
}

// GetAllBonusWithRegional returns national + regional bonuses combined.
func GetAllBonusWithRegional() []models.Bonus {
	all := GetAllBonus()
	all = append(all, GetRegionalBonus()...)
	return all
}

// MatchBonus matches user profile against available bonuses.
// If bonusList is provided, uses that; otherwise falls back to GetAllBonusWithRegional().
func MatchBonus(profile models.UserProfile, bonusList ...[]models.Bonus) models.MatchResult {
	var allBonus []models.Bonus
	if len(bonusList) > 0 && len(bonusList[0]) > 0 {
		allBonus = bonusList[0]
	} else {
		allBonus = GetAllBonusWithRegional()
	}
	var matched []models.Bonus
	totalSaving := 0.0

	userRegion := strings.ToLower(strings.TrimSpace(profile.Residenza))

	for _, b := range allBonus {
		// Regional filter: if bonus has RegioniApplicabili, user must match
		if len(b.RegioniApplicabili) > 0 {
			if userRegion == "" {
				continue
			}
			found := false
			for _, r := range b.RegioniApplicabili {
				if strings.ToLower(r) == userRegion {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		score := calcScore(b.ID, profile)
		// For regional bonuses not in calcScore, use generic ISEE-based scoring
		if score == 0 && len(b.RegioniApplicabili) > 0 {
			score = calcRegionalScore(b, profile)
		}
		if score > 0 {
			b.Compatibilita = score
			b.ImportoReale = calcImportoReale(b.ID, profile.ISEE, profile)
			matched = append(matched, b)
			saving := estimateSaving(b.ID, profile)
			if saving == 0 && len(b.RegioniApplicabili) > 0 {
				saving = estimateRegionalSaving(b)
			}
			totalSaving += saving
		}
	}

	// Mark expired bonuses and count
	attivi := 0
	scaduti := 0
	activeSaving := 0.0
	for i := range matched {
		matched[i].Scaduto = isScaduto(matched[i].Scadenza)
		if matched[i].Scaduto {
			scaduti++
		} else {
			attivi++
			s := estimateSaving(matched[i].ID, profile)
			if s == 0 && len(matched[i].RegioniApplicabili) > 0 {
				s = estimateRegionalSaving(matched[i])
			}
			activeSaving += s
		}
	}

	// Sort: active first (by compat desc), then expired (by compat desc)
	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].Scaduto != matched[j].Scaduto {
			return !matched[i].Scaduto
		}
		return matched[i].Compatibilita > matched[j].Compatibilita
	})

	perso := calcPersoFinora(activeSaving)

	return models.MatchResult{
		BonusTrovati:     len(matched),
		BonusAttivi:      attivi,
		BonusScaduti:     scaduti,
		RisparmioStimato: formatEuro(activeSaving),
		PersoFinora:      perso,
		Bonus:            matched,
	}
}

// calcRegionalScore scores regional bonuses based on ISEE threshold and category match.
func calcRegionalScore(b models.Bonus, p models.UserProfile) int {
	if b.SogliaISEE > 0 && p.ISEE > 0 && p.ISEE > b.SogliaISEE {
		return 0
	}
	cat := strings.ToLower(b.Categoria)
	switch cat {
	case "famiglia":
		if p.NumeroFigli > 0 || p.FigliMinorenni > 0 || p.FigliUnder3 > 0 {
			if b.SogliaISEE == 0 || p.ISEE == 0 || p.ISEE <= b.SogliaISEE {
				return 75
			}
		}
		return 0
	case "istruzione":
		if p.Studente || p.FigliMinorenni > 0 || p.NumeroFigli > 0 {
			return 70
		}
		return 0
	case "casa":
		if p.Affittuario || p.PrimaAbitazione {
			return 70
		}
		return 0
	case "trasporti":
		if p.Studente || p.Eta < 26 || p.Over65 > 0 || (p.ISEE > 0 && p.ISEE <= 30000) {
			return 65
		}
		return 0
	}
	return 50
}

// estimateRegionalSaving estimates annual savings for regional bonuses.
func estimateRegionalSaving(b models.Bonus) float64 {
	cat := strings.ToLower(b.Categoria)
	switch cat {
	case "famiglia":
		return 800
	case "istruzione":
		return 200
	case "casa":
		return 2000
	case "trasporti":
		return 400
	}
	return 300
}

// calcPersoFinora calculates the estimated amount lost since January.
func calcPersoFinora(annualSaving float64) string {
	if annualSaving <= 0 {
		return ""
	}
	monthsElapsed := float64(time.Now().Month() - 1)
	if monthsElapsed <= 0 {
		return ""
	}
	monthlySaving := annualSaving / 12
	perso := monthlySaving * monthsElapsed
	if perso < 10 {
		return ""
	}
	return formatEuro(perso)
}

func calcImportoReale(bonusID string, isee float64, profile models.UserProfile) string {
	switch bonusID {
	case "assegno-unico":
		var perFiglio float64
		switch {
		case isee > 0 && isee <= 17090.61:
			perFiglio = 199.4
		case isee > 17090.61 && isee <= 45574.96:
			perFiglio = 199.4 - (isee-17090.61)/(45574.96-17090.61)*(199.4-57)
		default:
			perFiglio = 57
		}

		under3Bonus := 91.40 * float64(profile.FigliUnder3)

		var thirdChildBonus float64
		if profile.NumeroFigli >= 3 {
			thirdChildBonus = 17.10 * float64(profile.NumeroFigli-2)
		}

		monthly := math.Round(perFiglio*float64(profile.NumeroFigli)*100) / 100
		monthly += under3Bonus + thirdChildBonus
		monthly = math.Round(monthly*100) / 100
		yearly := math.Round(monthly*12*100) / 100

		return fmt.Sprintf("€%.2f/mese (€%.2f/anno)", monthly, yearly)

	case "bonus-nido":
		switch {
		case isee > 0 && isee <= 25000:
			return "€3.600/anno"
		case isee > 25000 && isee <= 40000:
			return "€2.500/anno"
		default:
			return "€1.500/anno"
		}

	case "bonus-psicologo":
		switch {
		case isee > 0 && isee <= 15000:
			return "fino a €1.500"
		case isee > 15000 && isee <= 30000:
			return "fino a €1.000"
		case isee > 30000 && isee <= 50000:
			return "fino a €500"
		}

	case "bonus-mamma":
		if profile.NumeroFigli >= 3 && profile.Occupazione == "dipendente" {
			return "esonero IVS fino a €3.000/anno (in busta paga)"
		}
		return "€60/mese (€720/anno, erogati a dicembre)"

	case "bonus-affitto-giovani":
		return "da €991,60 a €2.000/anno (20% del canone)"

	case "bonus-ristrutturazione":
		if profile.PrimaAbitazione {
			return "detrazione 50% fino a €96.000 (prima casa)"
		}
		return "detrazione 36% fino a €96.000 (seconda casa)"

	case "ecobonus":
		if profile.PrimaAbitazione {
			return "detrazione 50% (prima casa)"
		}
		return "detrazione 36% (seconda casa)"

	case "sismabonus":
		if profile.PrimaAbitazione {
			return "detrazione 50% fino a €96.000 (prima casa)"
		}
		return "detrazione 36% fino a €96.000 (seconda casa)"

	case "detrazione-mutuo":
		return "detrazione fino a €760/anno (19% su max €4.000 di interessi)"

	case "bonus-bollette":
		return "sconto automatico ~€400/anno su luce, gas, acqua e TARI"

	case "carta-acquisti":
		return "€40/mese (€480/anno)"
	}

	return ""
}

func calcScore(id string, p models.UserProfile) int {
	switch id {
	case "assegno-unico":
		if p.NumeroFigli > 0 {
			if p.ISEE > 0 && p.ISEE <= 17000 {
				return 98
			}
			return 85
		}
	case "bonus-nido":
		if p.FigliUnder3 > 0 {
			if p.ISEE > 0 && p.ISEE <= 25000 {
				return 95
			}
			return 70
		}
	case "bonus-nascita":
		if p.NuovoNato2026 && (p.ISEE == 0 || p.ISEE <= 40000) {
			return 95
		}
	case "bonus-mamma":
		if p.NumeroFigli >= 2 {
			// Dipendenti o autonome: bonus €60/mese
			if p.Occupazione == "dipendente" || p.Occupazione == "autonomo" {
				if p.RedditoAnnuo == 0 || p.RedditoAnnuo <= 40000 {
					return 85
				}
			}
		}
	case "bonus-ristrutturazione":
		if p.RistrutturazCasa {
			return 90
		}
	case "bonus-mobili":
		if p.RistrutturazCasa {
			return 80
		}
	case "bonus-affitto-giovani":
		if p.Eta >= 20 && p.Eta <= 30 && p.Affittuario {
			if p.RedditoAnnuo > 0 && p.RedditoAnnuo <= 15493 {
				return 95
			}
			return 60
		}
	case "prima-casa-under36":
		if p.Eta > 0 && p.Eta < 36 && p.PrimaAbitazione {
			if p.ISEE > 0 && p.ISEE <= 40000 {
				return 90
			}
			return 65
		}
	case "ecobonus":
		if p.RistrutturazCasa {
			return 75
		}
	case "bonus-verde":
		// Scaduto — mostrare solo con score basso per informazione
		if p.RistrutturazCasa || p.PrimaAbitazione {
			return 25
		}
	case "bonus-psicologo":
		if p.ISEE > 0 && p.ISEE <= 50000 {
			return 70
		}
		return 40 // always show, very popular
	case "carta-dedicata":
		if p.ISEE > 0 && p.ISEE <= 15000 && (p.NumeroFigli+1+p.Over65) >= 3 {
			return 90
		}
	case "carta-cultura":
		if p.Eta == 18 || p.Eta == 19 {
			if p.ISEE > 0 && p.ISEE <= 35000 {
				return 95
			}
			return 70
		}
	case "borsa-studio":
		if p.Studente && (p.ISEE == 0 || p.ISEE <= 26000) {
			return 90
		}
	case "bonus-decoder-tv":
		// Scaduto — score bassissimo
		if p.ISEE > 0 && p.ISEE <= 20000 {
			return 15
		}
	case "adi":
		if p.ISEE > 0 && p.ISEE <= 9360 {
			if p.FigliMinorenni > 0 || p.Disabilita || p.Over65 > 0 {
				return 95
			}
		}
	case "sfl":
		if p.Eta >= 18 && p.Eta <= 59 && p.ISEE > 0 && p.ISEE <= 6000 {
			if p.Occupazione == "disoccupato" || p.Occupazione == "inoccupato" {
				return 90
			}
		}
	case "bonus-animali":
		return 30 // generic, always show low
	case "bonus-colonnine":
		// Scaduto come bonus autonomo
		if p.PrimaAbitazione || p.RistrutturazCasa {
			return 20
		}
	case "bonus-acqua-potabile":
		// Scaduto
		if p.PrimaAbitazione || p.RistrutturazCasa {
			return 15
		}
	case "bonus-bollette":
		if p.ISEE > 0 && p.ISEE <= 9796 {
			return 95
		}
		if p.ISEE > 0 && p.ISEE <= 20000 && p.NumeroFigli >= 4 {
			return 95
		}
		if p.ISEE > 0 && p.ISEE <= 20000 {
			return 40 // mostra per informazione
		}
	case "carta-acquisti":
		if p.Over65 > 0 && p.ISEE > 0 && p.ISEE <= 8230 {
			return 90
		}
		if p.FigliUnder3 > 0 && p.ISEE > 0 && p.ISEE <= 8230 {
			return 90
		}
	case "sismabonus":
		if p.RistrutturazCasa {
			return 80
		}
	case "detrazione-spese-mediche":
		return 35 // universale
	case "detrazione-mutuo":
		if p.PrimaAbitazione && !p.Affittuario {
			return 80
		}
	case "bonus-barriere":
		// Scaduto
		if p.RistrutturazCasa || p.Over65 > 0 {
			return 15
		}
	}
	return 0
}

func estimateSaving(id string, p models.UserProfile) float64 {
	switch id {
	case "assegno-unico":
		base := 1500.0
		if p.ISEE > 0 && p.ISEE <= 17000 {
			base = 2400.0
		}
		return base * float64(p.NumeroFigli)
	case "bonus-nido":
		if p.ISEE <= 25000 {
			return 3600
		}
		return 1500
	case "bonus-nascita":
		return 1000
	case "bonus-mamma":
		if p.NumeroFigli >= 3 && p.Occupazione == "dipendente" {
			return 3000 // esonero IVS
		}
		return 720 // bonus €60/mese
	case "bonus-ristrutturazione":
		return 5000
	case "bonus-mobili":
		return 2500
	case "bonus-affitto-giovani":
		return 1500 // media tra min 991,60 e max 2000
	case "prima-casa-under36":
		return 3000 // solo garanzia, stime ridotte rispetto a quando c'erano esenzioni
	case "ecobonus":
		return 3000
	case "bonus-verde":
		return 0 // scaduto, non genera risparmio futuro
	case "bonus-psicologo":
		switch {
		case p.ISEE > 0 && p.ISEE <= 15000:
			return 1500
		case p.ISEE > 15000 && p.ISEE <= 30000:
			return 1000
		case p.ISEE > 30000 && p.ISEE <= 50000:
			return 500
		default:
			return 600
		}
	case "carta-dedicata":
		return 500
	case "carta-cultura":
		return 500
	case "borsa-studio":
		return 4000
	case "bonus-decoder-tv":
		return 0 // scaduto
	case "adi":
		return 6000
	case "sfl":
		return 4200
	case "bonus-animali":
		return 100
	case "bonus-colonnine":
		return 0 // scaduto come bonus autonomo
	case "bonus-acqua-potabile":
		return 0 // scaduto
	case "bonus-bollette":
		return 400
	case "carta-acquisti":
		return 480
	case "sismabonus":
		return 5000
	case "detrazione-spese-mediche":
		return 200
	case "detrazione-mutuo":
		return 760
	case "bonus-barriere":
		return 0 // scaduto
	}
	return 0
}

// populateValidity auto-derives TipoScadenza, ScadenzaDomanda, AnnoConferma, UltimaVerifica
// from existing Scadenza text for each bonus.
func populateValidity(bonuses []models.Bonus) {
	now := time.Now()
	for i := range bonuses {
		b := &bonuses[i]
		lower := strings.ToLower(strings.TrimSpace(b.Scadenza))

		// Derive TipoScadenza from Scadenza text
		switch {
		case strings.Contains(lower, "in vigore"):
			b.TipoScadenza = "permanente"
		case strings.Contains(lower, "erogazione automatica"):
			b.TipoScadenza = "permanente"
		case strings.Contains(lower, "entro 60 giorni"):
			b.TipoScadenza = "permanente"
		case strings.Contains(lower, "per arretrati"):
			b.TipoScadenza = "permanente"
		case strings.Contains(lower, "entro 30 giugno"):
			b.TipoScadenza = "permanente"
		case strings.Contains(lower, "esaurimento fondi"):
			b.TipoScadenza = "esaurimento_fondi"
		case strings.Contains(lower, "bando"):
			b.TipoScadenza = "bando_annuale"
		case strings.Contains(lower, "non prorogato"):
			b.TipoScadenza = "scaduto"
		case strings.Contains(lower, "fondi esauriti"):
			b.TipoScadenza = "scaduto"
		default:
			if m := itDateRe.FindStringSubmatch(lower); len(m) == 4 {
				day, _ := strconv.Atoi(m[1])
				month := italianMonthsMap[m[2]]
				year, _ := strconv.Atoi(m[3])
				b.TipoScadenza = "data_fissa"
				b.ScadenzaDomanda = time.Date(year, month, day, 23, 59, 59, 0, time.UTC)
			} else {
				b.TipoScadenza = "permanente"
			}
		}

		// AnnoConferma: derive from UltimoAggiornamento text
		if b.UltimoAggiornamento != "" {
			if m := yearOnlyRe.FindStringSubmatch(b.UltimoAggiornamento); len(m) == 2 {
				y, _ := strconv.Atoi(m[1])
				b.AnnoConferma = y
			}
		}
		if b.AnnoConferma == 0 {
			b.AnnoConferma = 2026
		}

		// UltimaVerifica: set to now (data is loaded from code)
		b.UltimaVerifica = now
	}
}