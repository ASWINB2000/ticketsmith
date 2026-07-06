package logs

import (
	"context"
	"path/filepath"
	"testing"

	"ticketsmith/internal/db"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	return NewStore(sqlDB)
}

func TestLogEntryCreateUpdateRoundTrip(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	created, err := s.Create(ctx, LogEntry{
		Action:           "generate",
		TemplateID:       "tmpl-1",
		RawInput:         "raw notes",
		GeneratedContent: `{"subject":"S"}`,
		Status:           "success",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" || created.Action != "generate" {
		t.Fatalf("unexpected created entry: %+v", created)
	}

	updated, err := s.Update(ctx, created.ID, "create", `{"subject":"S","description":"D"}`, "42", "https://example.com/wp/42", "success", "")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.ID != created.ID {
		t.Fatalf("Update should mutate the same row in place; got a different ID %q vs %q", updated.ID, created.ID)
	}
	if updated.Action != "create" || updated.ResultTicketID != "42" || updated.RawInput != "raw notes" {
		t.Fatalf("unexpected updated entry: %+v", updated)
	}

	got, err := s.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ResultTicketURL != "https://example.com/wp/42" {
		t.Fatalf("unexpected fetched entry: %+v", got)
	}

	if _, err := s.Get(ctx, "does-not-exist"); err != ErrNotFound {
		t.Fatalf("Get missing: err = %v, want ErrNotFound", err)
	}
}

func TestLogEntryListFiltersAndOrdersReverseChronologically(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	first, err := s.Create(ctx, LogEntry{Action: "generate", ConnectionID: "conn-a", Status: "success"})
	if err != nil {
		t.Fatalf("Create first: %v", err)
	}
	second, err := s.Create(ctx, LogEntry{Action: "create", ConnectionID: "conn-b", Status: "failure", ErrorMessage: "boom"})
	if err != nil {
		t.Fatalf("Create second: %v", err)
	}

	all, err := s.List(ctx, Filter{})
	if err != nil || len(all) != 2 {
		t.Fatalf("List all: %v, %+v", err, all)
	}
	if all[0].ID != second.ID || all[1].ID != first.ID {
		t.Fatalf("expected reverse-chronological order, got %+v", all)
	}

	byStatus, err := s.List(ctx, Filter{Status: "failure"})
	if err != nil || len(byStatus) != 1 || byStatus[0].ID != second.ID {
		t.Fatalf("List by status=failure: %v, %+v", err, byStatus)
	}

	byConn, err := s.List(ctx, Filter{ConnectionID: "conn-a"})
	if err != nil || len(byConn) != 1 || byConn[0].ID != first.ID {
		t.Fatalf("List by connection_id: %v, %+v", err, byConn)
	}

	limited, err := s.List(ctx, Filter{Limit: 1})
	if err != nil || len(limited) != 1 {
		t.Fatalf("List with limit: %v, %+v", err, limited)
	}
}
