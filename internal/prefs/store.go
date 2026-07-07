// Package prefs persists small pieces of user preference that aren't part of
// any domain entity — currently just the last-used Generate screen
// destination (connection + project), so it survives app restarts.
package prefs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// GenerateDestination is the last connection/project the user picked on the
// Generate screen's "Configure destination" sheet.
type GenerateDestination struct {
	ConnectionID string `json:"connectionId"`
	ProjectID    string `json:"projectId"`
}

// Store persists the single-row "default" preferences.
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// GetGenerateDestination returns the saved destination, or a zero-value one
// if none has been saved yet.
func (s *Store) GetGenerateDestination(ctx context.Context) (GenerateDestination, error) {
	var d GenerateDestination
	err := s.db.QueryRowContext(ctx,
		`SELECT connection_id, project_id FROM generate_prefs WHERE id = 'default'`,
	).Scan(&d.ConnectionID, &d.ProjectID)
	if errors.Is(err, sql.ErrNoRows) {
		return GenerateDestination{}, nil
	}
	if err != nil {
		return GenerateDestination{}, fmt.Errorf("prefs: get generate destination: %w", err)
	}
	return d, nil
}

// SaveGenerateDestination upserts the saved destination.
func (s *Store) SaveGenerateDestination(ctx context.Context, connectionID, projectID string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO generate_prefs (id, connection_id, project_id, updated_at)
		VALUES ('default', ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			connection_id = excluded.connection_id,
			project_id = excluded.project_id,
			updated_at = CURRENT_TIMESTAMP
	`, connectionID, projectID)
	if err != nil {
		return fmt.Errorf("prefs: save generate destination: %w", err)
	}
	return nil
}
