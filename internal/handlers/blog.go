package handlers

import (
	"bonusperme/internal/blog"
	"bonusperme/internal/config"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ---------- Italian date formatting ----------

var blogMonthNames = []string{
	"", "gennaio", "febbraio", "marzo", "aprile", "maggio", "giugno",
	"luglio", "agosto", "settembre", "ottobre", "novembre", "dicembre",
}

func formatBlogDate(t time.Time) string {
	return fmt.Sprintf("%d %s %d", t.Day(), blogMonthNames[t.Month()], t.Year())
}

// ---------- Category metadata ----------

type categoryInfo struct {
	Slug  string
	Label string
	Color string // CSS color for badge
}

var categories = []categoryInfo{
	{"famiglia", "Famiglia", "#2A6B45"},
	{"sostegno", "Sostegno", "#C0522E"},
	{"casa", "Casa", "#1B3A54"},
	{"lavoro", "Lavoro", "#6B21A8"},
	{"procedure", "Procedure", "#0369A1"},
	{"fiscale", "Fiscale", "#B45309"},
}

func categoryLabel(slug string) string {
	for _, c := range categories {
		if c.Slug == slug {
			return c.Label
		}
	}
	return slug
}

func categoryColor(slug string) string {
	for _, c := range categories {
		if c.Slug == slug {
			return c.Color
		}
	}
	return "#666"
}

// ---------- BlogListHandler ----------

func BlogListHandler(w http.ResponseWriter, r *http.Request) {
	catFilter := r.URL.Query().Get("cat")

	var posts []blog.Post
	if catFilter != "" {
		posts = blog.GetByCategory(catFilter)
	} else {
		posts = blog.GetAll()
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var sb strings.Builder

	title := "Guide ai Bonus 2026 — BonusPerMe"
	desc := "Guide complete su bonus, agevolazioni fiscali e aiuti economici in Italia. Requisiti, importi, scadenze e come fare domanda."

	sb.WriteString(`<!DOCTYPE html>
<html lang="it">
<head>
` + SharedMetaTags(title, desc, "/guide") + `
<style>` + SharedCSS() + blogListCSS() + `</style>
</head>
<body>
`)
	sb.WriteString(SharedTopbar())
	sb.WriteString(SharedHeader("/guide"))

	sb.WriteString(`<main class="container" style="padding-top:32px;padding-bottom:48px">`)

	// Page header
	sb.WriteString(`<div class="blog-header">
<h1>Guide ai Bonus e Agevolazioni</h1>
<p>Tutto quello che devi sapere su bonus, detrazioni fiscali e aiuti economici in Italia: requisiti, importi, scadenze e istruzioni per fare domanda.</p>
</div>`)

	// Category pills
	sb.WriteString(`<nav class="cat-nav" aria-label="Filtra per categoria">`)
	if catFilter == "" {
		sb.WriteString(`<a href="/guide" class="cat-pill cat-pill--active">Tutte</a>`)
	} else {
		sb.WriteString(`<a href="/guide" class="cat-pill">Tutte</a>`)
	}
	for _, c := range categories {
		active := ""
		if catFilter == c.Slug {
			active = " cat-pill--active"
		}
		sb.WriteString(`<a href="/guide?cat=` + c.Slug + `" class="cat-pill` + active + `">` + c.Label + `</a>`)
	}
	sb.WriteString(`</nav>`)

	// Article cards
	if len(posts) == 0 {
		sb.WriteString(`<p style="text-align:center;color:var(--ink-50);padding:40px 0">Nessuna guida trovata per questa categoria.</p>`)
	} else {
		sb.WriteString(`<div class="blog-grid">`)
		for _, p := range posts {
			blogCardHTML(&sb, p)
		}
		sb.WriteString(`</div>`)
	}

	// JSON-LD CollectionPage
	sb.WriteString(`<script type="application/ld+json">{
"@context":"https://schema.org",
"@type":"CollectionPage",
"name":"Guide ai Bonus e Agevolazioni",
"description":"` + htmlEscape(desc) + `",
"url":"` + config.Cfg.BaseURL + `/guide",
"itemListElement":[`)
	for i, p := range posts {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf(`{"@type":"ListItem","position":%d,"url":"%s/guide/%s","name":"%s"}`,
			i+1, config.Cfg.BaseURL, p.Slug, htmlEscape(p.Title)))
	}
	sb.WriteString(`]}</script>`)

	sb.WriteString(`</main>`)
	sb.WriteString(SharedFooter())
	sb.WriteString(SharedCookieBanner())
	sb.WriteString(SharedScripts())
	sb.WriteString(`</body></html>`)

	w.Write([]byte(sb.String()))
}

func blogCardHTML(sb *strings.Builder, p blog.Post) {
	excerpt := p.Description
	if len(excerpt) > 160 {
		excerpt = excerpt[:157] + "..."
	}

	sb.WriteString(`<article class="blog-card">`)
	sb.WriteString(`<a href="/guide/` + htmlEscape(p.Slug) + `" class="blog-card-link">`)
	sb.WriteString(`<div class="blog-card-meta">`)
	sb.WriteString(`<span class="blog-cat-badge" style="background:` + categoryColor(p.Category) + `">` + categoryLabel(p.Category) + `</span>`)
	sb.WriteString(`<time datetime="` + p.Date.Format("2006-01-02") + `">` + formatBlogDate(p.Date) + `</time>`)
	sb.WriteString(`</div>`)
	sb.WriteString(`<h2 class="blog-card-title">` + htmlEscape(p.Title) + `</h2>`)
	sb.WriteString(`<p class="blog-card-excerpt">` + htmlEscape(excerpt) + `</p>`)
	sb.WriteString(`<span class="blog-card-cta">Leggi la guida &rarr;</span>`)
	sb.WriteString(`</a></article>`)
}

// ---------- BlogPostHandler ----------

func BlogPostHandler(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/guide/")
	slug = strings.TrimSuffix(slug, "/")

	if slug == "" {
		BlogListHandler(w, r)
		return
	}

	post := blog.GetBySlug(slug)
	if post == nil {
		NotFoundHandler(w, r)
		return
	}

	// Related articles (same category, excluding current)
	related := blog.GetByCategory(post.Category)
	var relatedFiltered []blog.Post
	for _, rp := range related {
		if rp.Slug != post.Slug {
			relatedFiltered = append(relatedFiltered, rp)
		}
	}
	if len(relatedFiltered) > 4 {
		relatedFiltered = relatedFiltered[:4]
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var sb strings.Builder

	pageTitle := post.Title + " — BonusPerMe"

	sb.WriteString(`<!DOCTYPE html>
<html lang="it">
<head>
` + SharedMetaTags(pageTitle, post.Description, "/guide/"+post.Slug) + `
<style>` + SharedCSS() + blogPostCSS() + `</style>
</head>
<body>
`)
	sb.WriteString(SharedTopbar())
	sb.WriteString(SharedHeader("/guide"))

	sb.WriteString(`<main class="container" style="padding-top:24px;padding-bottom:48px">`)

	// Breadcrumb
	sb.WriteString(`<nav class="breadcrumb" aria-label="Breadcrumb">`)
	sb.WriteString(`<a href="/">Home</a> <span>&rsaquo;</span> `)
	sb.WriteString(`<a href="/guide">Guide</a> <span>&rsaquo;</span> `)
	sb.WriteString(`<span class="breadcrumb-current">` + htmlEscape(post.Title) + `</span>`)
	sb.WriteString(`</nav>`)

	sb.WriteString(`<div class="blog-layout">`)

	// Article
	sb.WriteString(`<article class="blog-article">`)
	sb.WriteString(`<header class="blog-article-header">`)
	sb.WriteString(`<span class="blog-cat-badge" style="background:` + categoryColor(post.Category) + `">` + categoryLabel(post.Category) + `</span>`)
	sb.WriteString(`<h1>` + htmlEscape(post.Title) + `</h1>`)
	sb.WriteString(`<div class="blog-article-meta">`)
	sb.WriteString(`<time datetime="` + post.Date.Format("2006-01-02") + `">` + formatBlogDate(post.Date) + `</time>`)
	sb.WriteString(` &middot; <span>` + htmlEscape(post.Author) + `</span>`)
	sb.WriteString(`</div></header>`)
	sb.WriteString(`<div class="blog-content">` + post.HTMLContent + `</div>`)
	sb.WriteString(`</article>`)

	// Sidebar
	if len(relatedFiltered) > 0 {
		sb.WriteString(`<aside class="blog-sidebar">`)
		sb.WriteString(`<h3>Guide correlate</h3>`)
		for _, rp := range relatedFiltered {
			sb.WriteString(`<a href="/guide/` + htmlEscape(rp.Slug) + `" class="sidebar-link">`)
			sb.WriteString(`<span class="sidebar-title">` + htmlEscape(rp.Title) + `</span>`)
			sb.WriteString(`<span class="sidebar-date">` + formatBlogDate(rp.Date) + `</span>`)
			sb.WriteString(`</a>`)
		}
		sb.WriteString(`<a href="/guide" class="sidebar-all">Tutte le guide &rarr;</a>`)
		sb.WriteString(`</aside>`)
	}

	sb.WriteString(`</div>`) // blog-layout

	// JSON-LD Article
	sb.WriteString(`<script type="application/ld+json">{
"@context":"https://schema.org",
"@type":"Article",
"headline":"` + htmlEscape(post.Title) + `",
"description":"` + htmlEscape(post.Description) + `",
"datePublished":"` + post.Date.Format("2006-01-02") + `",
"author":{"@type":"Organization","name":"` + htmlEscape(post.Author) + `"},
"publisher":{"@type":"Organization","name":"BonusPerMe","url":"` + config.Cfg.BaseURL + `"},
"mainEntityOfPage":"` + config.Cfg.BaseURL + `/guide/` + post.Slug + `"
}</script>`)

	sb.WriteString(`</main>`)
	sb.WriteString(SharedFooter())
	sb.WriteString(SharedCookieBanner())
	sb.WriteString(SharedScripts())
	sb.WriteString(`</body></html>`)

	w.Write([]byte(sb.String()))
}

// ---------- CSS ----------

func blogListCSS() string {
	return `
.blog-header{text-align:center;margin-bottom:28px}
.blog-header h1{font-size:1.8rem;color:var(--blue);margin-bottom:8px}
.blog-header p{color:var(--ink-50);max-width:600px;margin:0 auto;font-size:.95rem}
.cat-nav{display:flex;gap:8px;flex-wrap:wrap;justify-content:center;margin-bottom:28px}
.cat-pill{display:inline-block;padding:6px 16px;border-radius:20px;font-size:.82rem;font-weight:500;color:var(--ink-50);background:var(--ink-05);text-decoration:none;transition:all .15s}
.cat-pill:hover{background:var(--blue-light);color:var(--blue);text-decoration:none}
.cat-pill--active{background:var(--blue);color:#fff}
.cat-pill--active:hover{background:var(--blue);color:#fff}
.blog-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(300px,1fr));gap:20px}
.blog-card{background:#fff;border-radius:var(--radius-lg);border:1px solid var(--ink-15);overflow:hidden;transition:box-shadow .15s,transform .15s}
.blog-card:hover{box-shadow:var(--shadow-card);transform:translateY(-2px)}
.blog-card-link{display:block;padding:20px;text-decoration:none;color:inherit}
.blog-card-link:hover{text-decoration:none}
.blog-card-meta{display:flex;align-items:center;gap:10px;margin-bottom:10px;font-size:.78rem}
.blog-cat-badge{display:inline-block;padding:2px 10px;border-radius:12px;font-size:.72rem;font-weight:600;color:#fff}
.blog-card-meta time{color:var(--ink-50)}
.blog-card-title{font-family:'DM Serif Display',Georgia,serif;font-size:1.15rem;font-weight:400;color:var(--ink);margin-bottom:8px;line-height:1.35}
.blog-card-excerpt{font-size:.88rem;color:var(--ink-75);line-height:1.55;margin-bottom:12px}
.blog-card-cta{font-size:.82rem;font-weight:600;color:var(--blue-mid)}
@media(max-width:640px){.blog-grid{grid-template-columns:1fr}.blog-header h1{font-size:1.4rem}}
`
}

func blogPostCSS() string {
	return `
.breadcrumb{font-size:.82rem;color:var(--ink-50);margin-bottom:20px}
.breadcrumb a{color:var(--blue-mid);text-decoration:none}
.breadcrumb a:hover{text-decoration:underline}
.breadcrumb-current{color:var(--ink-75)}
.blog-layout{display:grid;grid-template-columns:1fr 280px;gap:32px;align-items:start}
.blog-article-header{margin-bottom:24px}
.blog-article-header h1{font-size:1.8rem;color:var(--blue);margin:8px 0 12px;line-height:1.25}
.blog-article-meta{font-size:.82rem;color:var(--ink-50)}
.blog-cat-badge{display:inline-block;padding:2px 10px;border-radius:12px;font-size:.72rem;font-weight:600;color:#fff}
.blog-content{font-size:.95rem;line-height:1.75;color:var(--ink)}
.blog-content h2{font-size:1.3rem;color:var(--blue);margin:28px 0 12px;padding-bottom:6px;border-bottom:1px solid var(--ink-15)}
.blog-content h3{font-size:1.1rem;color:var(--ink);margin:20px 0 8px}
.blog-content p{margin-bottom:14px}
.blog-content ul,.blog-content ol{margin-bottom:14px;padding-left:24px}
.blog-content li{margin-bottom:6px}
.blog-content a{color:var(--blue-mid);text-decoration:underline}
.blog-content a:hover{color:var(--blue)}
.blog-content strong{font-weight:600}
.blog-content blockquote{border-left:3px solid var(--blue);padding:12px 16px;margin:16px 0;background:var(--blue-light);border-radius:0 var(--radius) var(--radius) 0;font-size:.9rem;color:var(--ink-75)}
.blog-content table{width:100%;border-collapse:collapse;margin:16px 0;font-size:.88rem}
.blog-content th,.blog-content td{padding:8px 12px;border:1px solid var(--ink-15);text-align:left}
.blog-content th{background:var(--ink-05);font-weight:600}
.blog-sidebar{position:sticky;top:70px}
.blog-sidebar h3{font-family:'DM Serif Display',Georgia,serif;font-size:1rem;color:var(--ink);margin-bottom:12px;font-weight:400}
.sidebar-link{display:block;padding:10px 12px;border-radius:var(--radius);text-decoration:none;margin-bottom:6px;transition:background .15s}
.sidebar-link:hover{background:var(--ink-05);text-decoration:none}
.sidebar-title{display:block;font-size:.85rem;color:var(--ink);font-weight:500;line-height:1.35}
.sidebar-date{display:block;font-size:.75rem;color:var(--ink-50);margin-top:2px}
.sidebar-all{display:block;margin-top:12px;font-size:.82rem;font-weight:600;color:var(--blue-mid);text-decoration:none}
.sidebar-all:hover{text-decoration:underline}
@media(max-width:768px){.blog-layout{grid-template-columns:1fr}.blog-sidebar{position:static;margin-top:32px;padding-top:24px;border-top:1px solid var(--ink-15)}.blog-article-header h1{font-size:1.4rem}}
`
}
