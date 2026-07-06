package templates

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

func TestTemplateCRUDRoundTrip(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	tmpl := Template{
		Name:            "Bug Report",
		TrackerTypeName: "Bug",
		FieldsSchema: []Field{
			{Name: "steps", Label: "Steps to reproduce", Type: "textarea"},
		},
		AIInstructions: "Be concise.",
	}

	created, err := s.Create(ctx, tmpl)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" || len(created.FieldsSchema) != 1 || created.FieldsSchema[0].Name != "steps" {
		t.Fatalf("unexpected created template: %+v", created)
	}

	got, err := s.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "Bug Report" {
		t.Fatalf("unexpected fetched template: %+v", got)
	}

	list, err := s.List(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("List: %v, %+v", err, list)
	}

	created.Name = "Bug Report v2"
	created.FieldsSchema = append(created.FieldsSchema, Field{Name: "severity", Label: "Severity", Type: "text"})
	updated, err := s.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Bug Report v2" || len(updated.FieldsSchema) != 2 {
		t.Fatalf("unexpected updated template: %+v", updated)
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
