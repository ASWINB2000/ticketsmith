// Package aiusage persists per-request AI token usage — the "usage" object
// every OpenAI-compatible chat completion response carries in its body. This
// is the app's own audited record of consumption, independent of whatever
// rate-limit headers a provider does or doesn't send.
package aiusage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Record is one chat completion's token spend, as reported by the provider
// in the response body.
type Record struct {
	BaseURL          string
	Model            string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// PeriodTotals aggregates recorded usage over one time window.
type PeriodTotals struct {
	Requests         int `json:"requests"`
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

// Summary is the recorded-usage rollup shown on the Connect screen.
type Summary struct {
	Today     PeriodTotals `json:"today"`
	Last7Days PeriodTotals `json:"last7Days"`
	AllTime   PeriodTotals `json:"allTime"`
}

func (s *Store) Add(ctx context.Context, r Record) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO ai_usage (id, base_url, model, prompt_tokens, completion_tokens, total_tokens)
		VALUES (?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), r.BaseURL, r.Model, r.PromptTokens, r.CompletionTokens, r.TotalTokens)
	if err != nil {
		return fmt.Errorf("aiusage: add: %w", err)
	}
	return nil
}

// Summarize rolls up recorded usage for one provider endpoint (all models —
// so switching models doesn't hide history, and the totals reflect what the
// endpoint's quota actually saw). created_at is stored in UTC, so "today"
// is the UTC day.
func (s *Store) Summarize(ctx context.Context, baseURL string) (Summary, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
		  COUNT(*), COALESCE(SUM(prompt_tokens), 0), COALESCE(SUM(completion_tokens), 0), COALESCE(SUM(total_tokens), 0),
		  COALESCE(SUM(CASE WHEN created_at >= datetime('now', '-7 days') THEN 1 ELSE 0 END), 0),
		  COALESCE(SUM(CASE WHEN created_at >= datetime('now', '-7 days') THEN prompt_tokens ELSE 0 END), 0),
		  COALESCE(SUM(CASE WHEN created_at >= datetime('now', '-7 days') THEN completion_tokens ELSE 0 END), 0),
		  COALESCE(SUM(CASE WHEN created_at >= datetime('now', '-7 days') THEN total_tokens ELSE 0 END), 0),
		  COALESCE(SUM(CASE WHEN date(created_at) = date('now') THEN 1 ELSE 0 END), 0),
		  COALESCE(SUM(CASE WHEN date(created_at) = date('now') THEN prompt_tokens ELSE 0 END), 0),
		  COALESCE(SUM(CASE WHEN date(created_at) = date('now') THEN completion_tokens ELSE 0 END), 0),
		  COALESCE(SUM(CASE WHEN date(created_at) = date('now') THEN total_tokens ELSE 0 END), 0)
		FROM ai_usage WHERE base_url = ?`, baseURL)

	var sum Summary
	err := row.Scan(
		&sum.AllTime.Requests, &sum.AllTime.PromptTokens, &sum.AllTime.CompletionTokens, &sum.AllTime.TotalTokens,
		&sum.Last7Days.Requests, &sum.Last7Days.PromptTokens, &sum.Last7Days.CompletionTokens, &sum.Last7Days.TotalTokens,
		&sum.Today.Requests, &sum.Today.PromptTokens, &sum.Today.CompletionTokens, &sum.Today.TotalTokens,
	)
	if err != nil {
		return Summary{}, fmt.Errorf("aiusage: summarize: %w", err)
	}
	return sum, nil
}
