package handlers

import (
	"bonusperme/internal/config"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// ContattiHandler serves the GET /contatti page.
func ContattiHandler(w http.ResponseWriter, r *http.Request) {
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
` + SharedMetaTags("Contatti — BonusPerMe", "Contattaci per informazioni, segnalazioni o partnership. BonusPerMe — servizio gratuito per le famiglie italiane.", "/contatti") + `
` + tsScript + `
<style>` + SharedCSS() + `
.page-hero{padding:48px 0 32px;text-align:center}
.page-hero h1{font-size:clamp(1.5rem,3.5vw,2rem);margin-bottom:10px}
.page-hero p{color:var(--ink-75);font-size:1rem}
.contact-grid{display:grid;grid-template-columns:1fr 1fr;gap:32px;margin:32px 0 48px}
.contact-form-wrap{background:#fff;border:1px solid var(--ink-15);border-radius:var(--radius-lg);padding:28px 24px;box-shadow:var(--shadow-card)}
.contact-info{display:flex;flex-direction:column;gap:16px}
.info-card{background:#fff;border:1px solid var(--ink-15);border-radius:var(--radius-lg);padding:20px;box-shadow:var(--shadow-card)}
.info-card h3{font-size:1rem;margin-bottom:8px}
.info-card p{font-size:.88rem;color:var(--ink-75)}
.info-card a{word-break:break-all}
.field{margin-bottom:14px}
.field label{display:block;font-size:.82rem;font-weight:600;margin-bottom:4px;color:var(--ink-75)}
.field input,.field select,.field textarea{width:100%;padding:10px 14px;border:1px solid var(--ink-15);border-radius:var(--radius);font-family:inherit;font-size:.92rem;background:#fff}
.field textarea{min-height:120px;resize:vertical}
.field input:focus,.field select:focus,.field textarea:focus{outline:none;border-color:var(--blue-mid)}
.privacy-check{display:flex;align-items:flex-start;gap:8px;margin:16px 0;font-size:.85rem;color:var(--ink-75)}
.privacy-check input{margin-top:3px}
.btn-contact{display:block;width:100%;padding:12px 20px;background:var(--terra);color:#fff;border:none;border-radius:var(--radius);font-family:inherit;font-size:.95rem;font-weight:600;cursor:pointer}
.btn-contact:hover{background:var(--terra-dark)}
#contactResult{display:none;margin-top:16px;padding:12px 16px;border-radius:var(--radius);font-size:.9rem;font-weight:600;text-align:center}
@media(max-width:640px){.contact-grid{grid-template-columns:1fr}}
</style>
</head>
<body>
` + SharedTopbar() + `
` + SharedHeader("/contatti") + `

<div class="page-content">
<section class="page-hero">
<h1>Contattaci</h1>
<p>Hai domande, suggerimenti o vuoi segnalare un errore? Scrivici.</p>
</section>

<div class="contact-grid">
<div class="contact-form-wrap">
<div class="field"><label>Nome <span class="required-mark">*</span></label><input type="text" id="ct-nome" placeholder="Il tuo nome"></div>
<div class="field"><label>Email <span class="required-mark">*</span></label><input type="email" id="ct-email" placeholder="La tua email"></div>
<div class="field">
<label>Oggetto <span class="optional-label">(opzionale)</span></label>
<select id="ct-oggetto">
<option value="info">Informazioni generali</option>
<option value="bug">Segnalazione errore</option>
<option value="partner">Partnership / CAF</option>
<option value="altro">Altro</option>
</select>
</div>
<div class="field"><label>Messaggio <span class="required-mark">*</span></label><textarea id="ct-messaggio" placeholder="Scrivi il tuo messaggio..."></textarea></div>
<input type="checkbox" name="botcheck" style="display:none" tabindex="-1" autocomplete="off">
<label class="privacy-check"><input type="checkbox" id="ct-privacy"> Ho letto e accetto la <a href="/privacy" target="_blank">Privacy Policy</a></label>
<p class="required-note">I campi contrassegnati con <span class="required-mark">*</span> sono obbligatori.</p>
` + tsWidget + `<button class="btn-contact" id="contactSubmitBtn" onclick="submitContact()">Invia messaggio</button>
<div id="contactResult"></div>
</div>

<div class="contact-info">
<div class="info-card">
<h3>Email</h3>
<p><a href="mailto:info@bonusperme.it">info@bonusperme.it</a></p>
</div>
<div class="info-card">
<h3>Sede</h3>
<p>Via Morazzone 4<br>22100 Como (CO)</p>
</div>
<div class="info-card">
<h3>Titolare</h3>
<p>Simone Nogara<br>P.IVA 03817020138</p>
</div>
<div class="info-card">
<h3>Rispondiamo entro</h3>
<p>24-48 ore lavorative</p>
</div>
</div>
</div>
</div>

` + SharedFooter() + `
` + SharedCookieBanner() + `

<script>
var turnstileToken='';function onTurnstileSuccess(t){turnstileToken=t;}
function submitContact(){
  clearAllErrors();
  var nome=document.getElementById('ct-nome').value.trim();
  var email=document.getElementById('ct-email').value.trim();
  var oggetto=document.getElementById('ct-oggetto').value;
  var messaggio=document.getElementById('ct-messaggio').value.trim();
  var privacy=document.getElementById('ct-privacy').checked;
  var hasErr=false;

  if(!nome){markFieldError('ct-nome','Il nome è obbligatorio');hasErr=true}
  if(!email){markFieldError('ct-email','L\'email è obbligatoria');hasErr=true}
  else if(!email.includes('@')){markFieldError('ct-email','Inserisci un indirizzo email valido');hasErr=true}
  if(!messaggio){markFieldError('ct-messaggio','Il messaggio è obbligatorio');hasErr=true}
  if(hasErr){showToast('error','Campi mancanti','Compila tutti i campi obbligatori.');return}
  if(!privacy){showToast('error','Privacy Policy','Devi accettare la Privacy Policy.');return}
  if(document.querySelector('.cf-turnstile')&&!turnstileToken){showToast('error','Verifica di sicurezza','Completa la verifica anti-bot e riprova.');return}

  var btn=document.getElementById('contactSubmitBtn');
  var originalText=btn.textContent;
  btn.disabled=true;
  btn.textContent='Invio in corso...';

  var w3data={
    access_key:'` + config.Cfg.Web3FormsAccessKey + `',
    subject:'BonusPerMe — Nuovo messaggio contatti',
    from_name:'BonusPerMe Contatti',
    name:nome,
    email:email,
    oggetto:oggetto,
    message:messaggio,
    source:'bonusperme.it/contatti',
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
      showToast('success','Messaggio inviato!','Ti risponderemo entro 24-48 ore lavorative.');
      if(typeof pushDataLayer==='function'){pushDataLayer({event:'contact_form_submit',oggetto:oggetto});pushDataLayer({event:'form_submit',form_name:'contatti'});}
      document.getElementById('ct-nome').value='';
      document.getElementById('ct-email').value='';
      document.getElementById('ct-messaggio').value='';
      document.getElementById('ct-privacy').checked=false;
    } else {
      showToast('error','Errore invio',result.message||'Riprova tra qualche minuto.');
    }
  })
  .catch(function(err){
    console.error('Web3Forms error:',err);
    showToast('error','Errore di connessione','Verifica la connessione internet e riprova.');
  })
  .finally(function(){
    btn.disabled=false;
    btn.textContent=originalText;
  });

  // Backend logging (non-blocking)
  var logH={'Content-Type':'application/json'};if(turnstileToken)logH['X-Turnstile-Token']=turnstileToken;
  fetch('/api/contact',{method:'POST',headers:logH,body:JSON.stringify({nome:nome,email:email,oggetto:oggetto,messaggio:messaggio})}).catch(function(){});
}
['ct-nome','ct-email','ct-messaggio'].forEach(function(id){var el=document.getElementById(id);if(el)el.addEventListener('input',function(){clearFieldError(id)})});
</script>
` + SharedScripts() + `
</body>
</html>`)

	w.Write([]byte(sb.String()))
}

type contactRequest struct {
	Nome     string `json:"nome"`
	Email    string `json:"email"`
	Oggetto  string `json:"oggetto"`
	Messaggio string `json:"messaggio"`
}

// ContactHandler handles POST /api/contact
func ContactHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !verifyTurnstile(getTurnstileToken(r)) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": "Verifica di sicurezza non superata"})
		return
	}

	var req contactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": "Dati non validi"})
		return
	}
	defer r.Body.Close()

	req.Nome = strings.TrimSpace(req.Nome)
	req.Email = strings.TrimSpace(req.Email)
	req.Messaggio = strings.TrimSpace(req.Messaggio)

	if req.Nome == "" || req.Email == "" || req.Messaggio == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": "Compila tutti i campi obbligatori"})
		return
	}
	if !strings.Contains(req.Email, "@") {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": "Email non valida"})
		return
	}

	log.Printf("[contact] nome=%s email=%s oggetto=%s", req.Nome, req.Email, req.Oggetto)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
}
