package openaicompat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
