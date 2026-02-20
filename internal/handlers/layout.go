package handlers

import (
	"bonusperme/internal/config"
	"strings"
)

// SharedMetaTags returns common SEO meta tags for a page.
func SharedMetaTags(title, description, canonicalPath string) string {
	base := config.Cfg.BaseURL
	return `<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>` + title + `</title>
<meta name="description" content="` + description + `">
<meta name="robots" content="index, follow">
<meta property="og:type" content="website">
<meta property="og:url" content="` + base + canonicalPath + `">
<meta property="og:title" content="` + title + `">
<meta property="og:description" content="` + description + `">
<meta property="og:image" content="` + base + `/og-image.png">
<meta property="og:image:width" content="1200">
<meta property="og:image:height" content="630">
<meta property="og:site_name" content="BonusPerMe">
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="` + title + `">
<meta name="twitter:description" content="` + description + `">
<meta name="twitter:image" content="` + base + `/og-image.png">
<link rel="canonical" href="` + base + canonicalPath + `">
<link rel="icon" type="image/png" sizes="32x32" href="/favicon-32x32.png">
<link rel="icon" type="image/png" sizes="16x16" href="/favicon-16x16.png">
<link rel="apple-touch-icon" sizes="180x180" href="/apple-touch-icon.png">
<meta name="theme-color" content="#1B3A54">
<link rel="stylesheet" href="/fonts/fonts.css">`
}

// SharedCSS returns CSS for shared layout components (topbar, header, nav, footer, cookie banner).
func SharedCSS() string {
	return `
:root{--ink:#1C1C1F;--ink-75:#404045;--ink-50:#76767C;--ink-30:#757575;--ink-15:#D4D4D7;--ink-05:#F0F0F1;--warm-white:#FAFAF7;--warm-cream:#F4F3EE;--warm-sand:#EAE8E0;--blue:#1B3A54;--blue-mid:#2D5F8A;--blue-light:#E6EEF4;--terra:#C0522E;--terra-dark:#9E3F20;--terra-light:#FAF0EB;--green:#2A6B45;--green-light:#E8F3EC;--radius:5px;--radius-lg:8px;--shadow-card:0 1px 3px rgba(0,0,0,0.05),0 4px 16px rgba(0,0,0,0.04);--max-w:1040px;--gutter:24px}
.page-content{max-width:720px;margin:0 auto;padding:0 var(--gutter)}
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:'DM Sans',-apple-system,sans-serif;background:var(--warm-white);color:var(--ink);min-height:100vh;font-size:15px;line-height:1.65;-webkit-font-smoothing:antialiased}
h1,h2,h3{font-family:'DM Serif Display',Georgia,serif;font-weight:400}
a{color:var(--blue-mid);text-decoration:none}a:hover{text-decoration:underline}
.container{max-width:var(--max-w);margin:0 auto;padding:0 var(--gutter)}
.topbar{background:var(--blue);color:rgba(255,255,255,0.7);font-size:.72rem;padding:6px 0}
.topbar-inner{max-width:var(--max-w);margin:0 auto;padding:0 var(--gutter);display:flex;align-items:center;gap:6px;flex-wrap:wrap}
.topbar-separator{opacity:.4}
.topbar-right{margin-left:auto}
.lang-select{background:rgba(255,255,255,0.1);color:#fff;border:1px solid rgba(255,255,255,0.2);border-radius:4px;font-size:.72rem;padding:2px 6px;cursor:pointer;font-family:inherit}
.lang-select option{color:#333;background:#fff}
.site-header{background:#fff;border-bottom:1px solid var(--ink-15);position:sticky;top:0;z-index:100}
.site-header .header-inner{max-width:var(--max-w);margin:0 auto;padding:0 var(--gutter);display:flex;align-items:center;justify-content:space-between;height:54px}
.site-header .logo{text-decoration:none;display:flex;align-items:center;gap:8px;color:var(--ink)}
.site-header .logo:hover{text-decoration:none}
.site-header .logo-mark{width:26px;height:26px;background:var(--blue);border-radius:4px;display:flex;align-items:center;justify-content:center;color:#fff;font-family:'DM Serif Display',serif;font-size:14px;line-height:1}
.site-header .logo-text{font-family:'DM Serif Display',serif;font-size:1.15rem;font-weight:400;color:var(--blue)}
.main-nav{display:flex;gap:20px}
.nav-link{font-family:'DM Sans',sans-serif;font-size:.85rem;color:var(--ink-50);text-decoration:none;transition:color .15s}
.nav-link:hover{color:var(--blue);text-decoration:none}
.nav-link.active{color:var(--blue);font-weight:600}
.site-footer{background:var(--warm-cream);border-top:1px solid var(--warm-sand);padding:32px 0 24px;margin-top:48px}
.site-footer .footer-inner{max-width:var(--max-w);margin:0 auto;padding:0 var(--gutter);text-align:center}
.footer-nav{display:flex;justify-content:center;gap:16px;flex-wrap:wrap;margin-bottom:20px}
.footer-nav a{font-size:.8rem;color:var(--ink-50);text-decoration:none}
.footer-nav a:hover{color:var(--blue);text-decoration:underline}
.footer-legal{margin-bottom:16px;font-size:.75rem;color:var(--ink-30);line-height:1.6}
.footer-brand{font-family:'DM Serif Display',serif;font-size:1rem;color:var(--blue);margin-bottom:4px}
.footer-legal p{margin:0}
.footer-email a{color:var(--blue-mid);text-decoration:none}
.footer-email a:hover{text-decoration:underline}
.footer-disclaimer{font-size:.7rem;color:var(--ink-30);line-height:1.5;font-style:italic}
.footer-disclaimer p{margin:2px 0}
.cookie-banner{position:fixed;bottom:0;left:0;right:0;background:#fff;border-top:1px solid var(--ink-15);padding:16px 24px;z-index:1000;box-shadow:0 -2px 10px rgba(0,0,0,0.1);display:none}
.cookie-inner{max-width:var(--max-w);margin:0 auto;display:flex;align-items:center;gap:16px;flex-wrap:wrap}
.cookie-text{flex:1;font-size:.82rem;color:var(--ink-75)}.cookie-text a{color:var(--blue-mid)}
.cookie-btns{display:flex;gap:8px}
.cookie-btn{padding:8px 18px;border:none;border-radius:var(--radius);font-family:inherit;font-size:.82rem;font-weight:600;cursor:pointer}
.cookie-btn-accept{background:var(--blue);color:#fff}
.cookie-btn-reject{background:var(--ink-05);color:var(--ink-75)}
@media(max-width:640px){.topbar-inner{justify-content:center;text-align:center}.topbar-right{margin-left:0;margin-top:4px;width:100%;text-align:center}.main-nav{gap:12px}.nav-link{font-size:.78rem}.cookie-inner{flex-direction:column;text-align:center}.cookie-btns{width:100%;justify-content:center}.site-footer .footer-inner{text-align:center}}
.toast-container{position:fixed;top:20px;right:20px;z-index:10000;display:flex;flex-direction:column;gap:10px;pointer-events:none}
.toast{pointer-events:auto;display:flex;align-items:flex-start;gap:10px;background:#fff;border-radius:var(--radius-lg);padding:14px 16px;box-shadow:0 4px 20px rgba(0,0,0,0.12);border-left:4px solid var(--ink-30);max-width:380px;animation:toastIn .3s ease;position:relative}
.toast--error{border-left-color:#dc2626}.toast--success{border-left-color:var(--green)}.toast--warning{border-left-color:#d97706}.toast--info{border-left-color:var(--blue-mid)}
.toast-icon{font-size:1.1rem;flex-shrink:0;margin-top:1px}
.toast-content{flex:1;min-width:0}
.toast-title{font-weight:600;font-size:.88rem;margin-bottom:2px}
.toast-msg{font-size:.82rem;color:var(--ink-75)}
.toast-close{background:none;border:none;font-size:1.1rem;color:var(--ink-30);cursor:pointer;padding:0;line-height:1;flex-shrink:0}
.toast-close:hover{color:var(--ink)}
.toast.hiding{animation:toastOut .3s ease forwards}
@keyframes toastIn{from{opacity:0;transform:translateX(40px)}to{opacity:1;transform:translateX(0)}}
@keyframes toastOut{from{opacity:1;transform:translateX(0)}to{opacity:0;transform:translateX(40px)}}
@media(max-width:640px){.toast-container{top:auto;bottom:20px;right:10px;left:10px}.toast{max-width:100%}}
.required-mark{color:#dc2626;font-weight:700}
.optional-label{color:var(--ink-30);font-weight:400;font-size:.78rem;margin-left:4px}
.field-invalid{border-color:#dc2626!important;box-shadow:0 0 0 2px rgba(220,38,38,0.15)!important}
.field-error{color:#dc2626;font-size:.78rem;margin-top:3px;display:block}
.required-note{font-size:.78rem;color:var(--ink-50);margin:12px 0 4px}
`
}

// SharedGTMNoscript returns the GTM noscript iframe (placed right after <body>).
func SharedGTMNoscript() string {
	if config.Cfg.GTMID == "" {
		return ""
	}
	return `<noscript><iframe src="https://www.googletagmanager.com/ns.html?id=` + config.Cfg.GTMID + `" height="0" width="0" style="display:none;visibility:hidden"></iframe></noscript>`
}

// SharedTopbar returns the unified topbar HTML.
func SharedTopbar() string {
	return SharedGTMNoscript() + `<div class="topbar"><div class="topbar-inner">` +
		`<span id="lastUpdate" data-i18n="topbar.updated">Dati aggiornati al ...</span>` +
		`<span class="topbar-separator">&middot;</span>` +
		`<span><span id="proofCounter">0</span> <span data-i18n="hero.counter_label">famiglie aiutate</span></span>` +
		`<div class="topbar-right">` +
		`<select id="langSelect" class="lang-select" aria-label="Lingua" onchange="selectLang(this.value)">` +
		`<option value="it">Italiano</option>` +
		`<option value="en">English</option>` +
		`<option value="fr">Français</option>` +
		`<option value="es">Español</option>` +
		`<option value="ro">Română</option>` +
		`<option value="ar">العربية</option>` +
		`<option value="sq">Shqip</option>` +
		`</select></div></div></div>`
}

// SharedHeader returns the unified header with nav. activePage should be "/", "/per-caf", or "/contatti".
func SharedHeader(activePage string) string {
	homeActive, cafActive, guideActive, contattiActive := "", "", "", ""
	switch {
	case activePage == "/":
		homeActive = " active"
	case activePage == "/per-caf":
		cafActive = " active"
	case activePage == "/guide" || strings.HasPrefix(activePage, "/guide/"):
		guideActive = " active"
	case activePage == "/contatti":
		contattiActive = " active"
	}
	return `<header class="site-header"><div class="container header-inner">` +
		`<a href="/" class="logo" aria-label="BonusPerMe — Torna alla home">` +
		`<div class="logo-mark">B</div><span class="logo-text">BonusPerMe</span></a>` +
		`<nav class="main-nav" aria-label="Navigazione principale">` +
		`<a href="/" class="nav-link` + homeActive + `" data-i18n="nav.home">Home</a>` +
		`<a href="/guide" class="nav-link` + guideActive + `" data-i18n="nav.guide">Guide</a>` +
		`<a href="/per-caf" class="nav-link` + cafActive + `" data-i18n="nav.caf">Per i CAF</a>` +
		`<a href="/contatti" class="nav-link` + contattiActive + `" data-i18n="nav.contatti">Contatti</a>` +
		`</nav></div></header>`
}

// SharedFooter returns the unified footer HTML.
func SharedFooter() string {
	return `<footer class="site-footer" role="contentinfo"><div class="footer-inner">` +
		`<nav class="footer-nav" aria-label="Navigazione footer">` +
		`<a href="/" data-i18n="nav.home">Home</a>` +
		`<a href="/guide" data-i18n="nav.guide">Guide</a>` +
		`<a href="/per-caf" data-i18n="nav.caf">Per i CAF</a>` +
		`<a href="/contatti" data-i18n="nav.contatti">Contatti</a>` +
		`<a href="/privacy">Privacy Policy</a>` +
		`<a href="/cookie-policy">Cookie Policy</a>` +
		`</nav>` +
		`<div class="footer-legal">` +
		`<p class="footer-brand">BonusPerMe</p>` +
		`<p>Simone Nogara</p>` +
		`<p>P.IVA 03817020138 &middot; C.F. NGRSMN91P14C933V</p>` +
		`<p>Via Morazzone 4, 22100 Como (CO), Italia</p>` +
		`<p class="footer-email"><a href="mailto:info@bonusperme.it">info@bonusperme.it</a></p>` +
		`</div>` +
		`<div class="footer-disclaimer">` +
		`<p data-i18n="footer.disclaimer">Questo servizio è a scopo orientativo e non sostituisce la consulenza di un professionista, CAF o patronato.</p>` +
		`<p>Progetto gratuito e open source &middot; <a href="https://github.com/nogarasimone/bonusperme" style="color:var(--blue-mid)">GitHub</a></p>` +
		`</div></div></footer>`
}

// SharedCookieBanner returns the cookie consent banner HTML.
func SharedCookieBanner() string {
	return `<div id="cookieBanner" class="cookie-banner" style="display:none"><div class="cookie-inner">` +
		`<div class="cookie-text">Utilizziamo cookie tecnici e, con il tuo consenso, cookie analitici per migliorare il servizio. ` +
		`<a href="/cookie-policy">Cookie Policy</a> &middot; <a href="/privacy">Privacy Policy</a></div>` +
		`<div class="cookie-btns">` +
		`<button class="cookie-btn cookie-btn-accept" onclick="acceptCookies()">Accetta</button>` +
		`<button class="cookie-btn cookie-btn-reject" onclick="rejectCookies()">Rifiuta</button>` +
		`</div></div></div>`
}

// SharedScripts returns common JS for language switcher, counter, last update, and cookie banner.
// This is the baseline version for subpages. index.html has its own more sophisticated version.
func SharedScripts() string {
	return `<div id="toastContainer" class="toast-container" aria-live="polite"></div>
<script>
(function(){if(!document.getElementById('toastContainer')){var c=document.createElement('div');c.id='toastContainer';c.className='toast-container';c.setAttribute('aria-live','polite');document.body.appendChild(c)}})();
function escHtml(s){var d=document.createElement('div');d.textContent=s;return d.innerHTML}
function showToast(type,title,message,duration){
  duration=duration||5000;
  var container=document.getElementById('toastContainer');if(!container)return;
  var icons={error:'&#x26A0;',success:'&#x2713;',warning:'&#x26A0;',info:'&#x2139;'};
  var toast=document.createElement('div');toast.className='toast toast--'+type;
  toast.innerHTML='<span class="toast-icon">'+( icons[type]||icons.info)+'</span><div class="toast-content"><div class="toast-title">'+escHtml(title)+'</div><div class="toast-msg">'+escHtml(message)+'</div></div><button class="toast-close" aria-label="Chiudi">&times;</button>';
  var timer,remaining=duration,start=Date.now();
  function startTimer(){start=Date.now();timer=setTimeout(function(){dismiss()},remaining)}
  function dismiss(){toast.classList.add('hiding');setTimeout(function(){if(toast.parentNode)toast.parentNode.removeChild(toast)},300)}
  toast.querySelector('.toast-close').onclick=dismiss;
  toast.onmouseenter=function(){clearTimeout(timer);remaining-=Date.now()-start};
  toast.onmouseleave=function(){startTimer()};
  var toasts=container.querySelectorAll('.toast');if(toasts.length>=3){toasts[0].classList.add('hiding');setTimeout(function(){if(toasts[0].parentNode)toasts[0].parentNode.removeChild(toasts[0])},300)}
  container.appendChild(toast);startTimer();
}
function markFieldError(fieldId,msg){
  var el=document.getElementById(fieldId);if(!el)return;
  el.classList.add('field-invalid');
  var existing=el.parentNode.querySelector('.field-error');if(existing)existing.remove();
  var span=document.createElement('span');span.className='field-error';span.textContent=msg;
  el.parentNode.appendChild(span);
}
function clearFieldError(fieldId){
  var el=document.getElementById(fieldId);if(!el)return;
  el.classList.remove('field-invalid');
  var existing=el.parentNode.querySelector('.field-error');if(existing)existing.remove();
}
function clearAllErrors(){document.querySelectorAll('.field-invalid').forEach(function(el){el.classList.remove('field-invalid')});document.querySelectorAll('.field-error').forEach(function(el){el.remove()})}
window.addEventListener('offline',function(){showToast('warning','Offline','Connessione internet persa. Alcune funzionalità potrebbero non essere disponibili.')});
window.addEventListener('online',function(){showToast('success','Online','Connessione ripristinata.')});
window.dataLayer=window.dataLayer||[];
function pushDataLayer(obj){window.dataLayer.push(obj);}
var __GTM_ID__='` + config.Cfg.GTMID + `';
function loadGTM(){if(!__GTM_ID__||window._gtmLoaded)return;window._gtmLoaded=true;(function(w,d,s,l,i){w[l]=w[l]||[];w[l].push({'gtm.start':new Date().getTime(),event:'gtm.js'});var f=d.getElementsByTagName(s)[0],j=d.createElement(s),dl=l!='dataLayer'?'&l='+l:'';j.async=true;j.src='https://www.googletagmanager.com/gtm.js?id='+i+dl;f.parentNode.insertBefore(j,f)})(window,document,'script','dataLayer',__GTM_ID__);pushDataLayer({event:'cookie_consent_granted'});}
if(localStorage.getItem('cookie_consent')==='accepted'){loadGTM();}
var currentLang='it';var currentTranslations={};
function selectLang(lang){var prev=currentLang;currentLang=lang;var sel=document.getElementById('langSelect');if(sel)sel.value=lang;document.documentElement.setAttribute('lang',lang);document.documentElement.setAttribute('dir',lang==='ar'?'rtl':'ltr');fetch('/api/translations?lang='+lang).then(function(r){return r.json()}).then(function(t){currentTranslations=t;document.querySelectorAll('[data-i18n]').forEach(function(el){var key=el.dataset.i18n;if(t[key])el.textContent=t[key]})});if(prev!==lang)pushDataLayer({event:'language_change',from_lang:prev,to_lang:lang})}
fetch('/api/status').then(function(r){return r.json()}).then(function(d){var el=document.getElementById('lastUpdate');if(el&&d.last_update_display)el.textContent='Dati aggiornati al '+d.last_update_display}).catch(function(){});
fetch('/api/stats').then(function(r){return r.json()}).then(function(d){var el=document.getElementById('proofCounter');if(el&&d.scansioni)el.textContent=Number(d.scansioni).toLocaleString('it-IT')}).catch(function(){});
function acceptCookies(){localStorage.setItem('cookie_consent','accepted');document.getElementById('cookieBanner').style.display='none';loadGTM();pushDataLayer({event:'cookie_consent',consent:'accepted'})}
function rejectCookies(){localStorage.setItem('cookie_consent','rejected');document.getElementById('cookieBanner').style.display='none';pushDataLayer({event:'cookie_consent',consent:'rejected'})}
if(!localStorage.getItem('cookie_consent')){document.getElementById('cookieBanner').style.display='block'}
</script>`
}
