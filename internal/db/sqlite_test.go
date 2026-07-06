package db

import (
	"path/filepath"
	"testing"
)

func TestOpenIsIdempotentAndCreatesSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	db1, err := Open(path)
	if err != nil {
		t.Fatalf("first Open failed: %v", err)
	}
	defer db1.Close()

	db2, err := Open(path)
	if err != nil {
		t.Fatalf("second Open failed: %v", err)
	}
	defer db2.Close()

	wantTables := []string{"connections", "templates", "logs", "ai_settings"}
	for _, table := range wantTables {
		var name string
		err := db2.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}
