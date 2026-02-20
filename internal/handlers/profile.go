package handlers

import (
	"bonusperme/internal/models"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
)

// compactProfile holds only non-identifying fields for the profile code.
type compactProfile struct {
	Eta              int     `json:"e,omitempty"`
	NumeroFigli      int     `json:"f,omitempty"`
	FigliMinorenni   int     `json:"fm,omitempty"`
	FigliUnder3      int     `json:"f3,omitempty"`
	Over65           int     `json:"o,omitempty"`
	ISEE             float64 `json:"i,omitempty"`
	RedditoAnnuo     float64 `json:"r,omitempty"`
	Residenza        string  `json:"re,omitempty"`
	StatoCivile      string  `json:"sc,omitempty"`
	Occupazione      string  `json:"oc,omitempty"`
	Disabilita       bool    `json:"d,omitempty"`
	Affittuario      bool    `json:"af,omitempty"`
	PrimaAbitazione  bool    `json:"pa,omitempty"`
	RistrutturazCasa bool    `json:"rc,omitempty"`
	Studente         bool    `json:"st,omitempty"`
	NuovoNato2026    bool    `json:"nn,omitempty"`
}

func toCompact(p models.UserProfile) compactProfile {
	return compactProfile{
		Eta: p.Eta, NumeroFigli: p.NumeroFigli, FigliMinorenni: p.FigliMinorenni,
		FigliUnder3: p.FigliUnder3, Over65: p.Over65, ISEE: p.ISEE,
		RedditoAnnuo: p.RedditoAnnuo, Residenza: p.Residenza, StatoCivile: p.StatoCivile,
		Occupazione: p.Occupazione, Disabilita: p.Disabilita, Affittuario: p.Affittuario,
		PrimaAbitazione: p.PrimaAbitazione, RistrutturazCasa: p.RistrutturazCasa,
		Studente: p.Studente, NuovoNato2026: p.NuovoNato2026,
	}
}

func fromCompact(c compactProfile) models.UserProfile {
	return models.UserProfile{
		Eta: c.Eta, NumeroFigli: c.NumeroFigli, FigliMinorenni: c.FigliMinorenni,
		FigliUnder3: c.FigliUnder3, Over65: c.Over65, ISEE: c.ISEE,
		RedditoAnnuo: c.RedditoAnnuo, Residenza: c.Residenza, StatoCivile: c.StatoCivile,
		Occupazione: c.Occupazione, Disabilita: c.Disabilita, Affittuario: c.Affittuario,
		PrimaAbitazione: c.PrimaAbitazione, RistrutturazCasa: c.RistrutturazCasa,
		Studente: c.Studente, NuovoNato2026: c.NuovoNato2026,
	}
}

const codePrefix = "BPM-"

// EncodeProfileHandler encodes a profile into a shareable code.
func EncodeProfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var profile models.UserProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	compact := toCompact(profile)
	data, err := json.Marshal(compact)
	if err != nil {
		http.Error(w, "Encoding error", http.StatusInternalServerError)
		return
	}

	encoded := base64.RawURLEncoding.EncodeToString(data)
	code := codePrefix + encoded
	if len(code) > 64 {
		code = code[:64]
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]string{"code": code})
}

// DecodeProfileHandler decodes a profile code back to a UserProfile.
func DecodeProfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" || !strings.HasPrefix(code, codePrefix) {
		http.Error(w, "Codice non valido", http.StatusBadRequest)
		return
	}

	// Max length check to prevent abuse
	if len(code) > 256 {
		http.Error(w, "Codice troppo lungo", http.StatusBadRequest)
		return
	}

	encoded := strings.TrimPrefix(code, codePrefix)
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		http.Error(w, "Codice malformato", http.StatusBadRequest)
		return
	}

	var compact compactProfile
	if err := json.Unmarshal(data, &compact); err != nil {
		http.Error(w, "Codice non decodificabile", http.StatusBadRequest)
		return
	}

	profile := fromCompact(compact)

	// Validate decoded profile
	if msg, ok := validateProfile(profile); !ok {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(profile)
}
