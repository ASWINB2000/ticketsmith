package prefs

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

func TestGetGenerateDestinationDefaultsToZeroValue(t *testing.T) {
	s := newTestStore(t)

	d, err := s.GetGenerateDestination(context.Background())
	if err != nil {
		t.Fatalf("GetGenerateDestination: %v", err)
	}
	if d.ConnectionID != "" || d.ProjectID != "" {
		t.Errorf("GetGenerateDestination() = %+v, want zero value", d)
	}
}

func TestSaveGenerateDestinationRoundTripsAndOverwrites(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	if err := s.SaveGenerateDestination(ctx, "conn-1", "proj-1"); err != nil {
		t.Fatalf("SaveGenerateDestination: %v", err)
	}
	d, err := s.GetGenerateDestination(ctx)
	if err != nil {
		t.Fatalf("GetGenerateDestination: %v", err)
	}
	if d.ConnectionID != "conn-1" || d.ProjectID != "proj-1" {
		t.Errorf("GetGenerateDestination() = %+v, want conn-1/proj-1", d)
	}

	if err := s.SaveGenerateDestination(ctx, "conn-2", "proj-2"); err != nil {
		t.Fatalf("SaveGenerateDestination (overwrite): %v", err)
	}
	d, err = s.GetGenerateDestination(ctx)
	if err != nil {
		t.Fatalf("GetGenerateDestination: %v", err)
	}
	if d.ConnectionID != "conn-2" || d.ProjectID != "proj-2" {
		t.Errorf("GetGenerateDestination() after overwrite = %+v, want conn-2/proj-2", d)
	}
}
