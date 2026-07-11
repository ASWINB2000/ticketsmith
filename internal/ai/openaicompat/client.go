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
	"strconv"
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

// Ping validates that the client's base URL and API key are usable. It
// prefers GET /models, which spends no completion token, but that endpoint
// is optional in the OpenAI-compatible ecosystem — providers fronting Azure
// OpenAI (e.g. GitHub Models) don't implement it — so on failure it falls
// back to a minimal (1-token) chat completion, the one call every
// OpenAI-compatible provider must support.
func (c *Client) Ping(ctx context.Context) error {
	if _, err := ListModels(ctx, c.baseURL, c.apiKey); err == nil {
		return nil
	}

	maxTokens := 1
	_, _, err := c.chatCompletion(ctx, chatRequest{
		MaxTokens: &maxTokens,
		Messages:  []chatMessage{{Role: "user", Content: "hi"}},
	})
	return err
}

// Usage reports rate-limit/quota info from the provider's most recent
// response headers — the "x-ratelimit-*" convention started by OpenAI and
// echoed by Groq, Azure OpenAI, and (via Azure) GitHub Models. It's not part
// of the OpenAI-compatible spec, so self-hosted or other servers may omit
// it entirely — check Supported before trusting the numeric fields.
type Usage struct {
	Supported         bool   `json:"supported"`
	RequestsLimit     int    `json:"requestsLimit"`
	RequestsRemaining int    `json:"requestsRemaining"`
	TokensLimit       int    `json:"tokensLimit"`
	TokensRemaining   int    `json:"tokensRemaining"`
	ResetRequests     string `json:"resetRequests"`
	ResetTokens       string `json:"resetTokens"`
}

// Usage fetches current rate-limit/usage info via a minimal (1-token) chat
// completion — the one endpoint every OpenAI-compatible provider supports —
// and reads it off the response headers.
func (c *Client) Usage(ctx context.Context) (Usage, error) {
	maxTokens := 1
	_, headers, err := c.chatCompletion(ctx, chatRequest{
		MaxTokens: &maxTokens,
		Messages:  []chatMessage{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		return Usage{}, err
	}
	return parseUsage(headers), nil
}

func parseUsage(h http.Header) Usage {
	atoi := func(key string) int {
		n, _ := strconv.Atoi(h.Get(key))
		return n
	}
	return Usage{
		Supported:         h.Get("x-ratelimit-limit-requests") != "" || h.Get("x-ratelimit-limit-tokens") != "",
		RequestsLimit:     atoi("x-ratelimit-limit-requests"),
		RequestsRemaining: atoi("x-ratelimit-remaining-requests"),
		TokensLimit:       atoi("x-ratelimit-limit-tokens"),
		TokensRemaining:   atoi("x-ratelimit-remaining-tokens"),
		ResetRequests:     h.Get("x-ratelimit-reset-requests"),
		ResetTokens:       h.Get("x-ratelimit-reset-tokens"),
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatRequest struct {
	Model          string          `json:"model"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
	Messages       []chatMessage   `json:"messages"`
	Temperature    *float64        `json:"temperature,omitempty"`
	MaxTokens      *int            `json:"max_tokens,omitempty"`
}

// ticketTemperature is deliberately low: GenerateTicket is a structured
// extraction task, not creative writing, and provider defaults (e.g. Groq's
// temperature=1) were producing incoherent, self-repeating field values.
var ticketTemperature = 0.3

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

// chatCompletion posts reqBody (with Model filled in) to /chat/completions
// and returns the parsed response along with the raw response headers (so
// callers like Usage can read rate-limit headers) — the request/response
// plumbing shared by GenerateTicket, Rephrase, Ping, and Usage.
func (c *Client) chatCompletion(ctx context.Context, reqBody chatRequest) (chatResponse, http.Header, error) {
	reqBody.Model = c.model

	b, err := json.Marshal(reqBody)
	if err != nil {
		return chatResponse{}, nil, fmt.Errorf("ai: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return chatResponse{}, nil, fmt.Errorf("ai: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return chatResponse{}, nil, fmt.Errorf("ai: request chat/completions: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return chatResponse{}, nil, fmt.Errorf("ai: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := string(respBody)
		var apiErr apiErrorBody
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Error.Message != "" {
			msg = apiErr.Error.Message
		}
		return chatResponse{}, resp.Header, fmt.Errorf("ai: chat/completions: %s (status %d)", msg, resp.StatusCode)
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return chatResponse{}, nil, fmt.Errorf("ai: parse response: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return chatResponse{}, nil, fmt.Errorf("ai: response had no choices")
	}
	return chatResp, resp.Header, nil
}

// GenerateTicket asks the model to produce a StructuredTicket in JSON object
// mode (chosen over json_schema/strict mode for the widest compatibility
// across OpenAI, Groq, and self-hosted OpenAI-compatible servers).
func (c *Client) GenerateTicket(ctx context.Context, tmpl templates.Template, rawInput string) (ai.StructuredTicket, error) {
	chatResp, _, err := c.chatCompletion(ctx, chatRequest{
		ResponseFormat: &responseFormat{Type: "json_object"},
		Temperature:    &ticketTemperature,
		Messages: []chatMessage{
			{Role: "system", Content: buildSystemPrompt(tmpl)},
			{Role: "user", Content: rawInput},
		},
	})
	if err != nil {
		return ai.StructuredTicket{}, err
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

// Rephrase combines one or more freeform notes into a single coherent
// draft. Unlike GenerateTicket, this is plain text in and out — no
// response_format, no schema to parse, just the trimmed completion content.
func (c *Client) Rephrase(ctx context.Context, notes []string) (string, error) {
	if len(notes) == 0 {
		return "", fmt.Errorf("ai: rephrase requires at least one note")
	}

	chatResp, _, err := c.chatCompletion(ctx, chatRequest{
		Messages: []chatMessage{
			{Role: "system", Content: buildRephraseSystemPrompt()},
			{Role: "user", Content: buildRephraseUserMessage(notes)},
		},
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}

func buildRephraseSystemPrompt() string {
	return "You are a writing assistant that turns one or more freeform, informally-written notes " +
		"into a single clear, coherent draft suitable as the raw input for a downstream ticket-writing " +
		"system. Combine the notes into one flowing piece of prose (or a short bulleted list if the " +
		"source notes are themselves itemized) that preserves every distinct point made across all of " +
		"them — do not drop details, do not invent new ones, and do not comment on the merge itself. " +
		"If given only a single note, just clean it up: fix grammar and tighten the wording without " +
		"changing its meaning or adding anything new. Reply with ONLY the resulting text — no markdown " +
		"fences, no preamble, no explanation."
}

// buildRephraseUserMessage joins N notes with numbered separators so the
// model can address each distinct point instead of treating the whole
// block as one blob of text.
func buildRephraseUserMessage(notes []string) string {
	if len(notes) == 1 {
		return notes[0]
	}
	var b strings.Builder
	b.WriteString("Combine these notes:\n\n")
	for i, n := range notes {
		fmt.Fprintf(&b, "Note %d:\n%s\n\n", i+1, n)
	}
	return strings.TrimRight(b.String(), "\n")
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
	b.WriteString("\nWriting standard: write like an experienced QA/product analyst producing polished, ")
	b.WriteString("submission-ready documentation — never compress a field into a single line, a sentence ")
	b.WriteString("fragment, or a vague restatement of its label. \"description\" must be one or more full ")
	b.WriteString("paragraphs covering what the issue or work is, where/how it shows up, and why it matters — ")
	b.WriteString("not a one-line summary. Every field you choose to populate should be written as complete ")
	b.WriteString("sentences, or as a structured markdown list/table when the content is naturally itemized ")
	b.WriteString("(e.g. multiple steps, multiple field/value pairs) — detailed enough that someone with zero ")
	b.WriteString("prior context on the raw input could still act on it correctly. Use markdown formatting ")
	b.WriteString("(bold, bullet or numbered lists, tables) inside \"description\" and fields wherever it aids ")
	b.WriteString("clarity, not just prose. Still: never invent specifics (numbers, names, exact values) the ")
	b.WriteString("raw input didn't provide — elaborate on what's there, don't fabricate what isn't. If the raw ")
	b.WriteString("input truly doesn't cover a field, leave it as an empty string rather than guessing or ")
	b.WriteString("padding it with filler.")
	if tmpl.AIInstructions != "" {
		b.WriteString("\n\nAdditional instructions:\n")
		b.WriteString(tmpl.AIInstructions)
	}
	return b.String()
}
