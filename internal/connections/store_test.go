package connections

import (
	"context"
	"path/filepath"
	"testing"

	"ticketsmith/internal/db"
	"ticketsmith/internal/secrets"
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

// skipIfNoKeyring lets these tests skip gracefully in headless/CI environments
// where no OS keychain backend is available, rather than failing the whole suite.
func skipIfNoKeyring(t *testing.T) {
	t.Helper()
	if err := secrets.Set("ticketsmith-test-probe", "x"); err != nil {
		t.Skipf("OS keychain unavailable in this environment: %v", err)
	}
	secrets.Delete("ticketsmith-test-probe")
}

func TestConnectionCRUDRoundTrip(t *testing.T) {
	skipIfNoKeyring(t)
	s := newTestStore(t)
	ctx := context.Background()

	created, err := s.Create(ctx, "My OpenProject", "openproject", "https://example.com", "tok-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer secrets.Delete(created.KeyringKey)

	if created.Name != "My OpenProject" || created.TrackerKind != "openproject" {
		t.Fatalf("unexpected created connection: %+v", created)
	}

	got, err := s.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != created {
		t.Fatalf("Get mismatch: got %+v, want %+v", got, created)
	}

	stored, err := secrets.Get(created.KeyringKey)
	if err != nil || stored != "tok-1" {
		t.Fatalf("keyring token = %q, %v; want tok-1", stored, err)
	}

	list, err := s.List(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("List: %v, %+v", err, list)
	}

	updated, err := s.Update(ctx, created.ID, "Renamed", "https://example.org", "")
	if err != nil {
		t.Fatalf("Update (no token change): %v", err)
	}
	if updated.Name != "Renamed" || updated.BaseURL != "https://example.org" {
		t.Fatalf("unexpected updated connection: %+v", updated)
	}
	if stored, _ := secrets.Get(updated.KeyringKey); stored != "tok-1" {
		t.Fatalf("token should be unchanged when Update token=\"\", got %q", stored)
	}

	if _, err := s.Update(ctx, created.ID, "Renamed", "https://example.org", "tok-2"); err != nil {
		t.Fatalf("Update (token rotation): %v", err)
	}
	if stored, _ := secrets.Get(updated.KeyringKey); stored != "tok-2" {
		t.Fatalf("token should be rotated to tok-2, got %q", stored)
	}

	if err := s.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(ctx, created.ID); err != ErrNotFound {
		t.Fatalf("Get after delete: err = %v, want ErrNotFound", err)
	}
	if _, err := secrets.Get(created.KeyringKey); err == nil {
		t.Fatalf("expected keyring entry to be deleted")
	}
}
