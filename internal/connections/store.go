package connections

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"ticketsmith/internal/secrets"
)

var ErrNotFound = errors.New("connections: not found")

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) List(ctx context.Context) ([]Connection, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, tracker_kind, base_url, keyring_key, created_at FROM connections ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("connections: list: %w", err)
	}
	defer rows.Close()

	out := []Connection{}
	for rows.Next() {
		var c Connection
		if err := rows.Scan(&c.ID, &c.Name, &c.TrackerKind, &c.BaseURL, &c.KeyringKey, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("connections: scan: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) Get(ctx context.Context, id string) (Connection, error) {
	var c Connection
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, tracker_kind, base_url, keyring_key, created_at FROM connections WHERE id = ?`, id,
	).Scan(&c.ID, &c.Name, &c.TrackerKind, &c.BaseURL, &c.KeyringKey, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Connection{}, ErrNotFound
	}
	if err != nil {
		return Connection{}, fmt.Errorf("connections: get %q: %w", id, err)
	}
	return c, nil
}

func (s *Store) Create(ctx context.Context, name, trackerKind, baseURL, token string) (Connection, error) {
	id := uuid.NewString()
	keyringKey := uuid.NewString()
	if err := secrets.Set(keyringKey, token); err != nil {
		return Connection{}, fmt.Errorf("connections: store token: %w", err)
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO connections (id, name, tracker_kind, base_url, keyring_key) VALUES (?, ?, ?, ?, ?)`,
		id, name, trackerKind, baseURL, keyringKey)
	if err != nil {
		return Connection{}, fmt.Errorf("connections: create: %w", err)
	}
	return s.Get(ctx, id)
}

// Update edits a connection's name/baseURL, and rotates its token if a
// non-empty token is supplied. An empty token leaves the existing secret untouched.
func (s *Store) Update(ctx context.Context, id, name, baseURL, token string) (Connection, error) {
	existing, err := s.Get(ctx, id)
	if err != nil {
		return Connection{}, err
	}

	if token != "" {
		if err := secrets.Set(existing.KeyringKey, token); err != nil {
			return Connection{}, fmt.Errorf("connections: rotate token: %w", err)
		}
	}

	_, err = s.db.ExecContext(ctx,
		`UPDATE connections SET name = ?, base_url = ? WHERE id = ?`, name, baseURL, id)
	if err != nil {
		return Connection{}, fmt.Errorf("connections: update %q: %w", id, err)
	}
	return s.Get(ctx, id)
}

func (s *Store) Delete(ctx context.Context, id string) error {
	existing, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := secrets.Delete(existing.KeyringKey); err != nil {
		return fmt.Errorf("connections: delete token: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM connections WHERE id = ?`, id); err != nil {
		return fmt.Errorf("connections: delete %q: %w", id, err)
	}
	return nil
}
