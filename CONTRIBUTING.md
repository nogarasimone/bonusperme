# Contribuire a BonusPerMe

Grazie per il tuo interesse nel contribuire a BonusPerMe! Ogni contribuzione — dalla segnalazione di un bug alla traduzione di un bonus — aiuta migliaia di famiglie a scoprire agevolazioni che altrimenti perderebbero.

## Come segnalare un bug

1. Apri una [nuova issue](https://github.com/bonusperme/bonusperme/issues/new?template=bug_report.md) usando il template "Bug report"
2. Descrivi il problema nel modo più dettagliato possibile
3. Includi i passaggi per riprodurlo
4. Se possibile, aggiungi uno screenshot

## Come proporre una feature

1. Apri una [nuova issue](https://github.com/bonusperme/bonusperme/issues/new?template=feature_request.md) usando il template "Feature request"
2. Spiega il problema che la feature risolverebbe
3. Descrivi la soluzione che proponi
4. Indica eventuali alternative che hai considerato

## Come segnalare un bonus errato o mancante

1. Apri una [nuova issue](https://github.com/bonusperme/bonusperme/issues/new?template=bonus_errato.md) usando il template "Bonus errato"
2. Indica il nome del bonus e cosa è impreciso
3. Fornisci la fonte ufficiale con il link

## Come contribuire codice

### 1. Fork e setup

```bash
# Fork il repository su GitHub, poi:
git clone https://github.com/TUO-USERNAME/bonusperme.git
cd bonusperme
go mod download
go build .
```

### 2. Crea un branch

Usa una naming convention chiara:

```bash
git checkout -b feature/nome-feature   # per nuove funzionalità
git checkout -b fix/nome-bug           # per correzioni
git checkout -b docs/descrizione       # per documentazione
git checkout -b i18n/lingua            # per traduzioni
```

### 3. Scrivi il codice

**Convenzioni:**

- Formatta con `go fmt ./...`
- Verifica con `go vet ./...`
- Commenti in **italiano** per la logica di business (bonus, requisiti, matching)
- Commenti in **inglese** per il codice tecnico (HTTP, parsing, cache)
- Segui le convenzioni Go standard (nomi esportati in PascalCase, interni in camelCase)

**Struttura:**

- Handler HTTP → `internal/handlers/`
- Logica bonus e matching → `internal/matcher/`
- Scraper e fonti → `internal/scraper/`
- Traduzioni → `internal/i18n/`
- Frontend → `static/index.html` (singolo file)

### 4. Testa

```bash
go build ./...
go vet ./...
go test ./... -v  # quando ci saranno test
```

Se aggiungi logica nel matcher o nello scraper, scrivi un test.

### 5. Commit e Pull Request

**Convenzioni commit** — usiamo [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: aggiunge bonus regionale Lombardia
fix: corregge importo assegno unico per ISEE > 40k
docs: aggiorna README con nuove fonti
i18n: aggiunge traduzione tedesca
chore: aggiorna dipendenze Go
```

Apri una Pull Request con:
- Titolo chiaro e conciso
- Descrizione di cosa cambia e perché
- Link alla issue correlata (se esiste)

## Aree dove serve aiuto

Cerchiamo contributi in queste aree:

- **Traduzioni** — aggiungere nuove lingue o migliorare quelle esistenti
- **Bonus regionali** — ogni regione ha bonus specifici da mappare
- **Bonus comunali** — grandi città (Roma, Milano, Napoli, Torino) hanno agevolazioni locali
- **Parser scraper** — nuove fonti istituzionali da integrare
- **Test di accessibilità** — verifiche con screen reader e navigazione da tastiera
- **Segnalazione errori** — importi cambiati, scadenze aggiornate, requisiti modificati
- **Documentazione** — guide, tutorial, FAQ

## Code of Conduct

Questo progetto adotta il [Contributor Covenant](https://www.contributor-covenant.org/version/2/1/code_of_conduct/) come codice di condotta. Partecipando, ti impegni a mantenere un ambiente accogliente e rispettoso per tutti.

---

## For International Contributors

BonusPerMe welcomes contributions from developers worldwide. The codebase is in Go with a single-file vanilla HTML/CSS/JS frontend.

**Quick guide:**

1. Fork the repository
2. Create a branch (`feature/your-feature` or `fix/your-fix`)
3. Follow Go conventions (`go fmt`, `go vet`)
4. Business logic comments in Italian, technical comments in English
5. Open a Pull Request with a clear description

**Translation contributions** are especially welcome — see `internal/i18n/translations.go` for the translation format. Each language needs ~100 keys translated.

If you need help understanding the Italian codebase, feel free to open an issue in English — we'll be happy to help.
