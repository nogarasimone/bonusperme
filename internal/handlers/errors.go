package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
)

// NotFoundHandler serves a styled 404 page or JSON error for API routes.
func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "endpoint not found"})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(errorPageHTML("404", "Pagina non trovata", "La pagina che cerchi non esiste o è stata spostata.")))
}

// InternalErrorHandler serves a styled 500 page or JSON error for API routes.
func InternalErrorHandler(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(errorPageHTML("500", "Errore del server", "Si è verificato un errore. Riprova tra qualche istante.")))
}

func errorPageHTML(code, title, message string) string {
	return `<!DOCTYPE html>
<html lang="it">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + title + ` — BonusPerMe</title>
<meta name="robots" content="noindex">
<link rel="icon" type="image/png" sizes="32x32" href="/favicon-32x32.png">
<meta name="theme-color" content="#1B3A54">
<link rel="stylesheet" href="/fonts/fonts.css">
<style>
:root{--ink:#1C1C1F;--ink-75:#404045;--ink-50:#76767C;--ink-15:#D4D4D7;--warm-white:#FAFAF7;--blue:#1B3A54;--blue-mid:#2D5F8A;--terra:#C0522E;--radius:5px;--radius-lg:8px}
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:'DM Sans',-apple-system,sans-serif;background:var(--warm-white);color:var(--ink);min-height:100vh;display:flex;flex-direction:column;font-size:15px;line-height:1.65;-webkit-font-smoothing:antialiased}
h1,h2{font-family:'DM Serif Display',Georgia,serif;font-weight:400}
a{color:var(--blue-mid);text-decoration:none}a:hover{text-decoration:underline}
.topbar{background:var(--blue);color:rgba(255,255,255,0.85);font-size:.72rem;padding:7px 0;text-align:center;letter-spacing:.02em}
.header{background:#fff;border-bottom:1px solid var(--ink-15);height:54px;display:flex;align-items:center;justify-content:center}
.logo{display:flex;align-items:center;gap:8px;text-decoration:none;color:var(--ink)}
.logo-mark{width:26px;height:26px;background:var(--blue);border-radius:4px;display:flex;align-items:center;justify-content:center;color:#fff;font-family:'DM Serif Display',serif;font-size:.75rem;font-weight:700}
.error-wrap{flex:1;display:flex;align-items:center;justify-content:center;text-align:center;padding:40px 24px}
.error-code{font-family:'DM Serif Display',serif;font-size:clamp(5rem,15vw,8rem);color:var(--terra);line-height:1;margin-bottom:8px;opacity:.85}
.error-wrap h1{font-size:clamp(1.3rem,3vw,1.8rem);margin-bottom:12px}
.error-wrap p{color:var(--ink-75);max-width:480px;margin:0 auto 24px;font-size:1rem}
.btn-home{display:inline-block;padding:12px 28px;background:var(--blue);color:#fff;border-radius:var(--radius);font-weight:600;font-size:.95rem;text-decoration:none}
.btn-home:hover{background:var(--blue-mid);text-decoration:none}
footer{border-top:1px solid var(--ink-15);padding:24px 0;text-align:center;color:var(--ink-50);font-size:.82rem}
footer a{color:var(--blue-mid);margin:0 8px}
</style>
</head>
<body>
<div class="topbar">BonusPerMe — Servizio gratuito per le famiglie italiane</div>
<header class="header"><a href="/" class="logo"><div class="logo-mark">B</div> BonusPerMe</a></header>
<main class="error-wrap">
<div>
<div class="error-code">` + code + `</div>
<h1>` + title + `</h1>
<p>` + message + `</p>
<a href="/" class="btn-home">Torna alla home</a>
</div>
</main>
<footer><a href="/">Home</a><a href="/contatti">Contatti</a><a href="/privacy">Privacy</a></footer>
</body>
</html>`
}
