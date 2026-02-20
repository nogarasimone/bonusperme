package pipeline

import (
	"bonusperme/internal/config"
	"bonusperme/internal/logger"
	"bonusperme/internal/models"
	"bonusperme/internal/scraper"
	"bonusperme/internal/validity"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Orchestrator coordinates the 5-level verification pipeline.
type Orchestrator struct {
	guWatcher     *GUWatcher
	normattiva    *NormattivaFetcher
	rssMonitor    *RSSMonitor
	institutional *InstitutionalScraper

	bonuses   []models.Bonus
	bonusByID map[string]*models.Bonus
	mu        sync.RWMutex

	guEvents  chan GUEvent
	rssResults chan CorroborationResult
	triggerL2 chan string // bonusID
	triggerL4 chan string // bonusID
	stopCh    chan struct{}

	// Status tracking
	levelStatus map[string]*LevelStatus
	statusMu    sync.RWMutex
}

// NewOrchestrator creates all sub-components and wires them together.
func NewOrchestrator(bonuses []models.Bonus) *Orchestrator {
	byID := make(map[string]*models.Bonus)
	bonusCopy := make([]models.Bonus, len(bonuses))
	copy(bonusCopy, bonuses)
	for i := range bonusCopy {
		byID[bonusCopy[i].ID] = &bonusCopy[i]
	}

	o := &Orchestrator{
		guWatcher:     NewGUWatcher(bonuses),
		normattiva:    NewNormattivaFetcher(),
		rssMonitor:    NewRSSMonitor(bonuses),
		institutional: NewInstitutionalScraper(bonuses),

		bonuses:   bonusCopy,
		bonusByID: byID,

		guEvents:   make(chan GUEvent, 100),
		rssResults: make(chan CorroborationResult, 100),
		triggerL2:  make(chan string, 50),
		triggerL4:  make(chan string, 50),
		stopCh:     make(chan struct{}),

		levelStatus: map[string]*LevelStatus{
			"L1_GU":            {Enabled: config.Cfg.PipelineL1Enabled},
			"L2_Normattiva":    {Enabled: config.Cfg.PipelineL2Enabled},
			"L3_RSS":           {Enabled: config.Cfg.PipelineL3Enabled},
			"L4_Institutional": {Enabled: config.Cfg.PipelineL4Enabled},
		},
	}

	return o
}

// Start launches the pipeline tickers and event loop.
func (o *Orchestrator) Start() {
	logger.Info("pipeline: starting orchestrator", map[string]interface{}{
		"L1": config.Cfg.PipelineL1Enabled,
		"L2": config.Cfg.PipelineL2Enabled,
		"L3": config.Cfg.PipelineL3Enabled,
		"L4": config.Cfg.PipelineL4Enabled,
	})

	// L1 GU ticker
	if config.Cfg.PipelineL1Enabled {
		go func() {
			// Initial run after short delay
			time.Sleep(15 * time.Second)
			o.runL1()

			ticker := time.NewTicker(config.Cfg.PipelineL1Interval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					o.runL1()
				case <-o.stopCh:
					return
				}
			}
		}()
	}

	// L3 RSS ticker
	if config.Cfg.PipelineL3Enabled {
		go func() {
			time.Sleep(30 * time.Second)
			o.runL3()

			ticker := time.NewTicker(config.Cfg.PipelineL3Interval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					o.runL3()
				case <-o.stopCh:
					return
				}
			}
		}()
	}

	// L4 Institutional ticker
	if config.Cfg.PipelineL4Enabled {
		go func() {
			time.Sleep(60 * time.Second)
			o.runL4All()

			ticker := time.NewTicker(config.Cfg.PipelineL4Interval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					o.runL4All()
				case <-o.stopCh:
					return
				}
			}
		}()
	}

	// Event loop
	go o.eventLoop()
}

// Stop gracefully shuts down the orchestrator.
func (o *Orchestrator) Stop() {
	close(o.stopCh)
}

func (o *Orchestrator) runL1() {
	o.updateLevelStatus("L1_GU", func(s *LevelStatus) { s.RunCount++ })

	events := o.guWatcher.Check()
	now := time.Now()
	o.updateLevelStatus("L1_GU", func(s *LevelStatus) { s.LastRun = &now; s.LastErr = "" })

	for _, evt := range events {
		// Non-blocking send
		select {
		case o.guEvents <- evt:
		default:
			logger.Warn("pipeline: guEvents channel full, dropping event", map[string]interface{}{
				"bonus": evt.BonusID, "type": string(evt.Type),
			})
		}
	}
}

func (o *Orchestrator) runL3() {
	o.updateLevelStatus("L3_RSS", func(s *LevelStatus) { s.RunCount++ })

	o.mu.RLock()
	bonusCopy := make([]models.Bonus, len(o.bonuses))
	copy(bonusCopy, o.bonuses)
	o.mu.RUnlock()

	results := o.rssMonitor.RunFullCycle(bonusCopy)
	now := time.Now()
	o.updateLevelStatus("L3_RSS", func(s *LevelStatus) { s.LastRun = &now; s.LastErr = "" })

	for _, r := range results {
		select {
		case o.rssResults <- r:
		default:
			logger.Warn("pipeline: rssResults channel full, dropping", map[string]interface{}{
				"bonus": r.BonusID,
			})
		}
	}
}

func (o *Orchestrator) runL4All() {
	o.updateLevelStatus("L4_Institutional", func(s *LevelStatus) { s.RunCount++ })

	o.mu.RLock()
	ids := make([]string, 0, len(o.bonusByID))
	for id := range o.bonusByID {
		ids = append(ids, id)
	}
	o.mu.RUnlock()

	for _, id := range ids {
		if scraper.ShouldSkipSource("L4_" + id) {
			continue
		}
		o.runL4Single(id)
	}

	now := time.Now()
	o.updateLevelStatus("L4_Institutional", func(s *LevelStatus) { s.LastRun = &now; s.LastErr = "" })
}

func (o *Orchestrator) runL4Single(bonusID string) {
	result, err := o.institutional.Scrape(bonusID)
	if err != nil {
		scraper.RecordFailure("L4_"+bonusID, config.Cfg.PipelineL4Interval)
		return
	}

	scraper.RecordSuccess("L4_" + bonusID)

	now := time.Now()
	o.updateBonusField(bonusID, func(b *models.Bonus) {
		b.UltimaVerificaSito = &now
		if result.LinkOK {
			b.LinkVerificato = true
			b.LinkVerificatoAl = now.Format("2006-01-02")
		}
		b.ConfidenceScore = CalculateConfidence(b)
	})
}

func (o *Orchestrator) eventLoop() {
	for {
		select {
		case evt := <-o.guEvents:
			o.handleGUEvent(evt)

		case result := <-o.rssResults:
			o.handleRSSResult(result)

		case bonusID := <-o.triggerL2:
			o.handleL2Trigger(bonusID)

		case bonusID := <-o.triggerL4:
			o.handleL4Trigger(bonusID)

		case <-o.stopCh:
			logger.Info("pipeline: event loop stopped", nil)
			return
		}
	}
}

func (o *Orchestrator) handleGUEvent(evt GUEvent) {
	logger.Info("pipeline: handling GU event", map[string]interface{}{
		"bonus": evt.BonusID, "type": string(evt.Type), "ref": evt.NormRef,
	})

	now := time.Now()
	o.updateBonusField(evt.BonusID, func(b *models.Bonus) {
		b.UltimaVerificaGU = &now
	})

	switch evt.Type {
	case GUAbrogazione:
		o.updateBonusField(evt.BonusID, func(b *models.Bonus) {
			b.Scaduto = true
			b.Stato = "scaduto"
			b.StatoValidita = "scaduto"
			b.MotivoStato = "Abrogazione pubblicata in GU: " + evt.GUTitle
		})
		validity.AddAlert(validity.Alert{
			BonusID:   evt.BonusID,
			OldStato:  "attivo",
			NewStato:  "scaduto",
			Motivo:    "Abrogazione GU: " + evt.NormRef,
			Timestamp: time.Now(),
			Urgenza:   "alta",
		})

	case GUProroga, GURifinanziamento:
		o.updateBonusField(evt.BonusID, func(b *models.Bonus) {
			b.AnnoConferma = now.Year()
			b.ConfidenceScore = CalculateConfidence(b)
		})
		validity.AddAlert(validity.Alert{
			BonusID:   evt.BonusID,
			OldStato:  "attivo",
			NewStato:  "attivo",
			Motivo:    "Proroga/rifinanziamento GU: " + evt.NormRef,
			Timestamp: time.Now(),
			Urgenza:   "bassa",
		})
		// Also trigger L2 for detail extraction
		o.sendTriggerL2(evt.BonusID)

	case GUModifica, GUConversione, GUAttuazione:
		// Trigger L2 to fetch updated text
		o.sendTriggerL2(evt.BonusID)

	case GUNuovo:
		validity.AddAlert(validity.Alert{
			BonusID:   evt.BonusID,
			OldStato:  "",
			NewStato:  "nuovo",
			Motivo:    "Nuovo provvedimento GU: " + evt.GUTitle,
			Timestamp: time.Now(),
			Urgenza:   "media",
		})
	}
}

func (o *Orchestrator) handleRSSResult(result CorroborationResult) {
	logger.Info("pipeline: handling RSS result", map[string]interface{}{
		"bonus":  result.BonusID,
		"action": string(result.Action),
		"reason": result.TriggerReason,
	})

	switch result.Action {
	case ActionConfirm:
		now := time.Now()
		o.updateBonusField(result.BonusID, func(b *models.Bonus) {
			b.FontiCorroborate = len(result.Sources)
			b.UltimaVerificaRSS = &now
			b.ConfidenceScore = CalculateConfidence(b)
		})

	case ActionMarkExpired:
		validity.AddAlert(validity.Alert{
			BonusID:   result.BonusID,
			OldStato:  "attivo",
			NewStato:  "potenzialmente_scaduto",
			Motivo:    result.TriggerReason,
			Timestamp: time.Now(),
			Urgenza:   "alta",
		})
		// Also trigger L4 to verify on institutional site
		o.sendTriggerL4(result.BonusID)

	case ActionTriggerL2:
		o.sendTriggerL2(result.BonusID)

	case ActionTriggerL4:
		o.sendTriggerL4(result.BonusID)

	case ActionAlertManual:
		validity.AddAlert(validity.Alert{
			BonusID:   result.BonusID,
			OldStato:  "attivo",
			NewStato:  "verifica_manuale",
			Motivo:    result.TriggerReason,
			Timestamp: time.Now(),
			Urgenza:   "media",
		})
	}
}

func (o *Orchestrator) handleL2Trigger(bonusID string) {
	if !config.Cfg.PipelineL2Enabled {
		logger.Info("pipeline: L2 disabled, skipping", map[string]interface{}{"bonus": bonusID})
		return
	}

	if scraper.ShouldSkipSource("L2_normattiva") {
		logger.Info("pipeline: L2 circuit open, skipping", map[string]interface{}{"bonus": bonusID})
		return
	}

	// Find norm refs for this bonus
	o.mu.RLock()
	bonus, ok := o.bonusByID[bonusID]
	var refs []string
	if ok {
		refs = bonus.RiferimentiNormativi
	}
	o.mu.RUnlock()

	if !ok || len(refs) == 0 {
		return
	}

	for _, rawRef := range refs {
		normRef := NormalizeNormRef(rawRef)
		result, err := o.normattiva.FetchConsolidated(normRef)
		if err != nil {
			scraper.RecordFailure("L2_normattiva", config.Cfg.PipelineL3Interval)
			logger.Warn("pipeline: L2 fetch failed", map[string]interface{}{
				"bonus": bonusID, "ref": normRef, "error": err.Error(),
			})
			continue
		}

		scraper.RecordSuccess("L2_normattiva")

		// Update bonus with extracted data
		if len(result.ImportiTrovati) > 0 {
			o.updateBonusField(bonusID, func(b *models.Bonus) {
				b.ImportoConfermato = result.ImportiTrovati[0].Valore
			})
		}
		if len(result.ISEETrovate) > 0 {
			o.updateBonusField(bonusID, func(b *models.Bonus) {
				b.ISEEConfermata = result.ISEETrovate[0].Valore
			})
		}

		o.updateBonusField(bonusID, func(b *models.Bonus) {
			b.ConfidenceScore = CalculateConfidence(b)
		})
	}

	// After L2, also trigger L4
	o.sendTriggerL4(bonusID)
}

func (o *Orchestrator) handleL4Trigger(bonusID string) {
	if !config.Cfg.PipelineL4Enabled {
		return
	}

	if scraper.ShouldSkipSource("L4_" + bonusID) {
		return
	}

	o.runL4Single(bonusID)
}

// updateBonusField applies a thread-safe mutation to a bonus.
func (o *Orchestrator) updateBonusField(bonusID string, fn func(*models.Bonus)) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if b, ok := o.bonusByID[bonusID]; ok {
		fn(b)
	}
}

func (o *Orchestrator) sendTriggerL2(bonusID string) {
	select {
	case o.triggerL2 <- bonusID:
	default:
		logger.Warn("pipeline: triggerL2 channel full", map[string]interface{}{"bonus": bonusID})
	}
}

func (o *Orchestrator) sendTriggerL4(bonusID string) {
	select {
	case o.triggerL4 <- bonusID:
	default:
		logger.Warn("pipeline: triggerL4 channel full", map[string]interface{}{"bonus": bonusID})
	}
}

func (o *Orchestrator) updateLevelStatus(level string, fn func(*LevelStatus)) {
	o.statusMu.Lock()
	defer o.statusMu.Unlock()
	if s, ok := o.levelStatus[level]; ok {
		fn(s)
	}
}

// AdminStatusHandler serves GET /api/admin/pipeline.
// Protected by ADMIN_API_KEY.
func (o *Orchestrator) AdminStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check admin key
	key := config.Cfg.AdminAPIKey
	if key != "" {
		qKey := r.URL.Query().Get("key")
		hKey := r.Header.Get("X-Admin-Key")
		if qKey != key && hKey != key {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	o.statusMu.RLock()
	levels := make(map[string]LevelStatus)
	for k, v := range o.levelStatus {
		levels[k] = *v
	}
	o.statusMu.RUnlock()

	// Channel buffer usage
	channelStatus := map[string]interface{}{
		"gu_events_buffered":   len(o.guEvents),
		"rss_results_buffered": len(o.rssResults),
		"trigger_l2_buffered":  len(o.triggerL2),
		"trigger_l4_buffered":  len(o.triggerL4),
	}

	result := map[string]interface{}{
		"enabled":  config.Cfg.PipelineEnabled,
		"levels":   levels,
		"channels": channelStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(result)
}
