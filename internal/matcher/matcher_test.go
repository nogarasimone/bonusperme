package matcher

import (
	"bonusperme/internal/models"
	"math"
	"strings"
	"testing"
)

func TestMatchBonus_FamigliaConFigli(t *testing.T) {
	profile := models.UserProfile{
		Eta: 35, NumeroFigli: 2, FigliMinorenni: 2, FigliUnder3: 1,
		ISEE: 15000, Residenza: "Lombardia", StatoCivile: "sposato", Occupazione: "dipendente",
	}
	result := MatchBonus(profile)
	if result.BonusTrovati == 0 {
		t.Error("Profilo con 2 figli e ISEE 15000 dovrebbe trovare almeno 1 bonus")
	}
	found := false
	for _, b := range result.Bonus {
		if b.ID == "assegno-unico" || b.Nome == "Assegno Unico Universale" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Assegno Unico mancante")
	}
}

func TestMatchBonus_SingleSenzaFigli(t *testing.T) {
	profile := models.UserProfile{Eta: 28, NumeroFigli: 0, ISEE: 60000, Occupazione: "dipendente"}
	result := MatchBonus(profile)
	for _, b := range result.Bonus {
		if b.Categoria == "famiglia" && b.Compatibilita > 50 {
			t.Errorf("Single senza figli non dovrebbe avere bonus famiglia: %s (%d%%)", b.Nome, b.Compatibilita)
		}
	}
}

func TestMatchBonus_Minorenne(t *testing.T) {
	result := MatchBonus(models.UserProfile{Eta: 16})
	_ = result // non deve crashare
}

func TestMatchBonus_ISEEZero(t *testing.T) {
	profile := models.UserProfile{Eta: 30, NumeroFigli: 1, FigliMinorenni: 1, ISEE: 0}
	result := MatchBonus(profile)
	if result.BonusTrovati == 0 {
		t.Error("Con figli e ISEE 0, almeno Assegno Unico minimo")
	}
}

func TestMatchBonus_RisparmioCoerente(t *testing.T) {
	profile := models.UserProfile{
		Eta: 32, NumeroFigli: 3, FigliMinorenni: 3, FigliUnder3: 1,
		ISEE: 12000, Affittuario: true, Residenza: "Lazio",
	}
	result := MatchBonus(profile)
	if result.BonusTrovati > 0 && result.RisparmioStimato == "" {
		t.Error("Risparmio stimato vuoto con bonus trovati")
	}
}

func TestMatchBonus_BonusRegionali(t *testing.T) {
	pLom := models.UserProfile{Eta: 35, NumeroFigli: 1, FigliMinorenni: 1, ISEE: 12000, Residenza: "Lombardia"}
	pCam := models.UserProfile{Eta: 35, NumeroFigli: 1, FigliMinorenni: 1, ISEE: 12000, Residenza: "Campania"}
	rLom := MatchBonus(pLom)
	rCam := MatchBonus(pCam)
	hasLom, hasCam := false, false
	for _, b := range rLom.Bonus {
		if b.ID == "dote-scuola-lombardia" {
			hasLom = true
		}
	}
	for _, b := range rCam.Bonus {
		if b.ID == "dote-scuola-lombardia" {
			hasCam = true
		}
	}
	if !hasLom {
		t.Error("Lombardia dovrebbe vedere Dote Scuola")
	}
	if hasCam {
		t.Error("Campania NON dovrebbe vedere Dote Scuola Lombardia")
	}
}

func TestMatchBonus_RegionaliSicilia(t *testing.T) {
	p := models.UserProfile{Eta: 30, PrimaAbitazione: true, ISEE: 25000, Residenza: "Sicilia"}
	result := MatchBonus(p)
	hasPrimaCasa := false
	for _, b := range result.Bonus {
		if b.ID == "prima-casa-giovani-sicilia" {
			hasPrimaCasa = true
		}
	}
	if !hasPrimaCasa {
		t.Error("Sicilia dovrebbe vedere Prima Casa Giovani")
	}
}

func TestMatchBonus_SenzaRegione(t *testing.T) {
	p := models.UserProfile{Eta: 30, NumeroFigli: 1, FigliMinorenni: 1, ISEE: 15000}
	result := MatchBonus(p)
	for _, b := range result.Bonus {
		if len(b.RegioniApplicabili) > 0 {
			t.Errorf("Senza regione non dovrebbe avere bonus regionali: %s", b.Nome)
		}
	}
}

// ═══════════════════════════════════════════════════════
// Assegno Unico 2026 — Test dettagliati
// Valori da Circolare INPS n. 7 del 30 gennaio 2026
// ═══════════════════════════════════════════════════════

func TestAssegnoUnico2026(t *testing.T) {

	t.Run("ISEE_basso_2_figli_minorenni", func(t *testing.T) {
		// ISEE €15.000 (sotto soglia €17.468,51) → €203,80/mese per figlio
		p := models.UserProfile{
			Eta: 35, NumeroFigli: 2, FigliMinorenni: 2, ISEE: 15000,
		}
		monthly := calcAssegnoUnicoMensile(p)
		expected := 203.80 * 2
		if math.Abs(monthly-expected) > 0.02 {
			t.Errorf("Atteso ~€%.2f/mese, ottenuto €%.2f", expected, monthly)
		}
	})

	t.Run("ISEE_medio_1_figlio_interpolazione", func(t *testing.T) {
		// ISEE €30.000 → interpolazione lineare tra €203,80 e €58,30
		p := models.UserProfile{
			Eta: 35, NumeroFigli: 1, FigliMinorenni: 1, ISEE: 30000,
		}
		monthly := calcAssegnoUnicoMensile(p)
		if monthly <= 58.30 || monthly >= 203.80 {
			t.Errorf("Con ISEE €30.000 il mensile dovrebbe essere tra €58,30 e €203,80, ottenuto €%.2f", monthly)
		}
	})

	t.Run("senza_ISEE_1_figlio_importo_minimo", func(t *testing.T) {
		// ISEE 0 (non presentato) → €58,30/mese
		p := models.UserProfile{
			Eta: 30, NumeroFigli: 1, FigliMinorenni: 1, ISEE: 0,
		}
		monthly := calcAssegnoUnicoMensile(p)
		if math.Abs(monthly-58.30) > 0.02 {
			t.Errorf("Senza ISEE, atteso €58,30/mese, ottenuto €%.2f", monthly)
		}
	})

	t.Run("ISEE_basso_3_figli_1_under1_maggiorazioni", func(t *testing.T) {
		// ISEE €15.000, 3 figli minorenni, 2 under 3, 1 under 1
		// Base: 203,80 × 3 = 611,40
		// Maggiorazione under 1 (50%): 203,80 × 0,50 × 1 = 101,90
		// Maggiorazione figli 1-3 in nuclei 3+ figli: 203,80 × 0,50 × 1 (figli1a3 = under3-under1 = 2-1 = 1) = 101,90
		// Maggiorazione 3° figlio: 99,10 × 1 = 99,10
		// Totale: 611,40 + 101,90 + 101,90 + 99,10 = 914,30
		p := models.UserProfile{
			Eta: 35, NumeroFigli: 3, FigliMinorenni: 3, FigliUnder3: 2, FigliUnder1: 1,
			ISEE: 15000,
		}
		monthly := calcAssegnoUnicoMensile(p)
		expected := 914.30
		if math.Abs(monthly-expected) > 0.10 {
			t.Errorf("Atteso ~€%.2f/mese, ottenuto €%.2f", expected, monthly)
		}
	})

	t.Run("ISEE_alto_1_figlio_oltre_soglia", func(t *testing.T) {
		// ISEE €50.000 (oltre €46.582,71) → €58,30/mese
		p := models.UserProfile{
			Eta: 35, NumeroFigli: 1, FigliMinorenni: 1, ISEE: 50000,
		}
		monthly := calcAssegnoUnicoMensile(p)
		if math.Abs(monthly-58.30) > 0.02 {
			t.Errorf("ISEE oltre soglia, atteso €58,30/mese, ottenuto €%.2f", monthly)
		}
	})

	t.Run("1_figlio_maggiorenne_ISEE_basso", func(t *testing.T) {
		// 1 figlio 18-21 con ISEE basso → €99,10/mese (non €203,80)
		p := models.UserProfile{
			Eta: 45, NumeroFigli: 1, FigliMinorenni: 0, FigliMaggiorenni: 1,
			ISEE: 15000,
		}
		monthly := calcAssegnoUnicoMensile(p)
		if math.Abs(monthly-99.10) > 0.02 {
			t.Errorf("Figlio maggiorenne ISEE basso, atteso €99,10/mese, ottenuto €%.2f", monthly)
		}
	})

	t.Run("entrambi_genitori_lavoratori_ISEE_basso", func(t *testing.T) {
		// 2 figli minorenni + entrambi genitori lavoratori + ISEE basso
		// Base: 203,80 × 2 = 407,60
		// Maggiorazione lavoratori: 34,90 × 2 = 69,80
		// Totale: 477,40
		p := models.UserProfile{
			Eta: 35, NumeroFigli: 2, FigliMinorenni: 2, ISEE: 15000,
			EntrambiGenitoriLavoratori: true,
		}
		monthly := calcAssegnoUnicoMensile(p)
		expected := 407.60 + 69.80
		if math.Abs(monthly-expected) > 0.10 {
			t.Errorf("Con entrambi lavoratori, atteso ~€%.2f/mese, ottenuto €%.2f", expected, monthly)
		}
	})

	t.Run("figlio_disabile_grave_ISEE_basso", func(t *testing.T) {
		// 1 figlio minorenne disabile grave, ISEE basso
		// Base: 203,80
		// Maggiorazione disabilità grave: 110,60
		// Totale: 314,40
		p := models.UserProfile{
			Eta: 40, NumeroFigli: 1, FigliMinorenni: 1, ISEE: 15000,
			FigliDisabili: 1, DisabilitaFigli: "grave",
		}
		monthly := calcAssegnoUnicoMensile(p)
		expected := 203.80 + 110.60
		if math.Abs(monthly-expected) > 0.10 {
			t.Errorf("Figlio disabile grave, atteso ~€%.2f/mese, ottenuto €%.2f", expected, monthly)
		}
	})

	t.Run("4_figli_forfait_150", func(t *testing.T) {
		// 4 figli minorenni, ISEE basso → include forfait €150
		// Base: 203,80 × 4 = 815,20
		// Magg. 3° e 4° figlio: 99,10 × 2 = 198,20
		// Forfait 4+ figli: 150,00
		// Totale: 1163,40
		p := models.UserProfile{
			Eta: 40, NumeroFigli: 4, FigliMinorenni: 4, ISEE: 15000,
		}
		monthly := calcAssegnoUnicoMensile(p)
		expected := 815.20 + 198.20 + 150.0
		if math.Abs(monthly-expected) > 0.10 {
			t.Errorf("4 figli con forfait, atteso ~€%.2f/mese, ottenuto €%.2f", expected, monthly)
		}
	})

	t.Run("madre_under21", func(t *testing.T) {
		// Madre under 21, 1 figlio, ISEE basso
		// Base: 203,80
		// Magg. madre under 21: 23,30 × 1 = 23,30
		// Totale: 227,10
		p := models.UserProfile{
			Eta: 20, NumeroFigli: 1, FigliMinorenni: 1, ISEE: 15000,
			MadreUnder21: true,
		}
		monthly := calcAssegnoUnicoMensile(p)
		expected := 203.80 + 23.30
		if math.Abs(monthly-expected) > 0.10 {
			t.Errorf("Madre under 21, atteso ~€%.2f/mese, ottenuto €%.2f", expected, monthly)
		}
	})

	t.Run("calcScore_ISEE_basso", func(t *testing.T) {
		p := models.UserProfile{Eta: 35, NumeroFigli: 2, FigliMinorenni: 2, ISEE: 15000}
		score := calcScore("assegno-unico", p)
		if score != 98 {
			t.Errorf("Score con ISEE ≤17.468,51 dovrebbe essere 98, ottenuto %d", score)
		}
	})

	t.Run("calcScore_ISEE_sopra_soglia", func(t *testing.T) {
		p := models.UserProfile{Eta: 35, NumeroFigli: 1, FigliMinorenni: 1, ISEE: 30000}
		score := calcScore("assegno-unico", p)
		if score != 85 {
			t.Errorf("Score con ISEE >17.468,51 dovrebbe essere 85, ottenuto %d", score)
		}
	})

	t.Run("calcImportoReale_contiene_mese_anno", func(t *testing.T) {
		p := models.UserProfile{Eta: 35, NumeroFigli: 2, FigliMinorenni: 2, ISEE: 15000}
		importo := calcImportoReale("assegno-unico", p.ISEE, p)
		if !strings.Contains(importo, "/mese") || !strings.Contains(importo, "/anno") {
			t.Errorf("ImportoReale dovrebbe contenere /mese e /anno, ottenuto: %s", importo)
		}
	})

	t.Run("estimateSaving_coerente_con_mensile", func(t *testing.T) {
		p := models.UserProfile{Eta: 35, NumeroFigli: 2, FigliMinorenni: 2, ISEE: 15000}
		saving := estimateSaving("assegno-unico", p)
		monthly := calcAssegnoUnicoMensile(p)
		expectedAnnual := math.Round(monthly*12*100) / 100
		if math.Abs(saving-expectedAnnual) > 0.10 {
			t.Errorf("estimateSaving (€%.2f) dovrebbe essere coerente con mensile×12 (€%.2f)", saving, expectedAnnual)
		}
	})

	t.Run("entrambi_lavoratori_ISEE_alto_nessuna_maggiorazione", func(t *testing.T) {
		// ISEE oltre soglia max → maggiorazione lavoratori = 0
		p := models.UserProfile{
			Eta: 35, NumeroFigli: 1, FigliMinorenni: 1, ISEE: 50000,
			EntrambiGenitoriLavoratori: true,
		}
		monthly := calcAssegnoUnicoMensile(p)
		if math.Abs(monthly-58.30) > 0.02 {
			t.Errorf("Con ISEE alto, maggiorazione lavoratori deve essere 0, ottenuto €%.2f", monthly)
		}
	})
}
