package openaicompat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ticketsmith/internal/ai"
	"ticketsmith/internal/templates"
)

func TestGenerateTicketSendsExpectedRequestAndParsesResponse(t *testing.T) {
	var captured chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %q, want /chat/completions", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Errorf("Authorization = %q, want Bearer sk-test", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"subject\":\"Login fails\",\"description\":\"Users can't log in\",\"fields\":{\"steps\":\"click login\"}}"}}]}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "sk-test", "llama-3.1-8b-instant")
	tmpl := templates.Template{
		Name:            "Bug Report",
		TrackerTypeName: "Bug",
		FieldsSchema: []templates.Field{
			{Name: "steps", Label: "Steps to reproduce", Type: "textarea"},
		},
		AIInstructions: "Be concise and specific.",
	}

	ticket, err := c.GenerateTicket(context.Background(), tmpl, "login is broken for everyone")
	if err != nil {
		t.Fatalf("GenerateTicket: %v", err)
	}

	if ticket.Subject != "Login fails" || ticket.Fields["steps"] != "click login" {
		t.Errorf("unexpected ticket: %+v", ticket)
	}

	if captured.Model != "llama-3.1-8b-instant" {
		t.Errorf("Model = %q", captured.Model)
	}
	if captured.ResponseFormat.Type != "json_object" {
		t.Errorf("ResponseFormat.Type = %q, want json_object", captured.ResponseFormat.Type)
	}
	if len(captured.Messages) != 2 {
		t.Fatalf("Messages len = %d, want 2", len(captured.Messages))
	}
	system := captured.Messages[0].Content
	if !strings.Contains(system, "steps") || !strings.Contains(system, "Be concise and specific.") {
		t.Errorf("system prompt missing field name / instructions: %q", system)
	}
	if captured.Messages[1].Content != "login is broken for everyone" {
		t.Errorf("user message = %q", captured.Messages[1].Content)
	}
}

func TestGenerateTicketStripsJSONFence(t *testing.T) {
	content := "```json\n{\"subject\":\"S\",\"description\":\"D\",\"fields\":{}}\n```"
	respBody, err := json.Marshal(chatResponse{
		Choices: []chatChoice{{Message: chatMessage{Content: content}}},
	})
	if err != nil {
		t.Fatalf("marshal test fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(respBody)
	}))
	defer srv.Close()

	c := New(srv.URL, "sk-test", "model")
	ticket, err := c.GenerateTicket(context.Background(), templates.Template{}, "raw")
	if err != nil {
		t.Fatalf("GenerateTicket: %v", err)
	}
	if ticket.Subject != "S" || ticket.Description != "D" {
		t.Errorf("unexpected ticket after fence stripping: %+v", ticket)
	}
}

func TestGenerateTicketErrorsOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "bad-key", "model")
	_, err := c.GenerateTicket(context.Background(), templates.Template{}, "raw")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid api key") || !strings.Contains(err.Error(), "401") {
		t.Errorf("error = %q, want it to contain \"invalid api key\" and \"401\"", err.Error())
	}
}

func TestRephraseSendsExpectedRequestAndParsesResponse(t *testing.T) {
	var captured chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %q, want /chat/completions", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"  Combined draft text.  "}}]}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "sk-test", "llama-3.1-8b-instant")
	draft, err := c.Rephrase(context.Background(), []string{"first note", "second note"})
	if err != nil {
		t.Fatalf("Rephrase: %v", err)
	}
	if draft != "Combined draft text." {
		t.Errorf("draft = %q", draft)
	}

	if captured.ResponseFormat != nil {
		t.Errorf("ResponseFormat = %+v, want nil (omitted)", captured.ResponseFormat)
	}
	if len(captured.Messages) != 2 {
		t.Fatalf("Messages len = %d, want 2", len(captured.Messages))
	}
	user := captured.Messages[1].Content
	if !strings.Contains(user, "Note 1:") || !strings.Contains(user, "Note 2:") {
		t.Errorf("user message missing numbered notes: %q", user)
	}
	if !strings.Contains(user, "first note") || !strings.Contains(user, "second note") {
		t.Errorf("user message missing note contents: %q", user)
	}
}

func TestRephraseSingleNoteOmitsNumbering(t *testing.T) {
	var captured chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Write([]byte(`{"choices":[{"message":{"content":"cleaned up note"}}]}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "sk-test", "model")
	if _, err := c.Rephrase(context.Background(), []string{"a single messy note"}); err != nil {
		t.Fatalf("Rephrase: %v", err)
	}

	if captured.Messages[1].Content != "a single messy note" {
		t.Errorf("user message = %q, want the note verbatim with no numbering", captured.Messages[1].Content)
	}
}

func TestRephraseErrorsOnEmptyInput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("expected no HTTP call for empty input")
	}))
	defer srv.Close()

	c := New(srv.URL, "sk-test", "model")
	if _, err := c.Rephrase(context.Background(), nil); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGenerateTicketErrorsOnMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"content":"not json at all"}}]}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "sk-test", "model")
	_, err := c.GenerateTicket(context.Background(), templates.Template{}, "raw")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not json at all") {
		t.Errorf("error = %q, want it to include a content snippet", err.Error())
	}
}

// suggestInstructionsWithResponse runs SuggestInstructions against a stub
// server that always replies with the given message content.
func suggestInstructionsWithResponse(t *testing.T, content string) (ai.TuningSuggestion, error) {
	t.Helper()
	respBody, err := json.Marshal(chatResponse{
		Choices: []chatChoice{{Message: chatMessage{Content: content}}},
	})
	if err != nil {
		t.Fatalf("marshal test fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(respBody)
	}))
	defer srv.Close()

	c := New(srv.URL, "sk-test", "model")
	examples := []ai.EditExample{{
		Generated: ai.StructuredTicket{Subject: "before"},
		Final:     ai.StructuredTicket{Subject: "after"},
	}}
	return c.SuggestInstructions(context.Background(), templates.Template{Name: "Bug"}, examples)
}

func TestSuggestInstructionsParsesStringFields(t *testing.T) {
	got, err := suggestInstructionsWithResponse(t,
		`{"summary":"- you shorten subjects","suggestedInstructions":"Write short subjects."}`)
	if err != nil {
		t.Fatalf("SuggestInstructions: %v", err)
	}
	if got.Summary != "- you shorten subjects" || got.SuggestedInstructions != "Write short subjects." {
		t.Errorf("unexpected suggestion: %+v", got)
	}
}

func TestSuggestInstructionsAcceptsArraySummary(t *testing.T) {
	got, err := suggestInstructionsWithResponse(t,
		`{"summary":["you shorten subjects","- you add line breaks"],"suggestedInstructions":["Write short subjects."]}`)
	if err != nil {
		t.Fatalf("SuggestInstructions: %v", err)
	}
	if got.Summary != "- you shorten subjects\n- you add line breaks" {
		t.Errorf("summary = %q, want newline-joined bullets", got.Summary)
	}
	if got.SuggestedInstructions != "Write short subjects." {
		t.Errorf("suggestedInstructions = %q", got.SuggestedInstructions)
	}
}

func TestSuggestInstructionsErrorsOnEmptySuggestion(t *testing.T) {
	if _, err := suggestInstructionsWithResponse(t, `{"summary":"- x","suggestedInstructions":""}`); err == nil {
		t.Fatal("expected error for empty suggestedInstructions, got nil")
	}
}

func TestParseUsageReadsRateLimitAndWindowHeaders(t *testing.T) {
	h := http.Header{}
	h.Set("X-Ratelimit-Limit-Requests", "60000")
	h.Set("X-Ratelimit-Remaining-Requests", "59999")
	h.Set("X-Ratelimit-Limit-Tokens", "10000000")
	h.Set("X-Ratelimit-Remaining-Tokens", "9999992")
	h.Set("X-Ratelimit-Renewalperiod-Requests", "10")
	h.Set("X-Ratelimit-Renewalperiod-Tokens", "60")

	u := parseUsage(h)
	if !u.Supported {
		t.Fatal("Supported = false, want true")
	}
	if u.RequestsLimit != 60000 || u.RequestsRemaining != 59999 {
		t.Errorf("requests = %d/%d", u.RequestsRemaining, u.RequestsLimit)
	}
	if u.TokensLimit != 10000000 || u.TokensRemaining != 9999992 {
		t.Errorf("tokens = %d/%d", u.TokensRemaining, u.TokensLimit)
	}
	if u.RequestsWindowSeconds != 10 || u.TokensWindowSeconds != 60 {
		t.Errorf("windows = %ds/%ds, want 10s/60s", u.RequestsWindowSeconds, u.TokensWindowSeconds)
	}
}

func TestParseUsageUnsupportedWhenHeadersAbsent(t *testing.T) {
	if u := parseUsage(http.Header{}); u.Supported {
		t.Errorf("Supported = true for empty headers: %+v", u)
	}
}

func TestChatCompletionReportsTokenUsageToHook(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}],"usage":{"prompt_tokens":12,"completion_tokens":3,"total_tokens":15}}`))
	}))
	defer srv.Close()

	var got TokenUsage
	c := New(srv.URL, "sk-test", "m").OnUsage(func(u TokenUsage) { got = u })
	if _, err := c.Rephrase(context.Background(), []string{"note"}); err != nil {
		t.Fatalf("Rephrase: %v", err)
	}
	if got.PromptTokens != 12 || got.CompletionTokens != 3 || got.TotalTokens != 15 {
		t.Errorf("hook got %+v, want 12/3/15", got)
	}
}
