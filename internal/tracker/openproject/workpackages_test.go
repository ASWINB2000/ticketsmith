package openproject

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ticketsmith/internal/tracker"
)

func TestCreateTicketBuildsExpectedBody(t *testing.T) {
	var captured createWorkPackageRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v3/projects/1/work_packages/form" {
			w.Write([]byte(`{"_embedded":{"schema":{}}}`))
			return
		}
		if r.URL.Path != "/api/v3/work_packages" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":42,"_links":{"self":{"href":"/api/v3/work_packages/42"}}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	ticket, err := c.CreateTicket(context.Background(), "1", "2", tracker.TicketInput{
		Subject:     "Something broke",
		Description: "It broke badly.",
		Fields:      []tracker.FieldValue{{Label: "Steps to reproduce", Value: "Click the button"}},
		ParentID:    "10",
		AssigneeID:  "7",
	})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}

	if ticket.ID != "42" || ticket.URL != srv.URL+"/work_packages/42" {
		t.Errorf("unexpected ticket: %+v", ticket)
	}
	if captured.Subject != "Something broke" {
		t.Errorf("Subject = %q", captured.Subject)
	}
	if captured.Description.Format != "markdown" {
		t.Errorf("Description.Format = %q, want markdown", captured.Description.Format)
	}
	if !strings.Contains(captured.Description.Raw, "It broke badly.") ||
		!strings.Contains(captured.Description.Raw, "#### Steps to reproduce\nClick the button") {
		t.Errorf("Description.Raw = %q", captured.Description.Raw)
	}
	if captured.Links.Project == nil || captured.Links.Project.Href != "/api/v3/projects/1" {
		t.Errorf("Links.Project = %+v", captured.Links.Project)
	}
	if captured.Links.Type == nil || captured.Links.Type.Href != "/api/v3/types/2" {
		t.Errorf("Links.Type = %+v", captured.Links.Type)
	}
	if captured.Links.Parent == nil || captured.Links.Parent.Href != "/api/v3/work_packages/10" {
		t.Errorf("Links.Parent = %+v", captured.Links.Parent)
	}
	if captured.Links.Assignee == nil || captured.Links.Assignee.Href != "/api/v3/users/7" {
		t.Errorf("Links.Assignee = %+v", captured.Links.Assignee)
	}
}

func TestCreateTicketPreservesFieldOrder(t *testing.T) {
	var captured createWorkPackageRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v3/projects/1/work_packages/form" {
			w.Write([]byte(`{"_embedded":{"schema":{}}}`))
			return
		}
		json.NewDecoder(r.Body).Decode(&captured)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1,"_links":{"self":{"href":"/api/v3/work_packages/1"}}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	_, err := c.CreateTicket(context.Background(), "1", "2", tracker.TicketInput{
		Subject:     "Ordered",
		Description: "Summary.",
		Fields: []tracker.FieldValue{
			{Label: "Test Data", Value: "url + creds"},
			{Label: "Steps to Reproduce", Value: "1. Do a thing"},
			{Label: "Expected Result", Value: "It works"},
			{Label: "Actual Result", Value: "It doesn't"},
		},
	})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}

	raw := captured.Description.Raw
	testDataIdx := strings.Index(raw, "#### Test Data")
	stepsIdx := strings.Index(raw, "#### Steps to Reproduce")
	expectedIdx := strings.Index(raw, "#### Expected Result")
	actualIdx := strings.Index(raw, "#### Actual Result")
	if testDataIdx == -1 || stepsIdx == -1 || expectedIdx == -1 || actualIdx == -1 {
		t.Fatalf("missing expected headings in Description.Raw = %q", raw)
	}
	if !(testDataIdx < stepsIdx && stepsIdx < expectedIdx && expectedIdx < actualIdx) {
		t.Errorf("fields not in declared order, Description.Raw = %q", raw)
	}
}

func TestCreateTicketIncludesPriorityAndDates(t *testing.T) {
	var captured createWorkPackageRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&captured)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1,"_links":{"self":{"href":"/api/v3/work_packages/1"}}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	_, err := c.CreateTicket(context.Background(), "1", "2", tracker.TicketInput{
		Subject:     "With priority and dates",
		Description: "Summary.",
		PriorityID:  "8",
		StartDate:   "2026-07-15",
		DueDate:     "2026-08-01",
	})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	if captured.Links.Priority == nil || captured.Links.Priority.Href != "/api/v3/priorities/8" {
		t.Errorf("Links.Priority = %+v", captured.Links.Priority)
	}
	if captured.StartDate != "2026-07-15" {
		t.Errorf("StartDate = %q, want 2026-07-15", captured.StartDate)
	}
	if captured.DueDate != "2026-08-01" {
		t.Errorf("DueDate = %q, want 2026-08-01", captured.DueDate)
	}
}

func TestCreateTicketOmitsOptionalLinksWhenEmpty(t *testing.T) {
	var captured createWorkPackageRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&captured)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1,"_links":{"self":{"href":"/api/v3/work_packages/1"}}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	_, err := c.CreateTicket(context.Background(), "1", "2", tracker.TicketInput{
		Subject:     "Plain ticket",
		Description: "No extras.",
	})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	if captured.Links.Parent != nil {
		t.Errorf("Links.Parent = %+v, want nil", captured.Links.Parent)
	}
	if captured.Links.Assignee != nil {
		t.Errorf("Links.Assignee = %+v, want nil", captured.Links.Assignee)
	}
	if captured.Description.Raw != "No extras." {
		t.Errorf("Description.Raw = %q, want unchanged description", captured.Description.Raw)
	}
	if captured.Links.Priority != nil {
		t.Errorf("Links.Priority = %+v, want nil", captured.Links.Priority)
	}
	if captured.DueDate != "" {
		t.Errorf("DueDate = %q, want empty", captured.DueDate)
	}
}

func TestCreateTicketMapsMatchingFieldToStringCustomField(t *testing.T) {
	var captured map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v3/projects/1/work_packages/form" {
			w.Write([]byte(`{"_embedded":{"schema":{"customField5":{"type":"String","name":"Steps to reproduce"}}}}`))
			return
		}
		json.NewDecoder(r.Body).Decode(&captured)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1,"_links":{"self":{"href":"/api/v3/work_packages/1"}}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	_, err := c.CreateTicket(context.Background(), "1", "2", tracker.TicketInput{
		Subject:     "Something broke",
		Description: "It broke badly.",
		Fields:      []tracker.FieldValue{{Label: "Steps to reproduce", Value: "Click the button"}},
	})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}

	if got, _ := captured["customField5"].(string); got != "Click the button" {
		t.Errorf("customField5 = %v, want %q", captured["customField5"], "Click the button")
	}
	desc, _ := captured["description"].(map[string]interface{})
	if raw, _ := desc["raw"].(string); strings.Contains(raw, "Steps to reproduce") {
		t.Errorf("Description.Raw = %q, matched field should not also be appended to description", raw)
	}
}

func TestCreateTicketMapsMatchingFieldToFormattableCustomField(t *testing.T) {
	var captured map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v3/projects/1/work_packages/form" {
			w.Write([]byte(`{"_embedded":{"schema":{"customField7":{"type":"Formattable","name":"Acceptance Criteria"}}}}`))
			return
		}
		json.NewDecoder(r.Body).Decode(&captured)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1,"_links":{"self":{"href":"/api/v3/work_packages/1"}}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	// Matching is case-insensitive.
	_, err := c.CreateTicket(context.Background(), "1", "2", tracker.TicketInput{
		Subject:     "A story",
		Description: "Summary.",
		Fields:      []tracker.FieldValue{{Label: "acceptance criteria", Value: "Given/When/Then"}},
	})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}

	cf, _ := captured["customField7"].(map[string]interface{})
	if raw, _ := cf["raw"].(string); raw != "Given/When/Then" {
		t.Errorf("customField7.raw = %v, want %q", cf["raw"], "Given/When/Then")
	}
}

func TestCreateTicketFallsBackToDescriptionWhenNoCustomFieldMatches(t *testing.T) {
	var captured map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v3/projects/1/work_packages/form" {
			w.Write([]byte(`{"_embedded":{"schema":{"customField5":{"type":"String","name":"Unrelated Field"}}}}`))
			return
		}
		json.NewDecoder(r.Body).Decode(&captured)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1,"_links":{"self":{"href":"/api/v3/work_packages/1"}}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	_, err := c.CreateTicket(context.Background(), "1", "2", tracker.TicketInput{
		Subject:     "Something broke",
		Description: "It broke badly.",
		Fields:      []tracker.FieldValue{{Label: "Steps to reproduce", Value: "Click the button"}},
	})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}

	if _, ok := captured["customField5"]; ok {
		t.Errorf("customField5 should not be set, got %v", captured["customField5"])
	}
	desc, _ := captured["description"].(map[string]interface{})
	if raw, _ := desc["raw"].(string); !strings.Contains(raw, "#### Steps to reproduce\nClick the button") {
		t.Errorf("Description.Raw = %q, want unmatched field appended", raw)
	}
}

func TestCreateTicketFallsBackToDescriptionWhenCustomFieldDiscoveryFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v3/projects/1/work_packages/form" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message":"boom"}`))
			return
		}
		var captured map[string]interface{}
		json.NewDecoder(r.Body).Decode(&captured)
		desc, _ := captured["description"].(map[string]interface{})
		if raw, _ := desc["raw"].(string); !strings.Contains(raw, "#### Steps to reproduce\nClick the button") {
			t.Errorf("Description.Raw = %q, want field appended despite discovery failure", raw)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1,"_links":{"self":{"href":"/api/v3/work_packages/1"}}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	_, err := c.CreateTicket(context.Background(), "1", "2", tracker.TicketInput{
		Subject:     "Something broke",
		Description: "It broke badly.",
		Fields:      []tracker.FieldValue{{Label: "Steps to reproduce", Value: "Click the button"}},
	})
	if err != nil {
		t.Fatalf("CreateTicket should still succeed when custom field discovery fails: %v", err)
	}
}
