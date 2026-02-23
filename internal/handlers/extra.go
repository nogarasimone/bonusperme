package handlers

import (
	"bonusperme/internal/linkcheck"
	"bonusperme/internal/matcher"
	"bonusperme/internal/models"
	"bonusperme/internal/scraper"
	sentryutil "bonusperme/internal/sentry"
	"bonusperme/internal/validity"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// ---------- helpers ----------

var italianMonths = map[string]int{
	"gennaio":   1,
	"febbraio":  2,
	"marzo":     3,
	"aprile":    4,
	"maggio":    5,
	"giugno":    6,
	"luglio":    7,
	"agosto":    8,
	"settembre": 9,
	"ottobre":   10,
	"novembre":  11,
	"dicembre":  12,
}

var italianDateRe = regexp.MustCompile(
	`(\d{1,2})\s+(gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)\s+(\d{4})`,
)

var slashDateRe = regexp.MustCompile(`(\d{2})/(\d{2})/(\d{4})`)

func parseItalianDate(s string) time.Time {
	lower := strings.ToLower(s)

	if m := italianDateRe.FindStringSubmatch(lower); len(m) == 4 {
		day, _ := strconv.Atoi(m[1])
		month := italianMonths[m[2]]
		year, _ := strconv.Atoi(m[3])
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	}

	if m := slashDateRe.FindStringSubmatch(s); len(m) == 4 {
		day, _ := strconv.Atoi(m[1])
		month, _ := strconv.Atoi(m[2])
		year, _ := strconv.Atoi(m[3])
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	}

	return time.Date(time.Now().Year(), 12, 31, 0, 0, 0, 0, time.UTC)
}

func slugifyName(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, s)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}

func transliterate(s string) string {
	replacer := strings.NewReplacer(
		"à", "a", "è", "e", "é", "e", "ì", "i", "ò", "o", "ù", "u",
		"À", "A", "È", "E", "É", "E", "Ì", "I", "Ò", "O", "Ù", "U",
		"\u2264", "<=", "\u2265", ">=",
		"€", "EUR ", "–", "-", "\u2018", "'", "\u2019", "'",
		"\u201C", "\"", "\u201D", "\"",
	)
	return replacer.Replace(s)
}

func parseEuroAmount(s string) float64 {
	re := regexp.MustCompile(`[0-9][0-9.,]*`)
	m := re.FindString(s)
	if m == "" {
		return 0
	}
	m = strings.ReplaceAll(m, ".", "")
	m = strings.Replace(m, ",", ".", 1)
	v, _ := strconv.ParseFloat(m, 64)
	return v
}

// ---------- validateProfile ----------

// Whitelists for enum fields
var validResidenza = map[string]bool{
	"": true, "Abruzzo": true, "Basilicata": true, "Calabria": true,
	"Campania": true, "Emilia-Romagna": true, "Friuli-Venezia Giulia": true,
	"Lazio": true, "Liguria": true, "Lombardia": true, "Marche": true,
	"Molise": true, "Piemonte": true, "Puglia": true, "Sardegna": true,
	"Sicilia": true, "Toscana": true, "Trentino-Alto Adige": true,
	"Umbria": true, "Valle d'Aosta": true, "Veneto": true,
}

var validStatoCivile = map[string]bool{
	"": true, "celibe/nubile": true, "coniugato/a": true,
	"convivente": true, "separato/a": true, "divorziato/a": true,
	"vedovo/a": true, "unione civile": true,
}

var validOccupazione = map[string]bool{
	"": true, "dipendente": true, "autonomo": true,
	"disoccupato": true, "pensionato": true, "studente": true,
	"casalinga": true, "inoccupato": true,
}

var validDisabilitaFigli = map[string]bool{
	"": true, "media": true, "grave": true, "non_autosufficienza": true,
}

func validateProfile(p models.UserProfile) (string, bool) {
	if p.Eta < 18 || p.Eta > 120 {
		return "Eta non valida (18-120)", false
	}
	if p.ISEE < 0 || p.ISEE > 500000 {
		return "ISEE non valido (0-500000)", false
	}
	if p.RedditoAnnuo < 0 || p.RedditoAnnuo > 1000000 {
		return "Reddito annuo non valido (0-1000000)", false
	}
	if p.NumeroFigli < 0 || p.NumeroFigli > 20 {
		return "Numero figli non valido (0-20)", false
	}
	if p.FigliMinorenni < 0 || p.FigliMinorenni > 20 {
		return "Figli minorenni non valido (0-20)", false
	}
	if p.FigliUnder3 < 0 || p.FigliUnder3 > 20 {
		return "Figli under 3 non valido (0-20)", false
	}
	if p.FigliUnder1 < 0 || p.FigliUnder1 > 20 {
		return "Figli under 1 non valido (0-20)", false
	}
	if p.FigliMaggiorenni < 0 || p.FigliMaggiorenni > 20 {
		return "Figli maggiorenni non valido (0-20)", false
	}
	if p.FigliDisabili < 0 || p.FigliDisabili > 20 {
		return "Figli disabili non valido (0-20)", false
	}
	if p.Over65 < 0 || p.Over65 > 10 {
		return "Over 65 non valido (0-10)", false
	}
	// Cross-field checks
	if p.FigliUnder1 > p.FigliUnder3 {
		return "Figli under 1 non puo superare figli under 3", false
	}
	if p.FigliUnder3 > p.FigliMinorenni {
		return "Figli under 3 non puo superare figli minorenni", false
	}
	if p.FigliMinorenni+p.FigliMaggiorenni > p.NumeroFigli {
		return "Figli minorenni + maggiorenni non puo superare numero figli", false
	}
	if p.FigliDisabili > p.NumeroFigli {
		return "Figli disabili non puo superare numero figli", false
	}
	// Whitelist checks
	if !validDisabilitaFigli[p.DisabilitaFigli] {
		return "Grado disabilita figli non valido", false
	}
	if !validResidenza[p.Residenza] {
		return "Regione non valida", false
	}
	if !validStatoCivile[p.StatoCivile] {
		return "Stato civile non valido", false
	}
	if !validOccupazione[p.Occupazione] {
		return "Occupazione non valida", false
	}
	return "", true
}

// ---------- 1. CalendarHandler ----------

func CalendarHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	raw := r.URL.Query().Get("bonuses")
	if raw == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var items []struct {
		Nome     string `json:"nome"`
		Scadenza string `json:"scadenza"`
	}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		http.Error(w, "Invalid bonuses JSON", http.StatusBadRequest)
		return
	}

	if len(items) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	now := time.Now().UTC().Format("20060102T150405Z")

	var sb strings.Builder
	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString("PRODID:-//BonusPerMe//IT\r\n")
	sb.WriteString("CALSCALE:GREGORIAN\r\n")
	sb.WriteString("METHOD:PUBLISH\r\n")

	validCount := 0
	for _, item := range items {
		if item.Nome == "" {
			continue
		}
		validCount++
		dt := parseItalianDate(item.Scadenza)
		dtStr := dt.Format("20060102")
		uid := slugifyName(item.Nome) + "@bonusperme.it"

		sb.WriteString("BEGIN:VEVENT\r\n")
		sb.WriteString("UID:" + uid + "\r\n")
		sb.WriteString("DTSTAMP:" + now + "\r\n")
		sb.WriteString("DTSTART;VALUE=DATE:" + dtStr + "\r\n")
		sb.WriteString("DTEND;VALUE=DATE:" + dtStr + "\r\n")
		sb.WriteString("SUMMARY:Scadenza: " + item.Nome + "\r\n")
		sb.WriteString("DESCRIPTION:Ricorda di presentare domanda per " + item.Nome + " prima della scadenza. Verifica requisiti su BonusPerMe.\r\n")
		sb.WriteString("BEGIN:VALARM\r\n")
		sb.WriteString("TRIGGER:-P7D\r\n")
		sb.WriteString("ACTION:DISPLAY\r\n")
		sb.WriteString("DESCRIPTION:Scadenza tra 7 giorni: " + item.Nome + "\r\n")
		sb.WriteString("END:VALARM\r\n")
		sb.WriteString("BEGIN:VALARM\r\n")
		sb.WriteString("TRIGGER:-P1D\r\n")
		sb.WriteString("ACTION:DISPLAY\r\n")
		sb.WriteString("DESCRIPTION:Scadenza domani: " + item.Nome + "\r\n")
		sb.WriteString("END:VALARM\r\n")
		sb.WriteString("END:VEVENT\r\n")
	}

	sb.WriteString("END:VCALENDAR\r\n")

	if validCount == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="bonusperme-scadenze.ics"`)
	w.Write([]byte(sb.String()))
}

// ---------- 2. SimulateHandler ----------

func SimulateHandler(w http.ResponseWriter, r *http.Request) {
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

	if msg, ok := validateProfile(profile); !ok {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	cachedBonus := scraper.GetCachedBonus()

	reale := matcher.MatchBonus(profile, cachedBonus)
	linkcheck.ApplyStatus(reale.Bonus)
	validity.ApplyStatus(reale.Bonus)
	reale.Avvisi = validity.GenerateAvvisi(reale.Bonus)

	simProfile := profile
	simProfile.ISEE = profile.ISEESimulato
	simulato := matcher.MatchBonus(simProfile, cachedBonus)
	linkcheck.ApplyStatus(simulato.Bonus)
	validity.ApplyStatus(simulato.Bonus)
	simulato.Avvisi = validity.GenerateAvvisi(simulato.Bonus)

	bonusExtra := simulato.BonusTrovati - reale.BonusTrovati
	if bonusExtra < 0 {
		bonusExtra = 0
	}

	risparmioReale := parseEuroAmount(reale.RisparmioStimato)
	risparmioSim := parseEuroAmount(simulato.RisparmioStimato)
	extraVal := risparmioSim - risparmioReale
	if extraVal < 0 {
		extraVal = 0
	}

	result := models.SimulateResult{
		Reale:          reale,
		Simulato:       simulato,
		BonusExtra:     bonusExtra,
		RisparmioExtra: fmt.Sprintf("EUR %.0f", extraVal),
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(result)
}


// =====================================================================
// 3. ReportHandler — Professional PDF Report v2
// =====================================================================
// Design: Swiss-style typography, left accent bars, generous whitespace.
// Replaces the old card-border layout with a clean document design.

// PDF design system colors
var (
	cBlue    = [3]int{27, 58, 84}
	cBlueLt  = [3]int{44, 82, 110}
	cBlueMid = [3]int{44, 95, 124}
	cTerra   = [3]int{192, 82, 46}
	cGreen   = [3]int{42, 107, 69}
	cGreenBg = [3]int{233, 245, 237}
	cAmber   = [3]int{154, 123, 46}
	cAmberBg = [3]int{250, 244, 230}
	cCream   = [3]int{248, 247, 243}
	cInk90   = [3]int{38, 38, 38}
	cInk75   = [3]int{64, 64, 64}
	cInk50   = [3]int{107, 107, 107}
	cInk30   = [3]int{160, 160, 160}
	cInk15   = [3]int{217, 217, 217}
	cInk08   = [3]int{235, 235, 235}
	cRed     = [3]int{200, 50, 50}
	cRedBg   = [3]int{254, 235, 235}
	cWhite   = [3]int{255, 255, 255}
)

const (
	pageW    = 210.0
	pageH    = 297.0
	marginL  = 20.0
	marginR  = 20.0
	marginT  = 20.0
	contentW = pageW - marginL - marginR // 170mm
)

func setFill(pdf *gofpdf.Fpdf, c [3]int)  { pdf.SetFillColor(c[0], c[1], c[2]) }
func setText(pdf *gofpdf.Fpdf, c [3]int)   { pdf.SetTextColor(c[0], c[1], c[2]) }
func setDraw(pdf *gofpdf.Fpdf, c [3]int)   { pdf.SetDrawColor(c[0], c[1], c[2]) }

func fmtEuro(amount float64) string {
	if amount == 0 {
		return "0"
	}
	neg := amount < 0
	if neg {
		amount = -amount
	}
	whole := int(amount)
	frac := int(math.Round((amount - float64(whole)) * 100))
	s := addDotSep(fmt.Sprintf("%d", whole))
	prefix := ""
	if neg {
		prefix = "-"
	}
	if frac > 0 {
		return fmt.Sprintf("%s%s,%02d", prefix, s, frac)
	}
	return prefix + s
}

func addDotSep(s string) string {
	n := len(s)
	if n <= 3 {
		return s
	}
	return addDotSep(s[:n-3]) + "." + s[n-3:]
}

func compatColor(pct int) ([3]int, [3]int) {
	if pct >= 80 {
		return cGreen, cGreenBg
	}
	if pct >= 50 {
		return cAmber, cAmberBg
	}
	return cInk30, [3]int{240, 240, 240}
}

func truncURL(url string, max int) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "www.")
	if len(url) > max {
		return url[:max-3] + "..."
	}
	return url
}

// ensureSpace checks if there's enough room; if not, adds a page.
func ensureSpace(pdf *gofpdf.Fpdf, needed float64) float64 {
	y := pdf.GetY()
	if y+needed > pageH-25 {
		pdf.AddPage()
		return marginT + 10 // below header
	}
	return y
}

// drawAccentBar draws a vertical colored bar on the left side of a bonus section.
func drawAccentBar(pdf *gofpdf.Fpdf, x, startY, endY float64, c [3]int) {
	setFill(pdf, c)
	pdf.Rect(x, startY, 2.5, endY-startY, "F")
}

// drawPill draws a rounded pill label.
func drawPill(pdf *gofpdf.Fpdf, x, y float64, text string, bg, fg [3]int) float64 {
	pdf.SetFont("Helvetica", "B", 7.5)
	w := pdf.GetStringWidth(transliterate(text)) + 8
	setFill(pdf, bg)
	pdf.RoundedRect(x, y, w, 5.5, 2.5, "1234", "F")
	setText(pdf, fg)
	pdf.SetXY(x, y+0.5)
	pdf.CellFormat(w, 5, transliterate(text), "", 0, "C", false, 0, "")
	return w
}

// drawStepNum draws a numbered circle for steps.
func drawStepNum(pdf *gofpdf.Fpdf, x, y float64, num int) {
	setFill(pdf, cBlue)
	pdf.Circle(x+2, y+2, 2.8, "F")
	pdf.SetFont("Helvetica", "B", 7)
	setText(pdf, cWhite)
	pdf.SetXY(x-0.5, y-0.2)
	pdf.CellFormat(5, 4.5, fmt.Sprintf("%d", num), "", 0, "C", false, 0, "")
}

// drawCheckmark draws a small green checkmark icon.
func drawCheckmark(pdf *gofpdf.Fpdf, x, y float64) {
	setDraw(pdf, cGreen)
	pdf.SetLineWidth(0.4)
	pdf.Line(x+0.3, y+1.8, x+1.2, y+2.8)
	pdf.Line(x+1.2, y+2.8, x+3, y+0.8)
}

// drawSquare draws a small empty checkbox.
func drawSquare(pdf *gofpdf.Fpdf, x, y float64) {
	setDraw(pdf, cInk30)
	pdf.SetLineWidth(0.25)
	pdf.Rect(x, y+0.3, 3, 3, "D")
}

// ReportHandler generates a professional PDF report.
func ReportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var profile models.UserProfile
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "application/x-www-form-urlencoded") || strings.Contains(ct, "multipart/form-data") {
		r.ParseForm()
		dataStr := r.FormValue("data")
		if dataStr == "" {
			http.Error(w, "Missing data field", http.StatusBadRequest)
			return
		}
		if err := json.Unmarshal([]byte(dataStr), &profile); err != nil {
			http.Error(w, "Invalid profile data", http.StatusBadRequest)
			return
		}
	} else {
		if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
	}

	if msg, ok := validateProfile(profile); !ok {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	cachedBonus := scraper.GetCachedBonus()
	result := matcher.MatchBonus(profile, cachedBonus)
	linkcheck.ApplyStatus(result.Bonus)
	validity.ApplyStatus(result.Bonus)
	result.Avvisi = validity.GenerateAvvisi(result.Bonus)

	profileCode := "BPM-..."
	compact := toCompact(profile)
	if data, err := json.Marshal(compact); err == nil {
		b64 := base64.RawURLEncoding.EncodeToString(data)
		code := codePrefix + b64
		if len(code) > 64 {
			code = code[:64]
		}
		profileCode = code
	}

	now := time.Now()
	dateStr := now.Format("2006-01-02")
	dateDisplay := now.Format("02/01/2006")

	var activeBonuses, expiredBonuses []models.Bonus
	for _, b := range result.Bonus {
		if b.Scaduto {
			expiredBonuses = append(expiredBonuses, b)
		} else {
			activeBonuses = append(activeBonuses, b)
		}
	}

	risparmioVal := parseEuroAmount(result.RisparmioStimato)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(marginL, 15, marginR)
	pdf.SetAutoPageBreak(false, 20)

	isFirstPage := true

	// ─── Footer on every page ───
	pdf.SetFooterFunc(func() {
		pdf.SetY(-14)
		setDraw(pdf, cInk08)
		pdf.SetLineWidth(0.3)
		pdf.Line(marginL, pdf.GetY(), pageW-marginR, pdf.GetY())
		pdf.SetY(-11)
		pdf.SetFont("Helvetica", "", 6.5)
		setText(pdf, cInk30)
		pdf.SetX(marginL)
		pdf.CellFormat(contentW/2, 8, "bonusperme.it", "", 0, "L", false, 0, "")
		pdf.CellFormat(contentW/2, 8, fmt.Sprintf("%d", pdf.PageNo()), "", 0, "R", false, 0, "")
	})

	// ─── Header on pages 2+ ───
	pdf.SetHeaderFunc(func() {
		if isFirstPage {
			return
		}
		pdf.SetY(8)
		pdf.SetX(marginL)
		pdf.SetFont("Helvetica", "B", 8)
		setText(pdf, cBlue)
		pdf.CellFormat(contentW/2, 4, "BonusPerMe", "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 7)
		setText(pdf, cInk30)
		pdf.CellFormat(contentW/2, 4, transliterate("Report del "+dateDisplay), "", 0, "R", false, 0, "")
		setDraw(pdf, cBlue)
		pdf.SetLineWidth(0.5)
		pdf.Line(marginL, 13.5, pageW-marginR, 13.5)
	})

	// ═════════════════════════════════════════════════════════════
	// PAGE 1 — COVER
	// ═════════════════════════════════════════════════════════════
	pdf.AddPage()

	// ── Blue header band (full width, 62mm) ──
	headerH := 62.0
	setFill(pdf, cBlue)
	pdf.Rect(0, 0, pageW, headerH, "F")

	// Subtle lighter stripe at bottom of header
	setFill(pdf, cBlueLt)
	pdf.Rect(0, headerH-3, pageW, 3, "F")

	// Title
	pdf.SetXY(marginL, 18)
	pdf.SetFont("Helvetica", "B", 28)
	setText(pdf, cWhite)
	pdf.CellFormat(contentW, 10, "BonusPerMe", "", 1, "L", false, 0, "")

	// Decorative thin line
	pdf.SetXY(marginL, 30)
	pdf.SetDrawColor(255, 255, 255)
	pdf.SetLineWidth(0.3)
	pdf.Line(marginL, 31, marginL+35, 31)

	// Subtitle
	pdf.SetXY(marginL, 34)
	pdf.SetFont("Helvetica", "", 12)
	pdf.SetTextColor(210, 220, 230)
	pdf.CellFormat(contentW, 6, "Report Personalizzato", "", 1, "L", false, 0, "")

	// Date
	pdf.SetXY(marginL, 42)
	pdf.SetFont("Helvetica", "", 8.5)
	pdf.SetTextColor(170, 185, 200)
	pdf.CellFormat(contentW, 5, transliterate("Generato il "+dateDisplay), "", 1, "L", false, 0, "")

	// ── Summary metrics (overlapping header) ──
	cardW := 150.0
	cardX := (pageW - cardW) / 2
	cardY := headerH - 12
	cardH := 30.0

	// Card shadow
	setFill(pdf, [3]int{210, 210, 210})
	pdf.RoundedRect(cardX+1.5, cardY+1.5, cardW, cardH, 4, "1234", "F")
	// Card background
	setFill(pdf, cWhite)
	pdf.RoundedRect(cardX, cardY, cardW, cardH, 4, "1234", "F")

	// Left metric: bonus count
	pdf.SetXY(cardX+12, cardY+5)
	pdf.SetFont("Courier", "B", 32)
	setText(pdf, cBlue)
	pdf.CellFormat(cardW/2-12, 10, fmt.Sprintf("%d", result.BonusAttivi), "", 0, "L", false, 0, "")

	pdf.SetXY(cardX+12, cardY+19)
	pdf.SetFont("Helvetica", "", 8.5)
	setText(pdf, cInk50)
	label := "bonus trovati"
	if result.BonusScaduti > 0 {
		label += fmt.Sprintf("  (+%d scadut", result.BonusScaduti)
		if result.BonusScaduti == 1 {
			label += "o)"
		} else {
			label += "i)"
		}
	}
	pdf.CellFormat(cardW/2-12, 4, transliterate(label), "", 0, "L", false, 0, "")

	// Vertical divider in card
	setDraw(pdf, cInk15)
	pdf.SetLineWidth(0.3)
	pdf.Line(cardX+cardW/2, cardY+6, cardX+cardW/2, cardY+cardH-6)

	// Right metric: risparmio
	euroStr := fmtEuro(risparmioVal)
	pdf.SetXY(cardX+cardW/2+8, cardY+5)
	pdf.SetFont("Courier", "B", 32)
	if len(euroStr) > 6 {
		pdf.SetFont("Courier", "B", 26)
	}
	setText(pdf, cGreen)
	pdf.CellFormat(cardW/2-20, 10, transliterate("EUR "+euroStr), "", 0, "R", false, 0, "")

	pdf.SetXY(cardX+cardW/2+8, cardY+19)
	pdf.SetFont("Helvetica", "", 8.5)
	setText(pdf, cInk50)
	pdf.CellFormat(cardW/2-20, 4, "risparmio stimato / anno", "", 0, "R", false, 0, "")

	// ── Profile section ──
	profStartY := cardY + cardH + 14
	pdf.SetY(profStartY)
	pdf.SetX(marginL)
	pdf.SetFont("Helvetica", "B", 7)
	setText(pdf, cInk30)
	pdf.CellFormat(contentW, 4, "PROFILO", "", 1, "L", false, 0, "")
	pdf.Ln(3)

	profBoxY := pdf.GetY()
	profBoxH := 28.0
	setFill(pdf, cCream)
	pdf.RoundedRect(marginL, profBoxY, contentW, profBoxH, 3, "1234", "F")

	colW := contentW / 3
	row1Y := profBoxY + 5
	row2Y := profBoxY + 16

	// Row 1: Eta, ISEE, Regione
	profileCell(pdf, marginL+6, row1Y, colW, "Eta", fmt.Sprintf("%d anni", profile.Eta))
	iseeStr := fmtEuro(profile.ISEE)
	profileCell(pdf, marginL+6+colW, row1Y, colW, "ISEE", transliterate("EUR "+iseeStr))
	regioneVal := profile.Residenza
	if regioneVal == "" {
		regioneVal = "-"
	}
	profileCell(pdf, marginL+6+colW*2, row1Y, colW, "Regione", transliterate(regioneVal))

	// Row 2: Figli, Occupazione, Stato civile
	figliStr := fmt.Sprintf("%d", profile.NumeroFigli)
	if profile.FigliMinorenni > 0 {
		figliStr += fmt.Sprintf(" (%d min.)", profile.FigliMinorenni)
	}
	profileCell(pdf, marginL+6, row2Y, colW, "Figli", figliStr)
	occVal := profile.Occupazione
	if occVal == "" {
		occVal = "-"
	}
	profileCell(pdf, marginL+6+colW, row2Y, colW, "Occupazione", transliterate(occVal))
	civVal := profile.StatoCivile
	if civVal == "" {
		civVal = "-"
	}
	profileCell(pdf, marginL+6+colW*2, row2Y, colW, "Stato civile", transliterate(civVal))

	// ── Panoramica ──
	pdf.SetY(profBoxY + profBoxH + 10)
	pdf.SetX(marginL)
	pdf.SetFont("Helvetica", "B", 7)
	setText(pdf, cInk30)
	pdf.CellFormat(contentW, 4, "PANORAMICA", "", 1, "L", false, 0, "")
	pdf.Ln(4)

	// Table header line
	tableStartY := pdf.GetY()
	setDraw(pdf, cBlue)
	pdf.SetLineWidth(0.5)
	pdf.Line(marginL, tableStartY, pageW-marginR, tableStartY)
	pdf.SetY(tableStartY + 2)

	// Active bonuses in clean table rows
	for i, b := range activeBonuses {
		y := pdf.GetY()
		fg, _ := compatColor(b.Compatibilita)

		// Alternating row background
		if i%2 == 0 {
			setFill(pdf, cCream)
			pdf.Rect(marginL, y-0.5, contentW, 6.5, "F")
		}

		// Colored dot
		setFill(pdf, fg)
		pdf.Circle(marginL+3.5, y+2.2, 1.3, "F")

		// Name
		pdf.SetXY(marginL+8, y)
		pdf.SetFont("Helvetica", "", 8.5)
		setText(pdf, cInk75)
		nome := b.Nome
		if len(nome) > 50 {
			nome = nome[:47] + "..."
		}
		pdf.CellFormat(contentW-55, 5.5, transliterate(nome), "", 0, "L", false, 0, "")

		// Importo right-aligned
		pdf.SetFont("Courier", "", 8)
		setText(pdf, cBlue)
		importoDisplay := transliterate(b.Importo)
		if importoDisplay == "" {
			importoDisplay = "-"
		}
		if len(importoDisplay) > 45 {
			importoDisplay = importoDisplay[:42] + "..."
		}
		pdf.CellFormat(47, 5.5, importoDisplay, "", 1, "R", false, 0, "")
		pdf.SetY(y + 6.5)
	}

	// Bottom line
	tableEndY := pdf.GetY()
	setDraw(pdf, cInk15)
	pdf.SetLineWidth(0.3)
	pdf.Line(marginL, tableEndY, pageW-marginR, tableEndY)

	// Expired bonuses below
	if len(expiredBonuses) > 0 {
		pdf.SetY(tableEndY + 3)
		for _, b := range expiredBonuses {
			y := pdf.GetY()
			// Red X
			setFill(pdf, cRed)
			pdf.Circle(marginL+3.5, y+2.2, 1.3, "F")
			setText(pdf, cWhite)
			pdf.SetFont("Helvetica", "B", 5)
			pdf.SetXY(marginL+2, y+0.5)
			pdf.CellFormat(3, 3.5, "x", "", 0, "C", false, 0, "")

			// Name in grey
			pdf.SetXY(marginL+8, y)
			pdf.SetFont("Helvetica", "", 8.5)
			setText(pdf, cInk30)
			pdf.CellFormat(contentW-40, 5.5, transliterate(b.Nome), "", 0, "L", false, 0, "")

			// SCADUTO
			pdf.SetFont("Helvetica", "B", 7)
			setText(pdf, cRed)
			pdf.CellFormat(32, 5.5, "SCADUTO", "", 1, "R", false, 0, "")
			pdf.SetY(y + 6.5)
		}
	}

	// Legend
	pdf.Ln(3)
	legendY := pdf.GetY()
	pdf.SetFont("Helvetica", "", 6.5)
	setText(pdf, cInk30)

	xLeg := marginL
	setFill(pdf, cGreen)
	pdf.Circle(xLeg+1.5, legendY+1.5, 1, "F")
	pdf.SetXY(xLeg+4, legendY)
	pdf.CellFormat(15, 3, "alta", "", 0, "L", false, 0, "")
	xLeg += 18

	setFill(pdf, cAmber)
	pdf.Circle(xLeg+1.5, legendY+1.5, 1, "F")
	pdf.SetXY(xLeg+4, legendY)
	pdf.CellFormat(15, 3, "media", "", 0, "L", false, 0, "")
	xLeg += 18

	setFill(pdf, cInk30)
	pdf.Circle(xLeg+1.5, legendY+1.5, 1, "F")
	pdf.SetXY(xLeg+4, legendY)
	pdf.CellFormat(15, 3, "bassa", "", 0, "L", false, 0, "")
	xLeg += 18

	if len(expiredBonuses) > 0 {
		setFill(pdf, cRed)
		pdf.Circle(xLeg+1.5, legendY+1.5, 1, "F")
		pdf.SetXY(xLeg+4, legendY)
		pdf.CellFormat(15, 3, "scaduto", "", 0, "L", false, 0, "")
	}

	// Cover footer
	pdf.SetY(pageH - 22)
	setDraw(pdf, cInk15)
	pdf.SetLineWidth(0.2)
	pdf.Line(marginL, pdf.GetY(), pageW-marginR, pdf.GetY())
	pdf.SetY(pageH - 19)
	pdf.SetX(marginL)
	pdf.SetFont("Helvetica", "", 6.5)
	setText(pdf, cInk30)
	pdf.CellFormat(contentW/2, 4, transliterate("Documento orientativo -- non sostituisce consulenza professionale"), "", 0, "L", false, 0, "")
	pdf.CellFormat(contentW/2, 4, transliterate(profileCode), "", 0, "R", false, 0, "")

	isFirstPage = false

	// ═════════════════════════════════════════════════════════════
	// PAGES 2+ — BONUS DETAILS
	// ═════════════════════════════════════════════════════════════

	// Disclaimer
	y := ensureSpace(pdf, 16)
	pdf.SetY(y)
	pdf.SetX(marginL)
	setFill(pdf, [3]int{255, 251, 235})
	pdf.RoundedRect(marginL, pdf.GetY(), contentW, 12, 2, "1234", "F")
	// Left amber accent
	setFill(pdf, [3]int{245, 158, 11})
	pdf.Rect(marginL, pdf.GetY(), 2.5, 12, "F")
	pdf.SetX(marginL + 6)
	pdf.SetFont("Helvetica", "", 7)
	pdf.SetTextColor(146, 64, 14)
	pdf.MultiCell(contentW-10, 3.5, transliterate("Risultati orientativi. Importi, requisiti e scadenze possono variare. Verifica sempre sui siti ufficiali (INPS, Agenzia delle Entrate, Regione) prima di fare domanda."), "", "L", false)
	pdf.Ln(5)

	// Active bonus details
	for _, b := range activeBonuses {
		drawBonusDetail(pdf, b, profile)
	}

	// Expired bonuses (compact section)
	if len(expiredBonuses) > 0 {
		y := ensureSpace(pdf, 20+float64(len(expiredBonuses))*14)
		pdf.SetY(y)

		// Section header
		pdf.SetX(marginL)
		pdf.SetFont("Helvetica", "B", 10)
		setText(pdf, cInk30)
		pdf.CellFormat(contentW, 6, "Bonus non attivi", "", 1, "L", false, 0, "")
		pdf.Ln(3)

		for _, b := range expiredBonuses {
			drawBonusExpired(pdf, b)
		}
	}

	// ═════════════════════════════════════════════════════════════
	// LAST PAGE — PROSSIMI PASSI + LEGAL
	// ═════════════════════════════════════════════════════════════
	ensureSpace(pdf, 140)
    y = ensureSpace(pdf, 140)
    if y < 24 {
        y = 24
    }
    pdf.SetY(y)

	// Section title
	pdf.SetX(marginL)
	pdf.SetFont("Helvetica", "B", 18)
	setText(pdf, cBlue)
	pdf.CellFormat(contentW, 10, "Prossimi passi", "", 1, "L", false, 0, "")

	// Thin blue accent line
	setDraw(pdf, cTerra)
	pdf.SetLineWidth(0.8)
	pdf.Line(marginL, pdf.GetY()+1, marginL+30, pdf.GetY()+1)
	pdf.Ln(8)

	nextSteps := []struct {
		Num   string
		Title string
		Desc  string
	}{
		{"1", "Verifica i requisiti", "Controlla ogni bonus sui siti ufficiali indicati nelle pagine precedenti."},
		{"2", "Prepara i documenti", "ISEE aggiornato, SPID o CIE, documento d'identita, coordinate bancarie."},
		{"3", "Presenta le domande", "Online sui portali ufficiali (INPS, Agenzia delle Entrate) o presso un CAF/patronato."},
	}

	for _, step := range nextSteps {
		stepY := pdf.GetY()

		// Number circle
		setFill(pdf, cBlue)
		pdf.Circle(marginL+6, stepY+5, 5, "F")
		pdf.SetFont("Helvetica", "B", 14)
		setText(pdf, cWhite)
		pdf.SetXY(marginL+2, stepY+1.5)
		pdf.CellFormat(8, 7, step.Num, "", 0, "C", false, 0, "")

		// Title
		pdf.SetXY(marginL+16, stepY+1)
		pdf.SetFont("Helvetica", "B", 11)
		setText(pdf, cInk90)
		pdf.CellFormat(contentW-20, 5, transliterate(step.Title), "", 1, "L", false, 0, "")

		// Description
		pdf.SetXY(marginL+16, stepY+7.5)
		pdf.SetFont("Helvetica", "", 8.5)
		setText(pdf, cInk50)
		pdf.CellFormat(contentW-20, 4.5, transliterate(step.Desc), "", 1, "L", false, 0, "")

		pdf.SetY(stepY + 18)
	}

	// ── Legal section ──
	pdf.Ln(8)
	sepY := pdf.GetY()
	setDraw(pdf, cBlue)
	pdf.SetLineWidth(0.6)
	pdf.Line(marginL, sepY, pageW-marginR, sepY)
	pdf.Ln(8)

	// Brand
	pdf.SetX(marginL)
	pdf.SetFont("Helvetica", "B", 14)
	setText(pdf, cBlue)
	pdf.CellFormat(contentW, 7, "BonusPerMe", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 8.5)
	setText(pdf, cInk50)
	pdf.CellFormat(contentW, 5, "bonusperme.it", "", 1, "C", false, 0, "")
	pdf.Ln(4)

	// Legal info
	pdf.SetFont("Helvetica", "", 7.5)
	setText(pdf, cInk30)
	pdf.CellFormat(contentW, 4, "Simone Nogara", "", 1, "C", false, 0, "")
	pdf.CellFormat(contentW, 4, "P.IVA 03817020138 -- C.F. NGRSMN91P14C933V", "", 1, "C", false, 0, "")
	pdf.CellFormat(contentW, 4, "Via Morazzone 4, 22100 Como (CO), Italia", "", 1, "C", false, 0, "")
	pdf.Ln(5)

	// Disclaimer
	pdf.SetFont("Helvetica", "I", 7)
	pdf.SetTextColor(146, 64, 14)
	pdf.CellFormat(contentW, 3.5, transliterate("Questi risultati sono orientativi e potrebbero contenere errori."), "", 1, "C", false, 0, "")
	pdf.CellFormat(contentW, 3.5, transliterate("Verifica sempre sui siti ufficiali prima di presentare domanda."), "", 1, "C", false, 0, "")
	pdf.CellFormat(contentW, 3.5, transliterate("BonusPerMe non e un CAF ne un patronato."), "", 1, "C", false, 0, "")

	// ═════════════════════════════════════════════════════════════
	// OUTPUT
	// ═════════════════════════════════════════════════════════════
	disposition := "attachment"
	if r.URL.Query().Get("mode") == "inline" {
		disposition = "inline"
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`%s; filename="bonusperme-report-%s.pdf"`, disposition, dateStr))

	if err := pdf.Output(w); err != nil {
		sentryutil.CaptureError(err, map[string]string{"handler": "report", "phase": "pdf-output"})
		http.Error(w, "Errore generazione PDF", http.StatusInternalServerError)
	}
}

// profileCell draws a label+value pair in the profile grid.
func profileCell(pdf *gofpdf.Fpdf, x, y, w float64, label, value string) {
	pdf.SetXY(x, y)
	pdf.SetFont("Helvetica", "", 7)
	setText(pdf, cInk50)
	pdf.CellFormat(w-6, 3.5, label, "", 1, "L", false, 0, "")
	pdf.SetXY(x, y+4.5)
	pdf.SetFont("Helvetica", "B", 9.5)
	setText(pdf, cInk90)
	pdf.CellFormat(w-6, 4.5, value, "", 0, "L", false, 0, "")
}

// ─── drawBonusDetail: clean left-accent-bar design ───
func drawBonusDetail(pdf *gofpdf.Fpdf, b models.Bonus, profile models.UserProfile) {
	// Estimate space needed
	needed := 60.0
	if len(b.Requisiti) > 0 {
		needed += float64(len(b.Requisiti))*5 + 8
	}
	if len(b.ComeRichiederlo) > 0 {
		needed += float64(len(b.ComeRichiederlo))*5 + 8
	}
	if len(b.Documenti) > 0 {
		needed += float64(len(b.Documenti))*5 + 8
	}
	// Cap at reasonable max (will page break if needed)
	if needed > 200 {
		needed = 80
	}

	startY := ensureSpace(pdf, needed)
	if startY < marginT+6 {
		startY = marginT + 6
	}
	pdf.SetY(startY)

	accentColor, _ := compatColor(b.Compatibilita)
	innerX := marginL + 6 // After accent bar
	innerW := contentW - 6

	// ── A) NAME + PERCENTAGE ──
	y := startY

	pdf.SetXY(innerX, y)
	pdf.SetFont("Helvetica", "B", 13)
	setText(pdf, cInk90)
	nome := transliterate(b.Nome)
	pdf.CellFormat(innerW-30, 7, nome, "", 0, "L", false, 0, "")

	// Percentage pill
	pillText := fmt.Sprintf("%d%%", b.Compatibilita)
	pillFg, pillBg := compatColor(b.Compatibilita)
	pillW := pdf.GetStringWidth(pillText) + 8
	drawPill(pdf, marginL+contentW-pillW, y+0.5, pillText, pillBg, pillFg)

	y += 9

	// ── B) ENTE + SCADENZA ──
	pdf.SetXY(innerX, y)
	pdf.SetFont("Helvetica", "", 8)
	setText(pdf, cInk50)
	meta := transliterate(b.Ente)
	if b.Scadenza != "" {
		meta += "  |  "
	}
	pdf.CellFormat(0, 4, meta, "", 0, "L", false, 0, "")
	if b.Scadenza != "" {
		xAfter := innerX + pdf.GetStringWidth(meta)
		pdf.SetXY(xAfter, y)
		setText(pdf, cTerra)
		pdf.SetFont("Helvetica", "B", 8)
		pdf.CellFormat(0, 4, transliterate(b.Scadenza), "", 0, "L", false, 0, "")
	}
	y += 7

	// ── C) IMPORTO BAR ──
	boxH := 14.0
	if b.ImportoReale != "" && b.ImportoReale != b.Importo {
		boxH = 18.0
	}
	setFill(pdf, cCream)
	pdf.RoundedRect(innerX, y, innerW, boxH, 2, "1234", "F")

	pdf.SetXY(innerX+5, y+3)
	pdf.SetFont("Courier", "B", 11)
	setText(pdf, cGreen)
	importoText := transliterate(b.Importo)
	if importoText == "" {
		importoText = "Vedi sito ufficiale"
		pdf.SetFont("Helvetica", "", 9)
		setText(pdf, cInk50)
	}
	pdf.CellFormat(innerW-10, 5, importoText, "", 1, "L", false, 0, "")

	if b.ImportoReale != "" && b.ImportoReale != b.Importo {
		pdf.SetXY(innerX+5, y+10)
		pdf.SetFont("Helvetica", "", 7.5)
		setText(pdf, cInk50)
		pdf.CellFormat(innerW-10, 4, transliterate("Stimato per te: "+b.ImportoReale), "", 0, "L", false, 0, "")

		// "STIMATO PER TE" label
		pdf.SetFont("Helvetica", "B", 6)
		setText(pdf, cGreen)
		pdf.SetXY(innerX+innerW-35, y+2)
		pdf.CellFormat(30, 3, "STIMATO PER TE", "", 0, "R", false, 0, "")
	}
	y += boxH + 4

	// ── D) DESCRIZIONE ──
	pdf.SetXY(innerX, y)
	pdf.SetFont("Helvetica", "", 8.5)
	setText(pdf, cInk75)
	desc := b.Descrizione
	if len(desc) > 280 {
		desc = desc[:277] + "..."
	}
	pdf.MultiCell(innerW, 4.2, transliterate(desc), "", "L", false)
	y = pdf.GetY() + 3

	// ── E) THIN SEPARATOR ──
	setDraw(pdf, cInk08)
	pdf.SetLineWidth(0.3)
	pdf.Line(innerX, y, innerX+innerW, y)
	y += 4

	// ── F) TWO-COLUMN LAYOUT: Requisiti | Documenti ──
	hasReq := len(b.Requisiti) > 0
	hasDoc := len(b.Documenti) > 0

	if hasReq && hasDoc {
		// Two columns
		colLeft := innerW/2 - 2
		colRight := innerW/2 - 2
		colRightX := innerX + innerW/2 + 2

		reqDocY := ensureSpace(pdf, float64(max(len(b.Requisiti), len(b.Documenti)))*5+10)
		if reqDocY > y {
			y = reqDocY
		}
		pdf.SetY(y)

		// Left: Requisiti
		pdf.SetXY(innerX, y)
		pdf.SetFont("Helvetica", "B", 7)
		setText(pdf, cInk30)
		pdf.CellFormat(colLeft, 4, "REQUISITI", "", 0, "L", false, 0, "")

		// Right: Documenti
		pdf.SetXY(colRightX, y)
		pdf.CellFormat(colRight, 4, "DOCUMENTI", "", 1, "L", false, 0, "")
		y += 6

		maxRows := len(b.Requisiti)
		if len(b.Documenti) > maxRows {
			maxRows = len(b.Documenti)
		}

		for i := 0; i < maxRows; i++ {
			rowY := y + float64(i)*5.5
			if i < len(b.Requisiti) {
				drawCheckmark(pdf, innerX, rowY)
				pdf.SetXY(innerX+5, rowY)
				pdf.SetFont("Helvetica", "", 7.5)
				setText(pdf, cInk75)
				req := b.Requisiti[i]
				if len(req) > 50 {
					req = req[:47] + "..."
				}
				pdf.CellFormat(colLeft-5, 4, transliterate(req), "", 0, "L", false, 0, "")
			}
			if i < len(b.Documenti) {
				drawSquare(pdf, colRightX, rowY)
				pdf.SetXY(colRightX+5, rowY)
				pdf.SetFont("Helvetica", "", 7.5)
				setText(pdf, cInk75)
				doc := b.Documenti[i]
				if len(doc) > 50 {
					doc = doc[:47] + "..."
				}
				pdf.CellFormat(colRight-5, 4, transliterate(doc), "", 0, "L", false, 0, "")
			}
		}
		y += float64(maxRows)*5.5 + 3
	} else if hasReq {
		// Single column: Requisiti
		y = ensureSpace(pdf, float64(len(b.Requisiti))*5+10)
		pdf.SetXY(innerX, y)
		pdf.SetFont("Helvetica", "B", 7)
		setText(pdf, cInk30)
		pdf.CellFormat(innerW, 4, "REQUISITI", "", 1, "L", false, 0, "")
		y += 6
		for i, req := range b.Requisiti {
			drawCheckmark(pdf, innerX, y+float64(i)*5.5)
			pdf.SetXY(innerX+5, y+float64(i)*5.5)
			pdf.SetFont("Helvetica", "", 7.5)
			setText(pdf, cInk75)
			pdf.CellFormat(innerW-5, 4, transliterate(req), "", 1, "L", false, 0, "")
		}
		y += float64(len(b.Requisiti))*5.5 + 3
	} else if hasDoc {
		// Single column: Documenti
		y = ensureSpace(pdf, float64(len(b.Documenti))*5+10)
		pdf.SetXY(innerX, y)
		pdf.SetFont("Helvetica", "B", 7)
		setText(pdf, cInk30)
		pdf.CellFormat(innerW, 4, "DOCUMENTI", "", 1, "L", false, 0, "")
		y += 6
		for i, doc := range b.Documenti {
			drawSquare(pdf, innerX, y+float64(i)*5.5)
			pdf.SetXY(innerX+5, y+float64(i)*5.5)
			pdf.SetFont("Helvetica", "", 7.5)
			setText(pdf, cInk75)
			pdf.CellFormat(innerW-5, 4, transliterate(doc), "", 1, "L", false, 0, "")
		}
		y += float64(len(b.Documenti))*5.5 + 3
	}

	// ── G) COME FARE DOMANDA ──
	if len(b.ComeRichiederlo) > 0 {
		y = ensureSpace(pdf, float64(len(b.ComeRichiederlo))*5.5+10)
		pdf.SetXY(innerX, y)
		pdf.SetFont("Helvetica", "B", 7)
		setText(pdf, cInk30)
		pdf.CellFormat(innerW, 4, "COME FARE DOMANDA", "", 1, "L", false, 0, "")
		y += 6

		for i, step := range b.ComeRichiederlo {
			drawStepNum(pdf, innerX, y+float64(i)*6, i+1)
			pdf.SetXY(innerX+7, y+float64(i)*6)
			pdf.SetFont("Helvetica", "", 7.5)
			setText(pdf, cInk75)
			pdf.CellFormat(innerW-7, 4, transliterate(step), "", 1, "L", false, 0, "")
		}
		y += float64(len(b.ComeRichiederlo))*6 + 2
	}

	// ── H) FOOTER: link + scadenza ──
	y += 1
	if b.LinkUfficiale != "" {
		pdf.SetXY(innerX, y)
		pdf.SetFont("Helvetica", "", 7)
		setText(pdf, cBlueMid)
		linkText := truncURL(b.LinkUfficiale, 60)
		pdf.WriteLinkString(3.5, transliterate(linkText), b.LinkUfficiale)
	}
	y += 5

	// ── Draw accent bar over full height ──
	drawAccentBar(pdf, marginL, startY, y, accentColor)

	// Bottom spacing
	pdf.SetY(y + 6)
}

// ─── drawBonusExpired: minimal compact expired card ───
func drawBonusExpired(pdf *gofpdf.Fpdf, b models.Bonus) {
	startY := ensureSpace(pdf, 16)
	if startY < marginT+6 {
		startY = marginT + 6
	}
	pdf.SetY(startY)

	innerX := marginL + 6
	innerW := contentW - 6
	y := startY

	// Name + SCADUTO pill
	pdf.SetXY(innerX, y)
	pdf.SetFont("Helvetica", "B", 10)
	setText(pdf, cInk30)
	pdf.CellFormat(innerW-35, 5.5, transliterate(b.Nome), "", 0, "L", false, 0, "")

	drawPill(pdf, marginL+contentW-pdf.GetStringWidth("SCADUTO")-8, y, "SCADUTO", cRedBg, cRed)
	y += 7

	// Importo struck through + note
	pdf.SetXY(innerX, y)
	pdf.SetFont("Courier", "", 8.5)
	setText(pdf, cInk30)
	importoText := transliterate(b.Importo)
	if importoText != "" {
		pdf.CellFormat(0, 4, importoText, "", 0, "L", false, 0, "")
		strW := pdf.GetStringWidth(importoText)
		setDraw(pdf, cInk30)
		pdf.SetLineWidth(0.25)
		pdf.Line(innerX, y+2, innerX+strW, y+2)
	}
	y += 5

	// Note
	pdf.SetXY(innerX, y)
	pdf.SetFont("Helvetica", "I", 7.5)
	setText(pdf, cInk50)
	nota := "Bonus non piu disponibile."
	if b.Scadenza != "" {
		nota += " Scaduto il " + b.Scadenza + "."
	}
	pdf.CellFormat(innerW, 4, transliterate(nota), "", 1, "L", false, 0, "")
	y += 6

	// Grey accent bar
	drawAccentBar(pdf, marginL, startY, y, cInk15)

	pdf.SetY(y + 4)
}



// ---------- 4. NotifySignupHandler ----------

var emailRe = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func NotifySignupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	body.Email = strings.TrimSpace(body.Email)

	if !emailRe.MatchString(body.Email) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Email non valida"})
		return
	}

	// Accept the request but do not persist email to disk (privacy)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}