package linkcheck

import (
	"bonusperme/internal/config"
	"bonusperme/internal/logger"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"
)

type waybackResponse struct {
	ArchivedSnapshots struct {
		Closest struct {
			Available bool   `json:"available"`
			URL       string `json:"url"`
			Timestamp string `json:"timestamp"`
			Status    string `json:"status"`
		} `json:"closest"`
	} `json:"archived_snapshots"`
}

var waybackClient = &http.Client{
	Timeout: 10 * time.Second,
}

// TryWaybackRecovery attempts to find an archived version of the URL via the Wayback Machine.
// Returns the archived URL and true if found, empty string and false otherwise.
func TryWaybackRecovery(rawURL string) (string, bool) {
	apiURL := "https://archive.org/wayback/available?url=" + url.QueryEscape(rawURL)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", false
	}
	req.Header.Set("User-Agent", config.Cfg.UserAgent)

	resp, err := waybackClient.Do(req)
	if err != nil {
		logger.Warn("wayback: request failed", map[string]interface{}{
			"url": rawURL, "error": err.Error(),
		})
		return "", false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", false
	}

	var wb waybackResponse
	if err := json.Unmarshal(body, &wb); err != nil {
		return "", false
	}

	snap := wb.ArchivedSnapshots.Closest
	if snap.Available && snap.URL != "" {
		logger.Info("wayback: found archived snapshot", map[string]interface{}{
			"original": rawURL, "archived": snap.URL, "timestamp": snap.Timestamp,
		})
		return snap.URL, true
	}

	return "", false
}
