package main

import (
	"bonusperme/internal/blog"
	"bonusperme/internal/config"
	"bonusperme/internal/handlers"
	"bonusperme/internal/i18n"
	"bonusperme/internal/linkcheck"
	"bonusperme/internal/logger"
	"bonusperme/internal/matcher"
	"bonusperme/internal/middleware"
	"bonusperme/internal/models"
	"bonusperme/internal/pipeline"
	"bonusperme/internal/scraper"
	sentryutil "bonusperme/internal/sentry"
	"bonusperme/internal/validity"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	// Load configuration from .env and environment variables
	config.Load()

	// Initialize Sentry (non-blocking if SENTRY_DSN is empty)
	sentryutil.Init()
	defer sentryutil.Flush()

	// Initialize persistent counter
	handlers.InitCounter()

	// Wire scraper callback to track last update time
	scraper.OnScrapeComplete = func(t time.Time) {
		handlers.SetLastScrape(t)
	}

	// Start scraper scheduler (respects SCRAPER_ENABLED config)
	scraper.StartScheduler()

	// Load blog posts
	if err := blog.LoadAll("content/blog"); err != nil {
		log.Printf("blog: %v", err)
	}

	// Connect i18n translations to handler
	handlers.SetTranslationLoader(i18n.GetAll)

	// Rate limiter from config
	limiter := handlers.NewRateLimiter(
		config.Cfg.RateLimitRPS,
		config.Cfg.RateLimitBurst,
		time.Second,
	)

	// Create mux
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/match", handlers.MatchHandler)
	mux.HandleFunc("/api/stats", handlers.StatsHandler)
	mux.HandleFunc("/api/health", handlers.HealthDetailedHandler)
	mux.HandleFunc("/api/parse-isee", handlers.ParseISEEHandler)
	mux.HandleFunc("/api/calendar", handlers.CalendarHandler)
	mux.HandleFunc("/api/simulate", handlers.SimulateHandler)
	mux.HandleFunc("/api/report", handlers.ReportHandler)
	mux.HandleFunc("/api/notify-signup", handlers.NotifySignupHandler)
	mux.HandleFunc("/api/analytics", handlers.AnalyticsHandler)
	mux.HandleFunc("/api/analytics-summary", handlers.AnalyticsSummaryHandler)
	mux.HandleFunc("/api/scraper-status", handlers.ScraperStatusHandler)
	mux.HandleFunc("/api/status", handlers.StatusHandler)
	mux.HandleFunc("/api/translations", handlers.TranslationsHandler)

	// New API routes
	mux.HandleFunc("/api/encode-profile", handlers.EncodeProfileHandler)
	mux.HandleFunc("/api/decode-profile", handlers.DecodeProfileHandler)
	mux.HandleFunc("/api/bonus", handlers.BonusListHandler)
	mux.HandleFunc("/api/bonus/", handlers.BonusDetailHandler)

	// Admin routes (protected by ADMIN_API_KEY)
	mux.HandleFunc("/api/admin/alerts", validity.AdminAlertsHandler)
	mux.HandleFunc("/api/admin/bonus-status", validity.AdminBonusStatusHandler)
	mux.HandleFunc("/api/admin/links", linkcheck.AdminLinksHandler)
	mux.HandleFunc("/api/admin/changes", scraper.AdminChangesHandler)

	// Pages
	mux.HandleFunc("/per-caf", handlers.PerCAFHandler)
	mux.HandleFunc("/contatti", handlers.ContattiHandler)
	mux.HandleFunc("/api/contact", handlers.ContactHandler)
	mux.HandleFunc("/api/caf-signup", handlers.CAFSignupHandler)
	mux.HandleFunc("/privacy", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/privacy.html")
	})
	mux.HandleFunc("/cookie-policy", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/cookie-policy.html")
	})

	// Blog / guide routes
	mux.HandleFunc("/guide", handlers.BlogListHandler)
	mux.HandleFunc("/guide/", handlers.BlogPostHandler)

	// SEO routes
	mux.HandleFunc("/bonus/", handlers.BonusPageHandler)
	mux.HandleFunc("/sitemap.xml", handlers.SitemapHandler)
	mux.HandleFunc("/robots.txt", handlers.RobotsTxtHandler)

	// Serve static files (index.html served via template handler for GTM injection)
	mux.HandleFunc("/index.html", handlers.IndexHandler)
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))
	// Root handler: serve index.html via template for GTM, fallback to static for other files
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			handlers.IndexHandler(w, r)
			return
		}
		// Block dotfile paths (.env, .git, etc.)
		if strings.Contains(r.URL.Path, "/.") {
			handlers.NotFoundHandler(w, r)
			return
		}
		// Check if static file exists, otherwise serve 404
		if _, err := os.Stat("static" + r.URL.Path); err != nil {
			handlers.NotFoundHandler(w, r)
			return
		}
		fs.ServeHTTP(w, r)
	})

	// Wrap with middleware: Recovery → SecurityHeaders → Gzip (if enabled) → Rate Limiter
	var handler http.Handler = limiter.Middleware(mux)
	if config.Cfg.GzipEnabled {
		handler = middleware.Gzip(handler)
	}
	handler = middleware.SecurityHeaders(handler)
	handler = middleware.Recovery(handler)

	// Link check at boot (background, respects config)
	if config.Cfg.LinkCheckEnabled {
		go func() {
			time.Sleep(config.Cfg.LinkCheckDelay)
			allBonus := matcher.GetAllBonusWithRegional()
			ptrs := make([]*models.Bonus, len(allBonus))
			for i := range allBonus {
				ptrs[i] = &allBonus[i]
			}
			broken := linkcheck.CheckAllLinks(ptrs)
			if broken > 0 {
				logger.Warn("link check: broken links found at boot", map[string]interface{}{"broken": broken})
			}
			handlers.SetLastScrape(time.Now())
		}()

		// Periodic link check
		go func() {
			ticker := time.NewTicker(config.Cfg.LinkCheckInterval)
			defer ticker.Stop()
			for range ticker.C {
				allBonus := matcher.GetAllBonusWithRegional()
				ptrs := make([]*models.Bonus, len(allBonus))
				for i := range allBonus {
					ptrs[i] = &allBonus[i]
				}
				linkcheck.CheckAllLinks(ptrs)
				handlers.SetLastScrape(time.Now())
			}
		}()
	}

	// Validity check at boot + daily (respects config)
	if config.Cfg.ValidityCheckEnabled {
		go func() {
			time.Sleep(10 * time.Second)
			allBonus := matcher.GetAllBonusWithRegional()
			validity.RunCheck(allBonus)
		}()

		// Daily midnight validity check
		go func() {
			for {
				now := time.Now()
				next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
				time.Sleep(time.Until(next))
				allBonus := matcher.GetAllBonusWithRegional()
				validity.RunCheck(allBonus)
			}
		}()
	}

	// News check (respects config, off by default)
	if config.Cfg.NewsCheckEnabled {
		go func() {
			time.Sleep(30 * time.Second)
			allBonus := matcher.GetAllBonusWithRegional()
			validity.RunNewsCheck(allBonus)
		}()

		go func() {
			ticker := time.NewTicker(config.Cfg.NewsCheckInterval)
			defer ticker.Stop()
			for range ticker.C {
				allBonus := matcher.GetAllBonusWithRegional()
				validity.RunNewsCheck(allBonus)
			}
		}()
	}

	// Pipeline (5-level verification, respects PIPELINE_ENABLED config)
	if config.Cfg.PipelineEnabled {
		allBonusPipeline := matcher.GetAllBonusWithRegional()
		orch := pipeline.NewOrchestrator(allBonusPipeline)
		orch.Start()
		mux.HandleFunc("/api/admin/pipeline", orch.AdminStatusHandler)
		logger.Info("pipeline: started", nil)
	}

	logger.Info("server starting", map[string]interface{}{"port": config.Cfg.Port})
	fmt.Printf("BonusPerMe running on http://localhost:%s\n", config.Cfg.Port)
	log.Fatal(http.ListenAndServe(":"+config.Cfg.Port, handler))
}
