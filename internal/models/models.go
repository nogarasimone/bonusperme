package models

import "time"

type UserProfile struct {
	Eta              int     `json:"eta"`
	Residenza        string  `json:"residenza"`
	Comune           string  `json:"comune"`
	StatoCivile      string  `json:"stato_civile"`
	Occupazione      string  `json:"occupazione"`
	NumeroFigli      int     `json:"numero_figli"`
	FigliMinorenni   int     `json:"figli_minorenni"`
	FigliUnder3      int     `json:"figli_under3"`
	Disabilita       bool    `json:"disabilita"`
	Over65           int     `json:"over65"`
	ISEE             float64 `json:"isee"`
	RedditoAnnuo     float64 `json:"reddito_annuo"`
	Affittuario      bool    `json:"affittuario"`
	PrimaAbitazione  bool    `json:"prima_abitazione"`
	RistrutturazCasa bool    `json:"ristrutturaz_casa"`
	Studente         bool    `json:"studente"`
	NuovoNato2026    bool    `json:"nuovo_nato_2026"`
	ISEESimulato     float64 `json:"isee_simulato,omitempty"`
}

type FAQ struct {
	Domanda  string `json:"domanda"`
	Risposta string `json:"risposta"`
}

type BonusTrad struct {
	Descrizione     string   `json:"descrizione"`
	Requisiti       []string `json:"requisiti,omitempty"`
	ComeRichiederlo []string `json:"come_richiederlo,omitempty"`
	FAQ             []FAQ    `json:"faq,omitempty"`
}

type Bonus struct {
	ID                   string               `json:"id"`
	Nome                 string               `json:"nome"`
	Categoria            string               `json:"categoria"`
	Descrizione          string               `json:"descrizione"`
	Importo              string               `json:"importo"`
	ImportoReale         string               `json:"importo_reale,omitempty"`
	Scadenza             string               `json:"scadenza"`
	Scaduto              bool                 `json:"scaduto"`
	Requisiti            []string             `json:"requisiti"`
	ComeRichiederlo      []string             `json:"come_richiederlo"`
	Documenti            []string             `json:"documenti,omitempty"`
	FAQ                  []FAQ                `json:"faq,omitempty"`
	LinkUfficiale        string               `json:"link_ufficiale"`
	Ente                 string               `json:"ente"`
	Compatibilita        int                  `json:"compatibilita"`
	UltimoAggiornamento  string               `json:"ultimo_aggiornamento,omitempty"`
	Fonte                string               `json:"fonte,omitempty"`
	Stato                string               `json:"stato,omitempty"`
	FonteURL             string               `json:"fonte_url,omitempty"`
	FonteNome            string               `json:"fonte_nome,omitempty"`
	RiferimentiNormativi []string             `json:"riferimenti_normativi,omitempty"`
	RegioniApplicabili        []string             `json:"regioni,omitempty"`
	SogliaISEE                float64              `json:"soglia_isee,omitempty"`
	LinkRicerca               string               `json:"link_ricerca,omitempty"`
	LinkVerificato            bool                  `json:"link_verificato"`
	LinkVerificatoAl          string               `json:"link_verificato_al,omitempty"`
	FonteAggiornamento        string               `json:"fonte_aggiornamento,omitempty"`
	VerificaManualeNecessaria bool                  `json:"verifica_manuale_necessaria,omitempty"`
	NotaVerifica              string               `json:"nota_verifica,omitempty"`
	ScadenzaDomanda           time.Time            `json:"scadenza_domanda,omitempty"`
	TipoScadenza              string               `json:"tipo_scadenza,omitempty"`
	AnnoConferma              int                  `json:"anno_conferma,omitempty"`
	UltimaVerifica            time.Time            `json:"ultima_verifica,omitempty"`
	StatoValidita             string               `json:"stato_validita,omitempty"`
	MotivoStato               string               `json:"motivo_stato,omitempty"`
	Traduzioni                map[string]BonusTrad `json:"traduzioni,omitempty"`
	LinkOriginale             string               `json:"link_originale,omitempty"`
	ConfidenceScore           float64              `json:"confidence_score,omitempty"`
	SourcesCount              int                  `json:"sources_count,omitempty"`
	Corroborated              bool                 `json:"corroborated,omitempty"`
	ConflictFields            []string             `json:"conflict_fields,omitempty"`
	ImportoConfermato         string               `json:"importo_confermato,omitempty"`
	ISEEConfermata            float64              `json:"isee_confermata,omitempty"`
	FontiCorroborate          int                  `json:"fonti_corroborate,omitempty"`
	UltimaVerificaGU          *time.Time           `json:"ultima_verifica_gu,omitempty"`
	UltimaVerificaRSS         *time.Time           `json:"ultima_verifica_rss,omitempty"`
	UltimaVerificaSito        *time.Time           `json:"ultima_verifica_sito,omitempty"`
}

type MatchResult struct {
	BonusTrovati     int     `json:"bonus_trovati"`
	BonusAttivi      int     `json:"bonus_attivi"`
	BonusScaduti     int     `json:"bonus_scaduti"`
	RisparmioStimato string  `json:"risparmio_stimato"`
	PersoFinora      string  `json:"perso_finora,omitempty"`
	Bonus            []Bonus   `json:"bonus"`
	Avvisi           []Avviso  `json:"avvisi,omitempty"`
}

type Avviso struct {
	BonusID   string `json:"bonus_id"`
	Tipo      string `json:"tipo"`
	Messaggio string `json:"messaggio"`
}

type SimulateResult struct {
	Reale          MatchResult `json:"reale"`
	Simulato       MatchResult `json:"simulato"`
	BonusExtra     int         `json:"bonus_extra"`
	RisparmioExtra string      `json:"risparmio_extra"`
}
