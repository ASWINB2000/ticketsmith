package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joho/godotenv"

	"ticketsmith/internal/ai"
	"ticketsmith/internal/connections"
	"ticketsmith/internal/db"
	"ticketsmith/internal/logs"
	"ticketsmith/internal/secrets"
	"ticketsmith/internal/templates"
	"ticketsmith/internal/tracker"
)

// newIsolatedApp builds an App wired exactly like startup() does, but against
// a throwaway temp-dir DB instead of the real app-data DB, so this test never
// touches the user's real Ticketsmith data.
func newIsolatedApp(t *testing.T) *App {
	t.Helper()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	return &App{
		ctx:              context.Background(),
		db:               sqlDB,
		connectionsStore: connections.NewStore(sqlDB),
		templatesStore:   templates.NewStore(sqlDB),
		logsStore:        logs.NewStore(sqlDB),
		aiConfigStore:    ai.NewConfigStore(sqlDB),
		trackerCache:     map[string]tracker.Tracker{},
	}
}

// TestFullAppFlow exercises the exact bound methods the UI calls: connection
// CRUD, live (read-only) OpenProject metadata calls against the real
// instance from .env, template CRUD, and the generate flow's graceful
// failure + audit-log behavior when the AI provider is unreachable.
func TestFullAppFlow(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		t.Skip(".env not found — skipping live integration test")
	}
	baseURL := os.Getenv("OPENPROJECT_BASE_URL")
	token := os.Getenv("OPENPROJECT_API_TOKEN")
	if baseURL == "" || token == "" {
		t.Skip("OPENPROJECT_BASE_URL/OPENPROJECT_API_TOKEN not set — skipping live integration test")
	}

	a := newIsolatedApp(t)

	// --- Connections CRUD ---
	conn, err := a.CreateConnection("Integration Test", "openproject", baseURL, token)
	if err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	t.Cleanup(func() { secrets.Delete(conn.KeyringKey) })

	list, err := a.ListConnections()
	if err != nil || len(list) != 1 {
		t.Fatalf("ListConnections: %v, %+v", err, list)
	}

	// --- Live (read-only) tracker metadata ---
	projects, err := a.GetTrackerProjects(conn.ID)
	if err != nil {
		t.Fatalf("GetTrackerProjects (live): %v", err)
	}
	if len(projects) == 0 {
		t.Fatalf("expected at least one real project from the live OpenProject instance")
	}
	t.Logf("live GetTrackerProjects returned %d projects", len(projects))

	types, err := a.GetTrackerTypes(conn.ID)
	if err != nil {
		t.Fatalf("GetTrackerTypes (live): %v", err)
	}
	if len(types) == 0 {
		t.Fatalf("expected at least one real ticket type from the live OpenProject instance")
	}
	t.Logf("live GetTrackerTypes returned %d types", len(types))

	if err := a.TestConnection(conn.ID); err != nil {
		t.Fatalf("TestConnection (live): %v", err)
	}

	// --- Templates CRUD ---
	tmpl, err := a.CreateTemplate(templates.Template{
		Name:            "Integration Test Template",
		TrackerTypeName: types[0].Name,
		FieldsSchema:    []templates.Field{{Name: "steps", Label: "Steps", Type: "textarea"}},
		AIInstructions:  "Be concise.",
	})
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	// --- Generate: AI not configured at all -> config error, no log row ---
	if _, err := a.GenerateTicket(conn.ID, tmpl.ID, "some raw notes"); err == nil {
		t.Fatalf("expected GenerateTicket to fail with no AI provider configured")
	} else if !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("expected a 'not configured' error, got: %v", err)
	}
	if entries, _ := a.ListLogs(logs.Filter{}); len(entries) != 0 {
		t.Fatalf("expected no log rows for a pre-flight config error, got %d", len(entries))
	}

	// --- Generate: AI configured but unreachable -> failure IS logged for audit ---
	if err := a.SaveAISettings("http://127.0.0.1:1", "fake-model", "fake-key"); err != nil {
		t.Fatalf("SaveAISettings: %v", err)
	}
	if _, err := a.GenerateTicket(conn.ID, tmpl.ID, "some raw notes"); err == nil {
		t.Fatalf("expected GenerateTicket to fail against an unreachable AI endpoint")
	}
	entries, err := a.ListLogs(logs.Filter{Action: "generate"})
	if err != nil {
		t.Fatalf("ListLogs: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 generate log row, got %d", len(entries))
	}
	if entries[0].Status != "failure" || entries[0].ErrorMessage == "" {
		t.Fatalf("expected a failure log row with an error message, got %+v", entries[0])
	}
	t.Logf("generate failure correctly logged: %s", entries[0].ErrorMessage)

	// --- Cleanup application-level state this test created ---
	if err := a.DeleteTemplate(tmpl.ID); err != nil {
		t.Fatalf("DeleteTemplate: %v", err)
	}
	if err := a.DeleteConnection(conn.ID); err != nil {
		t.Fatalf("DeleteConnection: %v", err)
	}
}
