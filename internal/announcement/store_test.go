package announcement

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

func TestGetLastDismissedIDDefaultsToEmpty(t *testing.T) {
	s := newTestStore(t)

	id, err := s.GetLastDismissedID(context.Background())
	if err != nil {
		t.Fatalf("GetLastDismissedID: %v", err)
	}
	if id != "" {
		t.Errorf("GetLastDismissedID() = %q, want empty", id)
	}
}

func TestSaveLastDismissedIDRoundTripsAndOverwrites(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	if err := s.SaveLastDismissedID(ctx, "ann-1"); err != nil {
		t.Fatalf("SaveLastDismissedID: %v", err)
	}
	id, err := s.GetLastDismissedID(ctx)
	if err != nil {
		t.Fatalf("GetLastDismissedID: %v", err)
	}
	if id != "ann-1" {
		t.Errorf("GetLastDismissedID() = %q, want ann-1", id)
	}

	if err := s.SaveLastDismissedID(ctx, "ann-2"); err != nil {
		t.Fatalf("SaveLastDismissedID (overwrite): %v", err)
	}
	id, err = s.GetLastDismissedID(ctx)
	if err != nil {
		t.Fatalf("GetLastDismissedID: %v", err)
	}
	if id != "ann-2" {
		t.Errorf("GetLastDismissedID() after overwrite = %q, want ann-2", id)
	}
}
