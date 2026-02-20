package handlers

import (
	"bonusperme/internal/config"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type turnstileResponse struct {
	Success bool `json:"success"`
}

// verifyTurnstile validates a Cloudflare Turnstile token.
// Returns true if verification passes or if no secret key is configured (dev mode).
func verifyTurnstile(token string) bool {
	secret := config.Cfg.TurnstileSecretKey
	if secret == "" {
		return true // dev mode — skip verification
	}
	if token == "" {
		return false
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify",
		url.Values{
			"secret":   {secret},
			"response": {token},
		})
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	var result turnstileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false
	}
	return result.Success
}

// getTurnstileToken extracts the Turnstile token from the request header.
func getTurnstileToken(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-Turnstile-Token"))
}
