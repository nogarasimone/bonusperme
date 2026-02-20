# Changelog

Tutte le modifiche rilevanti al progetto sono documentate in questo file.

Il formato è basato su [Keep a Changelog](https://keepachangelog.com/it-IT/1.1.0/),
e il progetto segue [Semantic Versioning](https://semver.org/lang/it/).

## [1.0.0] — 2025-02-07

### Aggiunto
- Lancio iniziale di BonusPerMe
- Database di 20+ bonus italiani attivi nel 2025
- Wizard a 4 step per la raccolta dati anonima
- Engine di matching con scoring 0-100 per compatibilità
- Calcolo importo personalizzato basato su ISEE reale
- Scraper automatico da 7 fonti istituzionali e giornalistiche (aggiornamento ogni 24h)
- Report PDF professionale scaricabile con fonti e riferimenti normativi
- Stampa ottimizzata per CAF e commercialisti
- Calendario scadenze .ics importabile in Google Calendar / Apple Calendar
- Checklist documenti spuntabile per ogni bonus
- Simulatore "cosa cambierebbe con un ISEE diverso"
- FAQ specifiche per ogni bonus (2-3 domande/risposte)
- Supporto 5 lingue: Italiano, English, Français, Español, Română
- Parsing automatico dell'attestazione ISEE da PDF (con drag & drop)
- Pagine SEO dedicate per ogni bonus
- Sitemap XML per indicizzazione motori di ricerca
- Mappa CAF vicini tramite Google Maps
- Condivisione risultati via WhatsApp (generale e per singolo bonus)
- Accessibilità WCAG AA (navigazione da tastiera, screen reader, contrasti)
- Rate limiting per protezione API (token bucket per IP)
- Analytics privacy-first (zero dati identificativi)
- Contatore verifiche persistente con debounce
- Sezione "Novità 2025" nella landing page
- Testimonial famiglie nella landing page
- Mini calcolatore "quanto lasci sul tavolo" nella hero
- Coming soon: notifiche Telegram/WhatsApp
- Struttura preparatoria bot Telegram
- Dockerfile e docker-compose.yml
- CI/CD con GitHub Actions

### Sicurezza
- Zero database, cookie, tracking o profilazione
- Dati utente solo in sessione, cancellati al refresh
- Server EU, GDPR compliant
- Rate limiting per IP su tutti gli endpoint API
- Validazione input server-side e client-side
- PDF ISEE elaborato in streaming, mai salvato su disco

## [Unreleased]

### In sviluppo
- Bot Telegram @bonuspermeitalia
- Notifiche nuovi bonus via Telegram e WhatsApp
- Bonus regionali (Lombardia, Lazio, Campania, Sicilia)
- Progressive Web App (PWA) per uso offline
- API pubblica con documentazione OpenAPI
- Widget embeddabile per siti CAF e patronati
