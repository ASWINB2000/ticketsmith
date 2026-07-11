package notes

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("notes: not found")

const (
	statusActive = "active"
	statusMerged = "merged"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

const selectColumns = `id, title, content, status, merged_from_ids, created_at, updated_at`

func scanNote(row interface {
	Scan(dest ...interface{}) error
}) (Note, error) {
	var n Note
	var mergedFromJSON sql.NullString
	if err := row.Scan(&n.ID, &n.Title, &n.Content, &n.Status, &mergedFromJSON, &n.CreatedAt, &n.UpdatedAt); err != nil {
		return Note{}, err
	}
	if mergedFromJSON.Valid && mergedFromJSON.String != "" {
		if err := json.Unmarshal([]byte(mergedFromJSON.String), &n.MergedFromIDs); err != nil {
			return Note{}, fmt.Errorf("notes: parse merged_from_ids: %w", err)
		}
	}
	return n, nil
}

// List returns active notes only, most-recently-updated first — mirrors
// logs' `timestamp DESC, rowid DESC` tie-break (updated_at has only second
// resolution, and a bulk merge can touch several rows within one second).
func (s *Store) List(ctx context.Context) ([]Note, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+selectColumns+` FROM notes WHERE status = ? ORDER BY updated_at DESC, rowid DESC`, statusActive)
	if err != nil {
		return nil, fmt.Errorf("notes: list: %w", err)
	}
	defer rows.Close()

	out := []Note{}
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, fmt.Errorf("notes: scan: %w", err)
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *Store) Get(ctx context.Context, id string) (Note, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+selectColumns+` FROM notes WHERE id = ?`, id)
	n, err := scanNote(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Note{}, ErrNotFound
	}
	if err != nil {
		return Note{}, fmt.Errorf("notes: get %q: %w", id, err)
	}
	return n, nil
}

func (s *Store) Create(ctx context.Context, title, content string) (Note, error) {
	id := uuid.NewString()
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO notes (id, title, content, status) VALUES (?, ?, ?, ?)`, id, title, content, statusActive); err != nil {
		return Note{}, fmt.Errorf("notes: create: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *Store) Update(ctx context.Context, id, title, content string) (Note, error) {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE notes SET title = ?, content = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, title, content, id); err != nil {
		return Note{}, fmt.Errorf("notes: update %q: %w", id, err)
	}
	return s.Get(ctx, id)
}

func (s *Store) Delete(ctx context.Context, id string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM notes WHERE id = ?`, id); err != nil {
		return fmt.Errorf("notes: delete %q: %w", id, err)
	}
	return nil
}

// CreateMerged persists a merge: a new active note carrying merged_from_ids
// as a JSON array of the source IDs, plus marking every source note
// status='merged' (never deleted). This is wrapped in a transaction —
// unlike every other single-row-per-call method in this store, a partial
// failure here (merged note created but a source not marked, or vice
// versa) would corrupt lineage. db.Open sets SetMaxOpenConns(1), so a
// transaction on that single connection is safe for this single-user app.
func (s *Store) CreateMerged(ctx context.Context, sourceIDs []string, title, content string) (Note, error) {
	mergedJSON, err := json.Marshal(sourceIDs)
	if err != nil {
		return Note{}, fmt.Errorf("notes: marshal merged_from_ids: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Note{}, fmt.Errorf("notes: begin merge: %w", err)
	}
	defer tx.Rollback()

	id := uuid.NewString()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO notes (id, title, content, status, merged_from_ids) VALUES (?, ?, ?, ?, ?)`,
		id, title, content, statusActive, string(mergedJSON)); err != nil {
		return Note{}, fmt.Errorf("notes: create merged note: %w", err)
	}
	for _, sourceID := range sourceIDs {
		if _, err := tx.ExecContext(ctx,
			`UPDATE notes SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, statusMerged, sourceID); err != nil {
			return Note{}, fmt.Errorf("notes: mark source %q merged: %w", sourceID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Note{}, fmt.Errorf("notes: commit merge: %w", err)
	}
	return s.Get(ctx, id)
}
