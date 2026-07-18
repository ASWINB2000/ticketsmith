package ai

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"ticketsmith/internal/db"
	"ticketsmith/internal/secrets"
)

func newTestProfileStore(t *testing.T) *ProfileStore {
	t.Helper()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	return NewProfileStore(sqlDB)
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

func TestProfileCRUDAndActiveSemantics(t *testing.T) {
	skipIfNoKeyring(t)
	s := newTestProfileStore(t)
	ctx := context.Background()

	if _, err := s.GetActive(ctx); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("GetActive on empty store: err = %v, want ErrProfileNotFound", err)
	}

	first, err := s.Create(ctx, "Groq", "https://api.groq.com/openai/v1", "gpt-oss-120b", "sk-first")
	if err != nil {
		t.Fatalf("Create first: %v", err)
	}
	t.Cleanup(func() { secrets.Delete(first.KeyringKey) })
	if !first.Active {
		t.Fatalf("first profile should become active automatically, got %+v", first)
	}
	if !first.HasKey {
		t.Fatalf("profile created with a key should report HasKey, got %+v", first)
	}
	if got, _ := secrets.Get(first.KeyringKey); got != "sk-first" {
		t.Fatalf("keychain secret = %q, want sk-first", got)
	}

	second, err := s.Create(ctx, "Gemini", "https://generativelanguage.googleapis.com/v1beta/openai", "gemini-2.5-flash", "sk-second")
	if err != nil {
		t.Fatalf("Create second: %v", err)
	}
	t.Cleanup(func() { secrets.Delete(second.KeyringKey) })
	if second.Active {
		t.Fatalf("second profile must not steal active status, got %+v", second)
	}

	if err := s.SetActive(ctx, second.ID); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	active, err := s.GetActive(ctx)
	if err != nil || active.ID != second.ID {
		t.Fatalf("GetActive after SetActive: %v, %+v", err, active)
	}
	if p, _ := s.Get(ctx, first.ID); p.Active {
		t.Fatalf("first profile should have been deactivated by SetActive")
	}

	// Update rotates the key in place and keeps the same keyring reference.
	updated, err := s.Update(ctx, second.ID, "Gemini Flash", second.BaseURL, "gemini-3.5-flash", "sk-rotated")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Gemini Flash" || updated.Model != "gemini-3.5-flash" {
		t.Fatalf("unexpected updated profile: %+v", updated)
	}
	if updated.KeyringKey != second.KeyringKey {
		t.Fatalf("update should reuse the keyring key, got %q vs %q", updated.KeyringKey, second.KeyringKey)
	}
	if got, _ := secrets.Get(updated.KeyringKey); got != "sk-rotated" {
		t.Fatalf("keychain secret after rotate = %q, want sk-rotated", got)
	}

	// Deleting the active profile promotes the remaining one.
	if err := s.Delete(ctx, second.ID); err != nil {
		t.Fatalf("Delete active: %v", err)
	}
	if _, err := secrets.Get(second.KeyringKey); err == nil {
		t.Fatalf("deleting a profile should delete its keychain secret")
	}
	active, err = s.GetActive(ctx)
	if err != nil || active.ID != first.ID {
		t.Fatalf("expected the remaining profile to become active, got %v, %+v", err, active)
	}

	if _, err := s.Get(ctx, "does-not-exist"); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("Get missing: err = %v, want ErrProfileNotFound", err)
	}
}

func TestProfileCreateWithoutKey(t *testing.T) {
	s := newTestProfileStore(t)
	ctx := context.Background()

	p, err := s.Create(ctx, "Local Ollama", "http://localhost:11434/v1", "llama3", "")
	if err != nil {
		t.Fatalf("Create without key: %v", err)
	}
	if p.HasKey || p.KeyringKey != "" {
		t.Fatalf("keyless profile should have no keyring entry, got %+v", p)
	}
}

func TestMigrateLegacyConfig(t *testing.T) {
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	s := NewProfileStore(sqlDB)
	ctx := context.Background()

	// No legacy row -> no-op.
	if err := s.MigrateLegacyConfig(ctx); err != nil {
		t.Fatalf("MigrateLegacyConfig (empty): %v", err)
	}
	if list, _ := s.List(ctx); len(list) != 0 {
		t.Fatalf("expected no profiles after no-op migration, got %+v", list)
	}

	// Seed a legacy ai_settings row directly (bypassing keychain).
	if _, err := sqlDB.ExecContext(ctx, `
		INSERT INTO ai_settings (id, base_url, model, keyring_key)
		VALUES ('default', 'https://api.groq.com/openai/v1', 'llama-3.3-70b-versatile', 'legacy-keyring-uuid')`,
	); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	if err := s.MigrateLegacyConfig(ctx); err != nil {
		t.Fatalf("MigrateLegacyConfig: %v", err)
	}
	list, err := s.List(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("expected exactly 1 migrated profile, got %v, %+v", err, list)
	}
	p := list[0]
	if p.Name != "Default" || !p.Active || p.BaseURL != "https://api.groq.com/openai/v1" ||
		p.Model != "llama-3.3-70b-versatile" || p.KeyringKey != "legacy-keyring-uuid" {
		t.Fatalf("unexpected migrated profile: %+v", p)
	}

	// Legacy row is cleared, so deleting the migrated profile and migrating
	// again must NOT resurrect it.
	if _, err := sqlDB.ExecContext(ctx, `DELETE FROM ai_profiles`); err != nil {
		t.Fatalf("clear profiles: %v", err)
	}
	if err := s.MigrateLegacyConfig(ctx); err != nil {
		t.Fatalf("MigrateLegacyConfig (second run): %v", err)
	}
	if list, _ := s.List(ctx); len(list) != 0 {
		t.Fatalf("migration must be one-shot; got resurrected profiles %+v", list)
	}
}
