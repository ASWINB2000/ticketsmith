// Package openproject implements tracker.Tracker for OpenProject's REST API v3.
package openproject

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"ticketsmith/internal/tracker"
)

// Client is a tracker.Tracker implementation backed by OpenProject's API v3.
type Client struct {
	baseURL string
	token   string
	http    *http.Client

	mu              sync.RWMutex
	typesCache      []tracker.TicketType
	projectsCache   []tracker.Project
	assigneesCache  map[string][]tracker.User
	prioritiesCache []tracker.Priority
}

// NewClient constructs an OpenProject client. baseURL is the root of the
// OpenProject instance (e.g. "https://projects.example.com").
func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL:        strings.TrimRight(baseURL, "/"),
		token:          token,
		http:           &http.Client{},
		assigneesCache: map[string][]tracker.User{},
	}
}

// InvalidateCache clears cached metadata, forcing the next GetTypes/
// GetProjects/GetAssignees call to re-fetch from OpenProject.
func (c *Client) InvalidateCache() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.typesCache = nil
	c.projectsCache = nil
	c.assigneesCache = map[string][]tracker.User{}
	c.prioritiesCache = nil
}

type opErrorBody struct {
	Message string `json:"message"`
}

func (c *Client) doRequest(ctx context.Context, method, path string, body, out interface{}) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("openproject: marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("openproject: build request %s %s: %w", method, path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	auth := base64.StdEncoding.EncodeToString([]byte("apikey:" + c.token))
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("openproject %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("openproject %s %s: read response: %w", method, path, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := string(respBody)
		var opErr opErrorBody
		if json.Unmarshal(respBody, &opErr) == nil && opErr.Message != "" {
			msg = opErr.Message
		}
		return fmt.Errorf("openproject %s %s: %s (status %d)", method, path, msg, resp.StatusCode)
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("openproject %s %s: parse response: %w", method, path, err)
		}
	}
	return nil
}
