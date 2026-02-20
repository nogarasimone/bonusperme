package datasource

import (
	"bonusperme/internal/config"
	"bonusperme/internal/logger"
	"bonusperme/internal/models"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// DataSource defines the interface for an official data source.
type DataSource interface {
	Name() string
	Enabled() bool
	Fetch() ([]models.Bonus, error)
}

// Manager coordinates all data sources.
type Manager struct {
	sources []DataSource
	client  *http.Client
}

// NewManager creates a Manager with all configured data sources.
func NewManager() *Manager {
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	m := &Manager{client: client}

	if config.Cfg.DatasourceINPS {
		m.sources = append(m.sources, &INPSSource{client: client})
	}
	if config.Cfg.DatasourceAdE {
		m.sources = append(m.sources, &AdESource{client: client})
	}
	if config.Cfg.DatasourceMISE {
		m.sources = append(m.sources, &MISESource{client: client})
	}
	if config.Cfg.DatasourceGU {
		m.sources = append(m.sources, &GURSSSource{client: client})
	}
	if config.Cfg.DatasourceOpenAPI {
		m.sources = append(m.sources, &OpenDataSource{client: client})
	}

	return m
}

// FetchAll runs all enabled sources concurrently and returns combined results.
func (m *Manager) FetchAll() []models.Bonus {
	var (
		mu     sync.Mutex
		wg     sync.WaitGroup
		result []models.Bonus
	)

	for _, src := range m.sources {
		if !src.Enabled() {
			continue
		}
		wg.Add(1)
		go func(s DataSource) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					logger.Warn("datasource: panic", map[string]interface{}{
						"source": s.Name(), "error": fmt.Sprintf("%v", r),
					})
				}
			}()

			logger.Info("datasource: fetching", map[string]interface{}{"source": s.Name()})
			bonuses, err := s.Fetch()
			if err != nil {
				logger.Warn("datasource: error", map[string]interface{}{
					"source": s.Name(), "error": err.Error(),
				})
				return
			}
			logger.Info("datasource: done", map[string]interface{}{
				"source": s.Name(), "count": len(bonuses),
			})

			mu.Lock()
			result = append(result, bonuses...)
			mu.Unlock()
		}(src)
	}

	wg.Wait()
	return result
}

// Status returns a map of source name -> status info.
func (m *Manager) Status() map[string]interface{} {
	info := make(map[string]interface{})
	for _, s := range m.sources {
		info[s.Name()] = map[string]interface{}{
			"enabled": s.Enabled(),
		}
	}
	return info
}

// fetchURL is a shared helper for all sources.
func fetchURL(client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", config.Cfg.UserAgent)
	req.Header.Set("Accept-Language", "it-IT,it;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
}
