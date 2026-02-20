package handlers

import (
	"bonusperme/internal/config"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// PerCAFHandler serves the /per-caf landing page.
func PerCAFHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tsScript := ""
	tsWidget := ""
	if config.Cfg.TurnstileSiteKey != "" {
		tsScript = `<script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>`
		tsWidget = `<div style="margin:16px 0"><div class="cf-turnstile" data-sitekey="` + config.Cfg.TurnstileSiteKey + `" data-callback="onTurnstileSuccess" data-theme="light"></div></div>`
	}

	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html lang="it">
<head>
` + SharedMetaTags("BonusPerMe per i CAF — Centro di Assistenza Fiscale", "BonusPerMe per i Centri di Assistenza Fiscale. I tuoi clienti arrivano con il report già pronto.", "/per-caf") + `
` + tsScript + `
<style>` + SharedCSS() + `
.hero-caf{padding:56px 0 40px;text-align:center}
.badge{display:inline-block;padding:5px 14px;background:var(--green-light);color:var(--green);font-size:.72rem;font-weight:700;border-radius:var(--radius);margin-bottom:14px;text-transform:uppercase;letter-spacing:.8px}
.hero-caf h1{font-size:clamp(1.6rem,4vw,2.3rem);margin-bottom:14px}
.hero-caf p{color:var(--ink-75);max-width:560px;margin:0 auto;font-size:1rem}
.value-cards{display:grid;grid-template-columns:repeat(3,1fr);gap:16px;margin:40px 0}
.value-card{background:#fff;border:1px solid var(--ink-15);border-radius:var(--radius-lg);padding:24px 20px;text-align:center;box-shadow:var(--shadow-card)}
.value-card .num{font-family:'DM Serif Display',serif;font-size:1.8rem;color:var(--terra);margin-bottom:4px}
.value-card p{font-size:.85rem;color:var(--ink-75)}
.steps{margin:48px 0}
.steps h2{text-align:center;margin-bottom:24px;font-size:1.5rem}
.step-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:20px}
.step{background:#fff;border:1px solid var(--ink-15);border-radius:var(--radius-lg);padding:24px 20px;text-align:center;box-shadow:var(--shadow-card)}
.step .step-num{width:36px;height:36px;background:var(--blue);color:#fff;border-radius:50%;display:inline-flex;align-items:center;justify-content:center;font-weight:700;margin-bottom:10px;font-size:.9rem}
.step h3{font-size:1rem;margin-bottom:6px}
.step p{font-size:.85rem;color:var(--ink-75)}
.advantages{margin:48px 0}
.advantages h2{text-align:center;margin-bottom:24px;font-size:1.5rem}
.adv-grid{display:grid;grid-template-columns:repeat(2,1fr);gap:16px}
.adv{background:#fff;padding:20px;border:1px solid var(--ink-15);border-radius:var(--radius-lg);box-shadow:var(--shadow-card)}
.adv h4{font-size:.92rem;margin-bottom:4px;font-family:'DM Sans',sans-serif;font-weight:600}
.adv p{font-size:.82rem;color:var(--ink-75)}
.caf-form-section{background:var(--warm-cream);border:1px solid var(--ink-15);border-radius:var(--radius-lg);padding:36px 32px;margin:48px 0;text-align:center}
.caf-form-section h2{font-size:1.3rem;margin-bottom:8px}
.caf-form-section>p{color:var(--ink-75);margin-bottom:24px;font-size:.95rem}
.caf-form{max-width:400px;margin:0 auto;text-align:left}
.caf-form .field{margin-bottom:14px}
.caf-form label{display:block;font-size:.82rem;font-weight:600;margin-bottom:4px;color:var(--ink-75)}
.caf-form input,.caf-form select{width:100%;padding:10px 14px;border:1px solid var(--ink-15);border-radius:var(--radius);font-family:inherit;font-size:.92rem;background:#fff}
.caf-form input:focus,.caf-form select:focus{outline:none;border-color:var(--blue-mid)}
.btn-caf{display:block;width:100%;padding:12px 20px;background:var(--terra);color:#fff;border:none;border-radius:var(--radius);font-family:inherit;font-size:.95rem;font-weight:600;cursor:pointer;margin-top:20px}
.btn-caf:hover{background:var(--terra-dark)}
#cafResult{display:none;margin-top:16px;padding:12px 16px;border-radius:var(--radius);font-size:.9rem;font-weight:600;text-align:center}
.api-box{border:1px solid var(--ink-15);border-radius:var(--radius-lg);padding:24px;margin:32px 0;background:#fff;box-shadow:var(--shadow-card)}
.api-box h3{font-size:1rem;margin-bottom:8px}
.api-box code{display:block;background:var(--warm-cream);padding:12px;border-radius:var(--radius);font-size:.85rem;margin:8px 0;overflow-x:auto;font-family:'JetBrains Mono',monospace}
@media(max-width:640px){.value-cards,.step-grid{grid-template-columns:1fr}.adv-grid{grid-template-columns:1fr}.caf-form-section{padding:24px 20px}}
</style>
</head>
<body>
` + SharedTopbar() + `
` + SharedHeader("/per-caf") + `

<div class="page-content">
<section class="hero-caf">
<div class="badge">Per i CAF</div>
<h1>I tuoi clienti arrivano con il report pronto</h1>
<p>BonusPerMe aiuta le famiglie a scoprire i bonus a cui hanno diritto. Il risultato? Clienti che arrivano al CAF sapendo esattamente cosa chiedere.</p>
</section>

<section class="value-cards">
<div class="value-card"><div class="num">-40%</div><p>Tempo consulenza base</p></div>
<div class="value-card"><div class="num">+30%</div><p>Pratiche evase al giorno</p></div>
<div class="value-card"><div class="num">€0</div><p>Costo per il tuo CAF</p></div>
</section>

<section class="steps">
<h2>Come funziona</h2>
<div class="step-grid">
<div class="step"><div class="step-num">1</div><h3>Il cliente usa BonusPerMe</h3><p>Risponde a 4 domande e scopre i bonus compatibili in 2 minuti.</p></div>
<div class="step"><div class="step-num">2</div><h3>Scarica il report PDF</h3><p>Report professionale con bonus, requisiti, documenti e link ufficiali.</p></div>
<div class="step"><div class="step-num">3</div><h3>Viene al CAF preparato</h3><p>Documenti pronti, domande chiare. Meno tempo, più pratiche.</p></div>
</div>
</section>

<section class="advantages">
<h2>Vantaggi per il tuo CAF</h2>
<div class="adv-grid">
<div class="adv"><h4>Clienti informati</h4><p>Arrivano sapendo cosa chiedere, con report e documenti pronti.</p></div>
<div class="adv"><h4>Meno consulenza base</h4><p>Le domande generiche vengono filtrate prima dell'appuntamento.</p></div>
<div class="adv"><h4>Più pratiche al giorno</h4><p>Clienti preparati = consulenze più rapide = più appuntamenti.</p></div>
<div class="adv"><h4>Zero costi</h4><p>BonusPerMe è gratuito, open source e senza pubblicità.</p></div>
</div>
</section>

<section class="caf-form-section">
<h2>Registra il tuo CAF</h2>
<p>Ricevi aggiornamenti e accesso anticipato al widget embeddabile.</p>
<div class="caf-form">
<div class="field"><label>Nome CAF <span class="required-mark">*</span></label><input type="text" id="caf-nome" placeholder="Es. CAF CISL Milano"></div>
<div class="field"><label>Email referente <span class="required-mark">*</span></label><input type="email" id="caf-email" placeholder="referente@caf.it"></div>
<div class="field"><label>Telefono <span class="optional-label">(opzionale)</span></label><input type="tel" id="caf-telefono" placeholder="Es. 02 1234567"></div>
<div class="field"><label>Provincia <span class="required-mark">*</span></label><input type="text" id="caf-provincia" placeholder="Es. Milano"></div>
<input type="checkbox" name="botcheck" style="display:none" tabindex="-1" autocomplete="off">
<p class="required-note">I campi contrassegnati con <span class="required-mark">*</span> sono obbligatori.</p>
` + tsWidget + `<button class="btn-caf" id="cafSubmitBtn" onclick="submitCAFSignup()">Registra il CAF</button>
<div id="cafResult"></div>
</div>
</section>

<section class="api-box">
<h3>Open Data API (disponibile ora)</h3>
<p>Accedi ai dati di tutti i bonus italiani tramite API REST:</p>
<code>GET /api/bonus — Lista completa bonus (nazionali + regionali)</code>
<code>GET /api/bonus/{id} — Dettaglio singolo bonus</code>
<p style="font-size:.85rem;color:var(--ink-50);margin-top:8px">Formato JSON, CORS abilitato, cache 1 ora. Rate limit: 60 richieste/minuto.</p>
</section>
</div>

` + SharedFooter() + `
` + SharedCookieBanner() + `

<script>
var turnstileToken='';function onTurnstileSuccess(t){turnstileToken=t;}
function submitCAFSignup(){
  clearAllErrors();
  var nome=document.getElementById('caf-nome').value.trim();
  var email=document.getElementById('caf-email').value.trim();
  var telefono=document.getElementById('caf-telefono').value.trim();
  var provincia=document.getElementById('caf-provincia').value.trim();
  var hasErr=false;

  if(!nome){markFieldError('caf-nome','Il nome CAF è obbligatorio');hasErr=true}
  if(!email){markFieldError('caf-email','L\'email è obbligatoria');hasErr=true}
  else if(!email.includes('@')){markFieldError('caf-email','Inserisci un indirizzo email valido');hasErr=true}
  if(!provincia){markFieldError('caf-provincia','La provincia è obbligatoria');hasErr=true}
  if(hasErr){showToast('error','Campi mancanti','Compila tutti i campi obbligatori.');return}
  if(document.querySelector('.cf-turnstile')&&!turnstileToken){showToast('error','Verifica di sicurezza','Completa la verifica anti-bot e riprova.');return}

  var btn=document.getElementById('cafSubmitBtn');
  var originalText=btn.textContent;
  btn.disabled=true;
  btn.textContent='Registrazione in corso...';

  var w3data={
    access_key:'` + config.Cfg.Web3FormsAccessKey + `',
    subject:'BonusPerMe — Nuova registrazione CAF',
    from_name:'BonusPerMe CAF Signup',
    nome_caf:nome,
    email:email,
    telefono:telefono,
    provincia:provincia,
    source:'bonusperme.it/per-caf',
    timestamp:new Date().toISOString()
  };

  fetch('https://api.web3forms.com/submit',{
    method:'POST',
    headers:{'Content-Type':'application/json','Accept':'application/json'},
    body:JSON.stringify(w3data)
  })
  .then(function(r){return r.json();})
  .then(function(result){
    if(result.success){
      showToast('success','CAF registrato!','Riceverai aggiornamenti e accesso anticipato al widget.');
      if(typeof pushDataLayer==='function')pushDataLayer({event:'caf_signup',provincia:provincia});
      document.querySelector('.caf-form').querySelectorAll('input[type="text"],input[type="email"],input[type="tel"]').forEach(function(i){i.value='';});
    } else {
      showToast('error','Errore registrazione',result.message||'Riprova.');
    }
  })
  .catch(function(){
    showToast('error','Errore di connessione','Verifica la connessione e riprova.');
  })
  .finally(function(){
    btn.disabled=false;
    btn.textContent=originalText;
  });

  // Backend logging (non-blocking)
  var logH={'Content-Type':'application/json'};if(turnstileToken)logH['X-Turnstile-Token']=turnstileToken;
  fetch('/api/caf-signup',{method:'POST',headers:logH,body:JSON.stringify({nome:nome,email:email,telefono:telefono,provincia:provincia})}).catch(function(){});
}
['caf-nome','caf-email','caf-provincia'].forEach(function(id){var el=document.getElementById(id);if(el)el.addEventListener('input',function(){clearFieldError(id)})});
</script>
` + SharedScripts() + `
</body>
</html>`)

	w.Write([]byte(sb.String()))
}

type cafSignupRequest struct {
	Nome      string `json:"nome"`
	Email     string `json:"email"`
	Telefono  string `json:"telefono"`
	Provincia string `json:"provincia"`
}

// CAFSignupHandler handles POST /api/caf-signup
func CAFSignupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !verifyTurnstile(getTurnstileToken(r)) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": "Verifica di sicurezza non superata"})
		return
	}

	var req cafSignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": "Dati non validi"})
		return
	}
	defer r.Body.Close()

	req.Nome = strings.TrimSpace(req.Nome)
	req.Email = strings.TrimSpace(req.Email)
	req.Provincia = strings.TrimSpace(req.Provincia)

	if req.Nome == "" || req.Email == "" || req.Provincia == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": "Compila tutti i campi obbligatori"})
		return
	}
	if !strings.Contains(req.Email, "@") {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": "Email non valida"})
		return
	}

	log.Printf("[caf-signup] nome=%s email=%s provincia=%s", req.Nome, req.Email, req.Provincia)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
}
