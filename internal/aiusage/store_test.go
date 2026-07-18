package aiusage

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

func TestAddAndSummarize(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	records := []Record{
		{BaseURL: "https://a.example/v1", Model: "m1", PromptTokens: 100, CompletionTokens: 40, TotalTokens: 140},
		{BaseURL: "https://a.example/v1", Model: "m2", PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		{BaseURL: "https://other.example/v1", Model: "m1", PromptTokens: 999, CompletionTokens: 999, TotalTokens: 1998},
	}
	for _, r := range records {
		if err := s.Add(ctx, r); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}

	sum, err := s.Summarize(ctx, "https://a.example/v1")
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}

	// Rows for other endpoints must not leak in; models on the same endpoint
	// aggregate together.
	want := PeriodTotals{Requests: 2, PromptTokens: 110, CompletionTokens: 45, TotalTokens: 155}
	if sum.AllTime != want {
		t.Errorf("AllTime = %+v, want %+v", sum.AllTime, want)
	}
	// Rows were just inserted, so every window sees them.
	if sum.Today != want {
		t.Errorf("Today = %+v, want %+v", sum.Today, want)
	}
	if sum.Last7Days != want {
		t.Errorf("Last7Days = %+v, want %+v", sum.Last7Days, want)
	}
}

func TestSummarizeEmpty(t *testing.T) {
	s := newTestStore(t)

	sum, err := s.Summarize(context.Background(), "https://a.example/v1")
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if sum.AllTime.Requests != 0 || sum.AllTime.TotalTokens != 0 {
		t.Errorf("expected zero summary, got %+v", sum)
	}
}

func TestSummarizeOldRowsFallOutOfWindows(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	if err := s.Add(ctx, Record{BaseURL: "https://a.example/v1", Model: "m", PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	// Backdate the row past both windows.
	if _, err := s.db.ExecContext(ctx, `UPDATE ai_usage SET created_at = datetime('now', '-10 days')`); err != nil {
		t.Fatalf("backdate: %v", err)
	}

	sum, err := s.Summarize(ctx, "https://a.example/v1")
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if sum.AllTime.Requests != 1 {
		t.Errorf("AllTime.Requests = %d, want 1", sum.AllTime.Requests)
	}
	if sum.Last7Days.Requests != 0 || sum.Today.Requests != 0 {
		t.Errorf("windows should be empty, got Last7Days=%+v Today=%+v", sum.Last7Days, sum.Today)
	}
}
