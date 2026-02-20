# BonusPerMe

**Scopri in 2 minuti tutti i bonus e le agevolazioni a cui hai diritto — gratuito, anonimo, open source.**

![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-AGPL--3.0-blue?style=flat-square)
![Updated](https://img.shields.io/badge/bonus_aggiornati-Febbraio_2026-green?style=flat-square)
![Languages](https://img.shields.io/badge/lingue-7_(IT_EN_FR_ES_RO_AR_SQ)-orange?style=flat-square)
![Privacy](https://img.shields.io/badge/tracking-zero-red?style=flat-square)

> **[bonusperme.it](https://bonusperme.it)** — nessuna registrazione, nessun dato salvato

---

## Problema

Ogni anno in Italia oltre 2 miliardi di euro in bonus e agevolazioni restano non richiesti. Le informazioni sono frammentate tra INPS, Agenzia delle Entrate, MEF, Regioni e Comuni. Capire a cosa si ha diritto richiede ore di ricerca su decine di siti diversi.

## Soluzione

BonusPerMe centralizza tutto: rispondi a 4 domande, ricevi la lista completa dei bonus compatibili con importi calcolati sulla tua situazione, requisiti e istruzioni passo-passo. Stampa il report e portalo al CAF.

---

## Funzionalità

- **20+ bonus nazionali e 20+ bonus regionali** aggiornati alla Legge di Bilancio 2026
- **Calcolo importo personalizzato** basato su ISEE, reddito e composizione familiare
- **Upload ISEE PDF** — estrazione automatica del valore ISEE dal documento
- **Report PDF** scaricabile con fonti normative e riferimenti ufficiali
- **Calendario scadenze** esportabile in formato .ics (Google Calendar, Apple Calendar)
- **Simulatore ISEE** — scopri quali bonus otterresti con un ISEE diverso
- **Profilo condivisibile** — codice BPM-* per condividere la propria situazione
- **Pagina dedicata per i CAF** — con registrazione e dati Open Data API
- **7 lingue** — Italiano, English, Francais, Espanol, Romana, العربية, Shqip
- **Scraper automatico** che aggiorna i dati da fonti istituzionali ogni 24h
- **Link checker** che verifica la raggiungibilita dei link ufficiali
- **Validity checker** che rileva bonus scaduti o modificati
- **Zero cookie, zero tracking, zero database** — GDPR compliant by design

---

## Architettura

```
Cloudflare (DNS + CDN + WAF)
    |
    | HTTPS (Full mode, Origin Certificate)
    v
AWS Elastic Beanstalk (Docker on Amazon Linux 2023)
    |
    | Nginx reverse proxy (:443 SSL -> :8080)
    v
Container Go (single binary, ~15 MB)
    |
    |-- Matcher engine (matching profilo -> bonus)
    |-- Scraper (INPS, AdE, MEF, editoriali)
    |-- Link checker (verifica URL ogni 24h)
    |-- Validity checker (scadenze, stato bonus)
    |-- PDF generator (report per CAF)
    |-- i18n (7 lingue, ~150 chiavi ciascuna)
    v
Nessun database — tutto in memoria + file statico
```

### Stack

| Componente | Tecnologia |
|------------|-----------|
| Backend | Go 1.25+ (`net/http`, zero framework) |
| Frontend | HTML/CSS/JS vanilla (singolo file, zero dipendenze JS) |
| PDF | [gofpdf](https://github.com/jung-kurt/gofpdf) |
| ISEE parser | [ledongthuc/pdf](https://github.com/ledongthuc/pdf) |
| HTML scraper | [golang.org/x/net/html](https://pkg.go.dev/golang.org/x/net/html) |
| Error tracking | [Sentry](https://sentry.io) (opzionale) |
| Anti-bot | [Cloudflare Turnstile](https://www.cloudflare.com/products/turnstile/) |
| Database | Nessuno |

---

## Quick Start

```bash
git clone https://github.com/nogarasimone/bonusperme.git
cd bonusperme
cp .env.example .env   # configura le variabili (opzionale)
go run main.go         # http://localhost:8080
```

### Docker

```bash
docker build -t bonusperme .
docker run -p 8080:8080 bonusperme
```

### Docker Compose

```bash
docker-compose up -d
```

Il container gira come utente non-root, filesystem read-only, con limite 256 MB RAM e health check integrato.

---

## Struttura del progetto

```
bonusperme/
├── main.go                          # Entry point, routing, middleware chain
├── internal/
│   ├── config/config.go             # Configurazione da .env / variabili ambiente
│   ├── handlers/
│   │   ├── handlers.go              # API: match, stats, parse-isee
│   │   ├── extra.go                 # API: calendar, simulate, report PDF
│   │   ├── opendata.go              # API: /api/bonus (Open Data)
│   │   ├── profile.go               # API: encode/decode profilo condivisibile
│   │   ├── infra.go                 # SEO: sitemap, robots.txt, pagine bonus
│   │   ├── index.go                 # Template index.html con GTM injection
│   │   ├── layout.go                # Layout condiviso (topbar, header, footer, CSS)
│   │   ├── percaf.go                # Pagina /per-caf
│   │   ├── contact.go               # Pagina /contatti + handler POST
│   │   ├── counter.go               # Contatore persistente (counter.json)
│   │   ├── ratelimit.go             # Rate limiter per IP (token bucket)
│   │   ├── turnstile.go             # Verifica Cloudflare Turnstile
│   │   └── errors.go                # Handler 404/500 personalizzati
│   ├── matcher/
│   │   ├── matcher.go               # Engine di matching + 20 bonus nazionali
│   │   └── regionals.go             # 20+ bonus regionali (tutte le regioni)
│   ├── models/models.go             # Struct: UserProfile, Bonus, MatchResult
│   ├── scraper/
│   │   ├── sources.go               # Lista sorgenti (INPS, AdE, MEF, editoriali)
│   │   ├── parsers.go               # Parser HTML per ogni tipo di fonte
│   │   ├── enricher.go              # Merge dati scraped con dati hardcoded
│   │   ├── scheduler.go             # Scheduler 24h + cache thread-safe
│   │   └── regional.go              # Scraper bonus regionali
│   ├── datasource/
│   │   ├── datasource.go            # Manager sorgenti dati ufficiali
│   │   ├── inps.go                  # Fetcher INPS
│   │   ├── ade.go                   # Fetcher Agenzia delle Entrate
│   │   ├── mise.go                  # Fetcher MISE
│   │   ├── gu_rss.go                # Fetcher Gazzetta Ufficiale RSS
│   │   └── opendata.go              # Fetcher OpenData PA
│   ├── i18n/translations.go         # 7 lingue, ~150 chiavi ciascuna
│   ├── middleware/middleware.go      # Recovery, Security Headers, Gzip
│   ├── linkcheck/linkcheck.go       # Verifica link ufficiali ogni 24h
│   ├── validity/
│   │   ├── validity.go              # Controllo scadenze e stato bonus
│   │   ├── checker.go               # Logic di validazione
│   │   ├── alerts.go                # Sistema alert bonus scaduti
│   │   ├── news.go                  # Monitoraggio novita normative
│   │   └── admin.go                 # API admin (protetta da API key)
│   ├── logger/logger.go             # Logger strutturato
│   ├── sentry/sentry.go             # Integrazione Sentry
│   └── telegram/bot.go              # Bot Telegram (coming soon)
├── static/
│   ├── index.html                   # Frontend completo (single file)
│   ├── privacy.html                 # Privacy policy
│   ├── cookie-policy.html           # Cookie policy
│   ├── fonts/                       # DM Sans, DM Serif Display, JetBrains Mono
│   ├── sw.js                        # Service Worker (PWA)
│   └── manifest.json                # Web App Manifest
├── Dockerfile                       # Multi-stage build (Go + Alpine)
├── docker-compose.yml               # Deploy con limiti e health check
├── .ebextensions/                   # AWS Elastic Beanstalk (security group SSL)
├── .platform/                       # Nginx config + predeploy hooks
└── .github/workflows/ci.yml         # CI: vet, build, test
```

---

## API

### Pubbliche

| Metodo | Path | Descrizione |
|--------|------|-------------|
| `POST` | `/api/match` | Calcola bonus compatibili (richiede Turnstile) |
| `POST` | `/api/simulate` | Simula con ISEE diverso |
| `POST` | `/api/parse-isee` | Estrai ISEE da PDF (max 5 MB) |
| `POST` | `/api/report` | Genera report PDF |
| `GET` | `/api/calendar?bonuses=id1,id2` | Calendario scadenze .ics |
| `GET` | `/api/translations?lang=it` | Dizionario traduzioni |
| `GET` | `/api/stats` | Contatore verifiche e statistiche |
| `GET` | `/api/health` | Health check dettagliato |
| `GET` | `/api/status` | Stato server e ultimo aggiornamento |
| `GET` | `/api/scraper-status` | Stato fonti scraper |

### Open Data

| Metodo | Path | Descrizione |
|--------|------|-------------|
| `GET` | `/api/bonus` | Lista completa bonus (nazionali + regionali) |
| `GET` | `/api/bonus/{id}` | Dettaglio singolo bonus |

Formato JSON, CORS abilitato, cache 1 ora, rate limit 60 req/min.

### Profilo condivisibile

| Metodo | Path | Descrizione |
|--------|------|-------------|
| `POST` | `/api/encode-profile` | Genera codice BPM-* da un profilo |
| `GET` | `/api/decode-profile?code=BPM-*` | Decodifica profilo da codice |

### Admin (protette da `ADMIN_API_KEY`)

| Metodo | Path | Descrizione |
|--------|------|-------------|
| `GET` | `/api/admin/alerts` | Alert bonus scaduti/modificati |
| `GET` | `/api/admin/bonus-status` | Stato validita di ogni bonus |

---

## Configurazione

Tutte le impostazioni sono in variabili d'ambiente (o file `.env`):

| Variabile | Default | Descrizione |
|-----------|---------|-------------|
| `PORT` | `8080` | Porta server |
| `BASE_URL` | `https://bonusperme.it` | URL base per SEO e OG tags |
| `GTM_ID` | _(vuoto)_ | Google Tag Manager (vuoto = disattivato) |
| `SENTRY_DSN` | _(vuoto)_ | Sentry DSN per error tracking |
| `SCRAPER_ENABLED` | `true` | Abilita scraper automatico |
| `SCRAPER_INTERVAL` | `24h` | Intervallo tra scrape |
| `RATE_LIMIT_RPS` | `30` | Richieste al secondo per IP |
| `RATE_LIMIT_BURST` | `60` | Burst massimo per IP |
| `LINKCHECK_ENABLED` | `true` | Verifica link ufficiali |
| `GZIP_ENABLED` | `true` | Compressione Gzip |
| `TURNSTILE_SITE_KEY` | _(vuoto)_ | Cloudflare Turnstile (anti-bot) |
| `TURNSTILE_SECRET_KEY` | _(vuoto)_ | Cloudflare Turnstile secret |
| `WEB3FORMS_ACCESS_KEY` | _(vuoto)_ | Web3Forms per form contatti |
| `VALIDITY_CHECK_ENABLED` | `true` | Controllo scadenze bonus |
| `NEWS_CHECK_ENABLED` | `false` | Monitoraggio novita normative |
| `ADMIN_API_KEY` | _(vuoto)_ | API key per endpoint admin |

---

## Fonti dati

Il matcher usa dati hardcoded verificati manualmente + arricchimento automatico da:

| Fonte | Tipo | Dati |
|-------|------|------|
| [INPS](https://www.inps.it) | Istituzionale | Assegno Unico, Bonus Nido, ADI, Bonus Mamme |
| [Agenzia delle Entrate](https://www.agenziaentrate.gov.it) | Istituzionale | Detrazioni casa, Ecobonus, Bonus Mobili |
| [MEF](https://www.mef.gov.it) | Istituzionale | Carta Dedicata a Te, misure fiscali |
| [Gazzetta Ufficiale](https://www.gazzettaufficiale.it) | Istituzionale | RSS aggiornamenti normativi |
| Fonti editoriali | Secondaria | Verifiche incrociate (Ti Consiglio, Fisco e Tasse) |

Se una fonte non e raggiungibile, il sistema usa i dati verificati piu recenti (fallback hardcoded).

---

## Privacy

- Nessun database
- Nessun cookie di profilazione (solo consenso cookie tecnico)
- Nessun tracking (GTM opzionale e disattivabile)
- I dati inseriti esistono solo nella sessione browser — cancellati al refresh
- Server in Unione Europea (AWS eu-west-1)
- Codice sorgente aperto e verificabile
- Conforme GDPR

---

## Deploy (produzione)

L'applicazione gira su **AWS Elastic Beanstalk** (Docker, single instance, eu-west-1) con **Cloudflare** davanti (DNS, CDN, WAF, SSL Full mode).

```
.ebextensions/01-ssl.config     # Security group per porta 443
.platform/hooks/predeploy/      # Hook: copia cert SSL, configura nginx
.platform/nginx/conf.d/         # Nginx: HTTPS termination + proxy
ssl/origin-cert.pem             # Cloudflare Origin Certificate (non committato)
ssl/origin-key.pem              # Cloudflare Origin Key (non committato)
```

Per deployare:

```bash
zip -r ../bonusperme.zip . -x ".git/*" -x ".claude/*" -x "__MACOSX/*" -x ".DS_Store"
# Upload bonusperme.zip su Elastic Beanstalk
```

---

## Contribuire

Le contribuzioni sono benvenute. Leggi [CONTRIBUTING.md](CONTRIBUTING.md).

**Aree dove serve aiuto:**

- Aggiunta bonus regionali mancanti
- Nuove lingue
- Segnalazione bonus errati o importi non aggiornati
- Test di accessibilita
- Miglioramento parser per nuove fonti istituzionali

---

## Licenza

[AGPL-3.0](LICENSE) — puoi usare, modificare e ridistribuire liberamente, a condizione che il codice derivato resti open source. Se lo usi come servizio web, devi rendere disponibile il sorgente.

---

## For International Contributors

BonusPerMe is a free, open-source tool that helps families in Italy discover government benefits (bonuses, tax deductions, subsidies) they're entitled to. It supports 7 languages and covers 40+ national and regional benefits. The stack is pure Go with a single-file vanilla HTML/CSS/JS frontend — zero JS frameworks, zero npm dependencies. See [CONTRIBUTING.md](CONTRIBUTING.md) to get started.

---

[Segnala un problema](https://github.com/nogarasimone/bonusperme/issues/new?template=bug_report.md) · [Richiedi una feature](https://github.com/nogarasimone/bonusperme/issues/new?template=feature_request.md) · [Segnala un bonus errato](https://github.com/nogarasimone/bonusperme/issues/new?template=bonus_errato.md)
