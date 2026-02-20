package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Cfg is the global configuration loaded at startup.
var Cfg Config

// Config holds all application configuration.
type Config struct {
	// Server
	Port    string
	BaseURL string

	// Sentry
	SentryDSN         string
	SentryEnvironment string
	SentryRelease     string

	// Analytics
	GTMID string

	// Scraper
	ScraperEnabled  bool
	ScraperInterval time.Duration

	// Rate limiter
	RateLimitRPS   int
	RateLimitBurst int

	// Link checker
	LinkCheckEnabled  bool
	LinkCheckInterval time.Duration
	LinkCheckDelay    time.Duration

	// Data sources
	DatasourceINPS    bool
	DatasourceAdE     bool
	DatasourceMISE    bool
	DatasourceGU      bool
	DatasourceOpenAPI bool

	// HTTP
	UserAgent string

	// Gzip
	GzipEnabled bool

	// Turnstile
	TurnstileSiteKey   string
	TurnstileSecretKey string

	// Web3Forms
	Web3FormsAccessKey string

	// Validity checker
	ValidityCheckEnabled bool
	NewsCheckEnabled     bool
	NewsCheckInterval    time.Duration
	AdminAPIKey          string

	// Pipeline (5-level verification)
	PipelineEnabled    bool
	PipelineL1Enabled  bool
	PipelineL2Enabled  bool
	PipelineL3Enabled  bool
	PipelineL4Enabled  bool
	PipelineL1Interval time.Duration
	PipelineL3Interval time.Duration
	PipelineL4Interval time.Duration
	QuorumConferma       float64
	QuorumScadenza       float64
	QuorumModificaImporto float64
	QuorumTriggerL2      float64
	QuorumTriggerL4      float64
	QuorumMinFonti       int
}

// Load reads .env (if present) and populates Cfg from environment variables.
func Load() {
	if err := godotenv.Load(); err != nil {
		log.Println("config: no .env file found, using environment variables")
	}

	Cfg = Config{
		Port:    envOr("PORT", "8080"),
		BaseURL: envOr("BASE_URL", "https://bonusperme.it"),

		SentryDSN:         os.Getenv("SENTRY_DSN"),
		SentryEnvironment: envOr("SENTRY_ENVIRONMENT", "production"),
		SentryRelease:     envOr("SENTRY_RELEASE", "bonusperme@1.0.0"),

		GTMID: envOr("GTM_ID", ""),

		ScraperEnabled:  envBool("SCRAPER_ENABLED", true),
		ScraperInterval: envDuration("SCRAPER_INTERVAL", 24*time.Hour),

		RateLimitRPS:   envInt("RATE_LIMIT_RPS", 30),
		RateLimitBurst: envInt("RATE_LIMIT_BURST", 60),

		LinkCheckEnabled:  envBool("LINKCHECK_ENABLED", true),
		LinkCheckInterval: envDuration("LINKCHECK_INTERVAL", 24*time.Hour),
		LinkCheckDelay:    envDuration("LINKCHECK_DELAY", 5*time.Second),

		DatasourceINPS:    envBool("DATASOURCE_INPS", true),
		DatasourceAdE:     envBool("DATASOURCE_ADE", true),
		DatasourceMISE:    envBool("DATASOURCE_MISE", true),
		DatasourceGU:      envBool("DATASOURCE_GU", true),
		DatasourceOpenAPI: envBool("DATASOURCE_OPENAPI", true),

		UserAgent: envOr("USER_AGENT", "Mozilla/5.0 (compatible; BonusPerMeBot/1.0; +https://bonusperme.it)"),

		GzipEnabled: envBool("GZIP_ENABLED", true),

		TurnstileSiteKey:   os.Getenv("TURNSTILE_SITE_KEY"),
		TurnstileSecretKey: os.Getenv("TURNSTILE_SECRET_KEY"),

		Web3FormsAccessKey: os.Getenv("WEB3FORMS_ACCESS_KEY"),

		ValidityCheckEnabled: envBool("VALIDITY_CHECK_ENABLED", true),
		NewsCheckEnabled:     envBool("NEWS_CHECK_ENABLED", false),
		NewsCheckInterval:    envDuration("NEWS_CHECK_INTERVAL", 6*time.Hour),
		AdminAPIKey:          os.Getenv("ADMIN_API_KEY"),

		PipelineEnabled:    envBool("PIPELINE_ENABLED", false),
		PipelineL1Enabled:  envBool("PIPELINE_L1_ENABLED", true),
		PipelineL2Enabled:  envBool("PIPELINE_L2_ENABLED", false),
		PipelineL3Enabled:  envBool("PIPELINE_L3_ENABLED", true),
		PipelineL4Enabled:  envBool("PIPELINE_L4_ENABLED", true),
		PipelineL1Interval: envDuration("PIPELINE_L1_INTERVAL", 2*time.Hour),
		PipelineL3Interval: envDuration("PIPELINE_L3_INTERVAL", 3*time.Hour),
		PipelineL4Interval: envDuration("PIPELINE_L4_INTERVAL", 12*time.Hour),
		QuorumConferma:       envFloat64("QUORUM_CONFERMA", 2.0),
		QuorumScadenza:       envFloat64("QUORUM_SCADENZA", 2.5),
		QuorumModificaImporto: envFloat64("QUORUM_MODIFICA_IMPORTO", 3.0),
		QuorumTriggerL2:      envFloat64("QUORUM_TRIGGER_L2", 2.0),
		QuorumTriggerL4:      envFloat64("QUORUM_TRIGGER_L4", 1.5),
		QuorumMinFonti:       envInt("QUORUM_MIN_FONTI", 2),
	}

	log.Printf("config: loaded (port=%s, scraper=%v, linkcheck=%v, gtm=%s)",
		Cfg.Port, Cfg.ScraperEnabled, Cfg.LinkCheckEnabled, maskGTM(Cfg.GTMID))
}

func maskGTM(id string) string {
	if id == "" {
		return "(disabled)"
	}
	return id
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envFloat64(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
