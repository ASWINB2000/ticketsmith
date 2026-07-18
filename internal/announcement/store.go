package announcement

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Store persists which announcement (by ID) the user has already dismissed.
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// GetLastDismissedID returns the ID of the last announcement the user
// dismissed, or "" if none has been dismissed yet.
func (s *Store) GetLastDismissedID(ctx context.Context) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx,
		`SELECT last_dismissed_id FROM announcement_prefs WHERE id = 'default'`,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("announcement: get last dismissed id: %w", err)
	}
	return id, nil
}

// SaveLastDismissedID upserts the dismissed announcement ID.
func (s *Store) SaveLastDismissedID(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO announcement_prefs (id, last_dismissed_id, updated_at)
		VALUES ('default', ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			last_dismissed_id = excluded.last_dismissed_id,
			updated_at = CURRENT_TIMESTAMP
	`, id)
	if err != nil {
		return fmt.Errorf("announcement: save last dismissed id: %w", err)
	}
	return nil
}
