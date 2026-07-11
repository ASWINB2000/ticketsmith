package notes

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

func TestNoteCRUDRoundTrip(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	created, err := s.Create(ctx, "My heading", "jot something down")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" || created.Title != "My heading" || created.Content != "jot something down" || created.Status != statusActive {
		t.Fatalf("unexpected created note: %+v", created)
	}

	got, err := s.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Content != "jot something down" {
		t.Fatalf("unexpected fetched note: %+v", got)
	}

	list, err := s.List(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("List: %v, %+v", err, list)
	}

	updated, err := s.Update(ctx, created.ID, "Revised heading", "revised content")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Title != "Revised heading" || updated.Content != "revised content" {
		t.Fatalf("unexpected updated note: %+v", updated)
	}
	if updated.UpdatedAt.IsZero() {
		t.Fatalf("expected updated_at to be set")
	}

	if err := s.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(ctx, created.ID); err != ErrNotFound {
		t.Fatalf("Get after delete: err = %v, want ErrNotFound", err)
	}
}

func TestCreateMergedMarksSourcesAndPreservesLineage(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	a, err := s.Create(ctx, "", "first note")
	if err != nil {
		t.Fatalf("Create a: %v", err)
	}
	b, err := s.Create(ctx, "", "second note")
	if err != nil {
		t.Fatalf("Create b: %v", err)
	}
	c, err := s.Create(ctx, "", "third note")
	if err != nil {
		t.Fatalf("Create c: %v", err)
	}

	sourceIDs := []string{a.ID, b.ID, c.ID}
	merged, err := s.CreateMerged(ctx, sourceIDs, "Merged heading", "combined draft")
	if err != nil {
		t.Fatalf("CreateMerged: %v", err)
	}
	if merged.Title != "Merged heading" || merged.Content != "combined draft" || merged.Status != statusActive {
		t.Fatalf("unexpected merged note: %+v", merged)
	}
	if len(merged.MergedFromIDs) != 3 {
		t.Fatalf("MergedFromIDs = %+v, want 3 entries", merged.MergedFromIDs)
	}
	for i, id := range sourceIDs {
		if merged.MergedFromIDs[i] != id {
			t.Fatalf("MergedFromIDs[%d] = %q, want %q", i, merged.MergedFromIDs[i], id)
		}
	}

	list, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].ID != merged.ID {
		t.Fatalf("List after merge = %+v, want only the merged note", list)
	}

	for _, id := range sourceIDs {
		src, err := s.Get(ctx, id)
		if err != nil {
			t.Fatalf("Get source %q: %v", id, err)
		}
		if src.Status != statusMerged {
			t.Fatalf("source %q status = %q, want %q (never deleted)", id, src.Status, statusMerged)
		}
	}
}
