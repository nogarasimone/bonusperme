package handlers

import (
	"bonusperme/internal/config"
	"html/template"
	"log"
	"net/http"
	"sync"
)

var (
	indexTmpl     *template.Template
	indexTmplOnce sync.Once
)

type indexData struct {
	GTMID            string
	TurnstileSiteKey string
}

func loadIndexTemplate() {
	var err error
	indexTmpl, err = template.ParseFiles("static/index.html")
	if err != nil {
		log.Printf("[index] template parse error: %v — falling back to static file", err)
		indexTmpl = nil
	}
}

// IndexHandler serves index.html with dynamic GTM_ID injection via Go templates.
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	indexTmplOnce.Do(loadIndexTemplate)

	if indexTmpl == nil {
		// Fallback: serve as static file
		http.ServeFile(w, r, "static/index.html")
		return
	}

	data := indexData{
		GTMID:            config.Cfg.GTMID,
		TurnstileSiteKey: config.Cfg.TurnstileSiteKey,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := indexTmpl.Execute(w, data); err != nil {
		log.Printf("[index] template execute error: %v", err)
		http.ServeFile(w, r, "static/index.html")
	}
}
