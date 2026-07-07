// Package openaicompat implements ai.Provider against any OpenAI-compatible
// chat completions endpoint (Groq, OpenAI, or a self-hosted compatible server).
package openaicompat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"ticketsmith/internal/ai"
	"ticketsmith/internal/templates"
)

// Client is an ai.Provider implementation for OpenAI-compatible chat completions APIs.
type Client struct {
	baseURL string
	apiKey  string
	model   string
	http    *http.Client
}

// New constructs a client. baseURL is the API root (e.g.
// "https://api.groq.com/openai/v1" or "https://api.openai.com/v1").
func New(baseURL, apiKey, model string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		http:    &http.Client{},
	}
}

type modelListResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// ListModels fetches the available model IDs from an OpenAI-compatible
// endpoint's GET /models. Not all self-hosted compatible servers implement
// this, so callers should treat failure as "not supported" rather than fatal.
func ListModels(ctx context.Context, baseURL, apiKey string) ([]string, error) {
	baseURL = strings.TrimRight(baseURL, "/")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("ai: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ai: request models: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ai: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := string(body)
		var apiErr apiErrorBody
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Error.Message != "" {
			msg = apiErr.Error.Message
		}
		return nil, fmt.Errorf("ai: models: %s (status %d)", msg, resp.StatusCode)
	}

	var parsed modelListResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("ai: parse models response: %w", err)
	}

	ids := make([]string, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		ids = append(ids, m.ID)
	}
	sort.Strings(ids)
	return ids, nil
}

// Ping validates that the client's base URL and API key are usable, without
// spending a completion token — it hits the same /models endpoint as
// ListModels and only checks that the call succeeds.
func (c *Client) Ping(ctx context.Context) error {
	_, err := ListModels(ctx, c.baseURL, c.apiKey)
	return err
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatRequest struct {
	Model          string         `json:"model"`
	ResponseFormat responseFormat `json:"response_format"`
	Messages       []chatMessage  `json:"messages"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
}

type apiErrorBody struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// GenerateTicket asks the model to produce a StructuredTicket in JSON object
// mode (chosen over json_schema/strict mode for the widest compatibility
// across OpenAI, Groq, and self-hosted OpenAI-compatible servers).
func (c *Client) GenerateTicket(ctx context.Context, tmpl templates.Template, rawInput string) (ai.StructuredTicket, error) {
	reqBody := chatRequest{
		Model:          c.model,
		ResponseFormat: responseFormat{Type: "json_object"},
		Messages: []chatMessage{
			{Role: "system", Content: buildSystemPrompt(tmpl)},
			{Role: "user", Content: rawInput},
		},
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return ai.StructuredTicket{}, fmt.Errorf("ai: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return ai.StructuredTicket{}, fmt.Errorf("ai: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return ai.StructuredTicket{}, fmt.Errorf("ai: request chat/completions: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ai.StructuredTicket{}, fmt.Errorf("ai: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := string(respBody)
		var apiErr apiErrorBody
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Error.Message != "" {
			msg = apiErr.Error.Message
		}
		return ai.StructuredTicket{}, fmt.Errorf("ai: chat/completions: %s (status %d)", msg, resp.StatusCode)
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return ai.StructuredTicket{}, fmt.Errorf("ai: parse response: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return ai.StructuredTicket{}, fmt.Errorf("ai: response had no choices")
	}

	content := stripJSONFence(chatResp.Choices[0].Message.Content)

	var ticket ai.StructuredTicket
	if err := json.Unmarshal([]byte(content), &ticket); err != nil {
		snippet := content
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return ai.StructuredTicket{}, fmt.Errorf("ai: parse structured ticket from model output: %w (content: %q)", err, snippet)
	}
	return ticket, nil
}

// stripJSONFence defensively removes ```json ... ``` fences some models add
// even when asked for plain json_object output.
func stripJSONFence(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	return strings.TrimSpace(content)
}

func buildSystemPrompt(tmpl templates.Template) string {
	var b strings.Builder
	b.WriteString("You are a ticket-writing assistant. Given freeform notes from a user, output ONLY a JSON object ")
	b.WriteString("(no markdown fences, no extra text) with exactly these keys:\n")
	b.WriteString(`- "subject": string, a concise ticket title` + "\n")
	b.WriteString(`- "description": string, a clear description in markdown` + "\n")
	b.WriteString(`- "fields": an object with the following keys, each a string value:` + "\n")
	for _, f := range tmpl.FieldsSchema {
		fmt.Fprintf(&b, "  - %q", f.Name)
		if f.Label != "" {
			fmt.Fprintf(&b, " (%s)", f.Label)
		}
		if f.Description != "" {
			fmt.Fprintf(&b, ": %s", f.Description)
		}
		b.WriteString("\n")
	}
	if tmpl.AIInstructions != "" {
		b.WriteString("\nAdditional instructions:\n")
		b.WriteString(tmpl.AIInstructions)
	}
	return b.String()
}
