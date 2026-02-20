package matcher

import (
	"bonusperme/internal/models"
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
