// Package announcement checks a remote JSON manifest for an in-app
// announcement banner (e.g. "provider X is being sunset, switch to Y") and
// tracks which one the user has already dismissed, so editing the manifest
// is enough to notify users without shipping a new app release.
package announcement

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Config points at the remote announcement manifest.
type Config struct {
	URL        string
	HTTPClient *http.Client // nil uses a default client; tests only
}

// Manifest is a single in-app announcement. Fields are exported and
// untagged — like updater.UpdateInfo, this struct round-trips through the
// Wails JS<->Go boundary, and encoding/json silently drops unexported
// fields — so the remote JSON's keys must match these names exactly
// (ID, Title, Body, Level, URL).
type Manifest struct {
	ID    string // change this to push a new (or re-show an already-dismissed) announcement
	Title string
	Body  string
	Level string // "info" or "warning"; anything else is treated as "info"
	URL   string // optional "learn more" link
}

func httpClient(cfg Config) *http.Client {
	if cfg.HTTPClient != nil {
		return cfg.HTTPClient
	}
	return &http.Client{Timeout: 10 * time.Second}
}

// fetch downloads and decodes the manifest at cfg.URL. Returns (nil, nil)
// if the manifest has no ID — an empty/placeholder manifest means "no
// announcement right now", not an error.
func fetch(ctx context.Context, cfg Config) (*Manifest, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("announcement: build request: %w", err)
	}
	resp, err := httpClient(cfg).Do(req)
	if err != nil {
		return nil, fmt.Errorf("announcement: fetch manifest: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("announcement: manifest request failed: status %d", resp.StatusCode)
	}
	var m Manifest
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, fmt.Errorf("announcement: decode manifest: %w", err)
	}
	if m.ID == "" {
		return nil, nil
	}
	return &m, nil
}

// Check fetches the current manifest and returns it only if its ID differs
// from lastDismissedID — i.e. there's something the user hasn't already
// dismissed. Returns (nil, nil) when there's nothing new.
func Check(ctx context.Context, cfg Config, lastDismissedID string) (*Manifest, error) {
	m, err := fetch(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if m == nil || m.ID == lastDismissedID {
		return nil, nil
	}
	return m, nil
}
