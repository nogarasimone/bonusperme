package handlers

import (
	"bonusperme/internal/i18n"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func init() {
	InitCounter()
	SetTranslationLoader(i18n.GetAll)
}

func TestMatchHandler_Valid(t *testing.T) {
	body := `{"eta":30,"residenza":"Lazio","stato_civile":"coniugato/a","occupazione":"dipendente","numero_figli":2,"figli_minorenni":2,"figli_under3":1,"isee":15000,"reddito_annuo":25000}`
	req := httptest.NewRequest(http.MethodPost, "/api/match", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	MatchHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}
	if result["bonus_trovati"].(float64) == 0 {
		t.Error("Dovrebbe trovare almeno 1 bonus")
	}
}

func TestMatchHandler_AgeLessThan18(t *testing.T) {
	body := `{"eta":16,"numero_figli":0,"isee":15000}`
	req := httptest.NewRequest(http.MethodPost, "/api/match", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	MatchHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 for age < 18, got %d", w.Code)
	}
}

func TestTranslationsHandler_EN(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/translations?lang=en", nil)
	w := httptest.NewRecorder()

	TranslationsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var translations map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &translations); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}
	if translations["hero.title"] == "" {
		t.Error("hero.title dovrebbe essere presente nelle traduzioni EN")
	}
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	HealthDetailedHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}
	if result["status"] != "ok" {
		t.Error("Health status should be ok")
	}
}

func TestEncodeDecodeProfile(t *testing.T) {
	// Encode
	body := `{"eta":30,"residenza":"Lombardia","numero_figli":2,"isee":15000}`
	req := httptest.NewRequest(http.MethodPost, "/api/encode-profile", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	EncodeProfileHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Encode: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var encResult map[string]string
	json.Unmarshal(w.Body.Bytes(), &encResult)
	code := encResult["code"]
	if code == "" || !strings.HasPrefix(code, "BPM-") {
		t.Fatalf("Expected code with BPM- prefix, got: %s", code)
	}

	// Decode
	req2 := httptest.NewRequest(http.MethodGet, "/api/decode-profile?code="+code, nil)
	w2 := httptest.NewRecorder()

	DecodeProfileHandler(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("Decode: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var profile map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &profile)
	if profile["eta"].(float64) != 30 {
		t.Errorf("Expected eta 30, got %v", profile["eta"])
	}
	if profile["residenza"] != "Lombardia" {
		t.Errorf("Expected residenza Lombardia, got %v", profile["residenza"])
	}
}

func TestBonusListHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/bonus", nil)
	w := httptest.NewRecorder()

	BonusListHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var bonuses []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &bonuses); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}
	if len(bonuses) < 40 {
		t.Errorf("Expected 40+ bonuses, got %d", len(bonuses))
	}
}
