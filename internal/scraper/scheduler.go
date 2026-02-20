package scraper

import (
	"bonusperme/internal/config"
	"bonusperme/internal/datasource"
	"bonusperme/internal/logger"
	"bonusperme/internal/matcher"
	"bonusperme/internal/models"
	sentryutil "bonusperme/internal/sentry"
	"fmt"
	"sync"
	"time"
)

// SourceStatus tracks the status of each scraping source.
type SourceStatus struct {
	LastFetch  time.Time `json:"last_fetch"`
	Success    bool      `json:"success"`
	BonusFound int       `json:"bonus_found"`
	Error      string    `json:"error,omitempty"`
}

// BonusCache holds the cached bonus data and source status information.
type BonusCache struct {
	mu            sync.RWMutex
	bonus         []models.Bonus
	lastUpdate    time.Time
	updateCount   int
	sourcesStatus map[string]SourceStatus
}

var cache = &BonusCache{
	sourcesStatus: make(map[string]SourceStatus),
}

// OnScrapeComplete is called at the end of each scrape cycle.
// Set from main.go to propagate last-update time.
var OnScrapeComplete func(time.Time)

// --- Circuit Breaker ---

// CircuitState represents the state of a source circuit breaker.
type CircuitState string

const (
	CircuitClosed   CircuitState = "closed"
	CircuitOpen     CircuitState = "open"
	CircuitHalfOpen CircuitState = "half_open"
)

// SourceCircuit tracks circuit breaker state for a source.
type SourceCircuit struct {
	State       CircuitState `json:"state"`
	Failures    int          `json:"failures"`
	LastFailure time.Time    `json:"last_failure,omitempty"`
	NextRetryAt time.Time    `json:"next_retry_at,omitempty"`
}

var circuits sync.Map // map[string]*SourceCircuit

const (
	circuitFailureThreshold = 3
	circuitOpenCycles       = 3
)

// GetCircuit returns the circuit breaker state for a source, creating one if needed.
func GetCircuit(name string) *SourceCircuit {
	v, _ := circuits.LoadOrStore(name, &SourceCircuit{State: CircuitClosed})
	return v.(*SourceCircuit)
}

// RecordSuccess resets the circuit breaker for a source after a successful call.
func RecordSuccess(name string) {
	circuit := GetCircuit(name)
	circuit.State = CircuitClosed
	circuit.Failures = 0
}

// RecordFailure records a failure for a source and opens the circuit if threshold is reached.
func RecordFailure(name string, interval time.Duration) {
	circuit := GetCircuit(name)
	circuit.Failures++
	circuit.LastFailure = time.Now()
	if circuit.Failures >= circuitFailureThreshold {
		circuit.State = CircuitOpen
		circuit.NextRetryAt = time.Now().Add(interval * time.Duration(circuitOpenCycles))
		logger.Warn("scraper: circuit opened", map[string]interface{}{
			"source": name, "failures": circuit.Failures,
			"retry_at": circuit.NextRetryAt.Format(time.RFC3339),
		})
	}
}

// ShouldSkipSource checks if a source should be skipped due to an open circuit.
func ShouldSkipSource(name string) bool {
	circuit := GetCircuit(name)
	switch circuit.State {
	case CircuitOpen:
		if time.Now().After(circuit.NextRetryAt) {
			circuit.State = CircuitHalfOpen
			logger.Info("scraper: circuit half-open, attempting retry", map[string]interface{}{"source": name})
			return false
		}
		return true
	default:
		return false
	}
}

// --- Parsed Data Cache ---

type parsedEntry struct {
	Bonuses  []models.Bonus
	ParsedAt time.Time
}

var (
	parsedCacheMu sync.RWMutex
	parsedCache   = make(map[string]parsedEntry)
)

// --- Smart Scheduling ---

type sourceSchedule struct {
	NextRun time.Time
	Source  Source
}

var (
	scheduleMu sync.Mutex
	schedules  = make(map[string]*sourceSchedule)
)

func sourceInterval(trust float64) time.Duration {
	now := time.Now()
	isBudgetSeason := now.Month() == time.December || now.Month() == time.January || now.Month() == time.February

	switch {
	case trust >= 0.9: // institutional
		if isBudgetSeason {
			return 12 * time.Hour
		}
		return 24 * time.Hour
	case trust >= 0.6: // editorial
		if isBudgetSeason {
			return 6 * time.Hour
		}
		return 12 * time.Hour
	default:
		return 12 * time.Hour
	}
}

func isSourceDue(src Source) bool {
	scheduleMu.Lock()
	defer scheduleMu.Unlock()

	sched, ok := schedules[src.Name]
	if !ok {
		// First run — always due
		schedules[src.Name] = &sourceSchedule{Source: src}
		return true
	}
	return time.Now().After(sched.NextRun)
}

func markSourceRun(src Source) {
	scheduleMu.Lock()
	defer scheduleMu.Unlock()

	interval := sourceInterval(src.Trust)
	schedules[src.Name] = &sourceSchedule{
		NextRun: time.Now().Add(interval),
		Source:  src,
	}
}

// StartScheduler runs an initial scrape and then re-scrapes at a base interval.
// Individual sources are checked against their smart schedule.
func StartScheduler() {
	if !config.Cfg.ScraperEnabled {
		logger.Info("scraper: disabled via config", nil)
		return
	}

	go func() {
		RunScrape()
		// Use a shorter tick to allow per-source scheduling
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			RunScrape()
		}
	}()
}

// RunScrape performs a full scrape cycle across all sources + datasource Manager.
func RunScrape() {
	logger.Info("scraper: starting scrape cycle", nil)
	sources := GetSources()
	var allScraped []models.Bonus

	// Snapshot old bonus data for change detection
	cache.mu.RLock()
	oldBonuses := make([]models.Bonus, len(cache.bonus))
	copy(oldBonuses, cache.bonus)
	cache.mu.RUnlock()

	// 1. Legacy scraper sources with circuit breaker + smart scheduling
	for _, src := range sources {
		// Circuit breaker check
		if ShouldSkipSource(src.Name) {
			logger.Info("scraper: skipping (circuit open)", map[string]interface{}{"source": src.Name})
			continue
		}

		// Smart scheduling check
		if !isSourceDue(src) {
			logger.Info("scraper: skipping (not due)", map[string]interface{}{"source": src.Name})
			// Still include parsed cache if available
			parsedCacheMu.RLock()
			if entry, ok := parsedCache[src.URL]; ok {
				allScraped = append(allScraped, entry.Bonuses...)
			}
			parsedCacheMu.RUnlock()
			continue
		}

		logger.Info("scraper: fetching source", map[string]interface{}{"source": src.Name, "url": src.URL})

		// Try fetching with HTTP cache
		body, changed, err := GetHTTPCache().FetchWithCache(src.URL, httpClient)
		if err != nil {
			logger.Error("scraper: fetch error", map[string]interface{}{"source": src.Name, "error": err.Error()})
			interval := sourceInterval(src.Trust)
			RecordFailure(src.Name, interval)

			status := SourceStatus{
				LastFetch: time.Now(),
				Success:   false,
				Error:     err.Error(),
			}
			cache.mu.Lock()
			cache.sourcesStatus[src.Name] = status
			cache.mu.Unlock()

			// Use parsed cache as fallback
			parsedCacheMu.RLock()
			if entry, ok := parsedCache[src.URL]; ok {
				allScraped = append(allScraped, entry.Bonuses...)
			}
			parsedCacheMu.RUnlock()
			continue
		}

		// If HTTP content not changed, use parsed cache
		if !changed {
			parsedCacheMu.RLock()
			entry, ok := parsedCache[src.URL]
			parsedCacheMu.RUnlock()
			if ok {
				logger.Info("scraper: using parsed cache (not changed)", map[string]interface{}{"source": src.Name})
				allScraped = append(allScraped, entry.Bonuses...)
				RecordSuccess(src.Name)
				markSourceRun(src)
				continue
			}
		}

		// Parse the body
		bonuses := ParseSourceFromBody(src, body)

		status := SourceStatus{
			LastFetch:  time.Now(),
			Success:    true,
			BonusFound: len(bonuses),
		}

		if len(bonuses) == 0 {
			status.Error = "no bonuses found"
			sentryutil.CaptureError(fmt.Errorf("scraper: 0 bonuses from %s", src.Name), map[string]string{"source": src.Name})
		}

		cache.mu.Lock()
		cache.sourcesStatus[src.Name] = status
		cache.mu.Unlock()

		// Update parsed cache
		parsedCacheMu.Lock()
		parsedCache[src.URL] = parsedEntry{Bonuses: bonuses, ParsedAt: time.Now()}
		parsedCacheMu.Unlock()

		RecordSuccess(src.Name)
		markSourceRun(src)

		allScraped = append(allScraped, bonuses...)
		logger.Info("scraper: source complete", map[string]interface{}{"source": src.Name, "found": len(bonuses)})
	}

	// 2. Official data sources via datasource.Manager
	mgr := datasource.NewManager()
	officialBonuses := mgr.FetchAll()
	if len(officialBonuses) > 0 {
		allScraped = append(allScraped, officialBonuses...)
		logger.Info("scraper: official datasources", map[string]interface{}{"found": len(officialBonuses)})
	}

	hardcoded := matcher.GetAllBonus()
	enriched := EnrichBonusData(allScraped, hardcoded)

	// Detect changes before updating cache
	if len(oldBonuses) > 0 {
		DetectChanges(oldBonuses, enriched)
	}

	cache.mu.Lock()
	cache.bonus = enriched
	cache.lastUpdate = time.Now()
	cache.updateCount++
	cache.mu.Unlock()

	logger.Info("scraper: cache updated", map[string]interface{}{"total": len(enriched), "cycle": cache.updateCount})

	if OnScrapeComplete != nil {
		OnScrapeComplete(time.Now())
	}
}

// GetCachedBonus returns the cached list of bonuses.
// Falls back to hardcoded if cache is empty.
func GetCachedBonus() []models.Bonus {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	if len(cache.bonus) == 0 {
		return matcher.GetAllBonus()
	}
	result := make([]models.Bonus, len(cache.bonus))
	copy(result, cache.bonus)
	return result
}

// GetScraperStatus returns a status map with scraper state and per-source info.
func GetScraperStatus() map[string]interface{} {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	sources := make(map[string]SourceStatus)
	for k, v := range cache.sourcesStatus {
		sources[k] = v
	}

	// Circuit breaker states
	circuitStates := make(map[string]SourceCircuit)
	circuits.Range(func(key, value interface{}) bool {
		circuitStates[key.(string)] = *value.(*SourceCircuit)
		return true
	})

	// HTTP cache stats
	httpStats := GetHTTPCache().GetCacheStats()

	// Parsed cache stats
	parsedCacheMu.RLock()
	parsedCacheSize := len(parsedCache)
	parsedCacheMu.RUnlock()

	return map[string]interface{}{
		"last_run":          cache.lastUpdate,
		"next_run":          cache.lastUpdate.Add(config.Cfg.ScraperInterval),
		"bonus_count":       len(cache.bonus),
		"update_count":      cache.updateCount,
		"sources":           sources,
		"circuits":          circuitStates,
		"http_cache_stats":  httpStats,
		"parsed_cache_size": parsedCacheSize,
	}
}
