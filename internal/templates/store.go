package templates

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("templates: not found")

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func scanTemplate(row interface {
	Scan(dest ...interface{}) error
}) (Template, error) {
	var t Template
	var fieldsJSON string
	if err := row.Scan(&t.ID, &t.Name, &t.TrackerTypeName, &fieldsJSON, &t.AIInstructions, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return Template{}, err
	}
	if err := json.Unmarshal([]byte(fieldsJSON), &t.FieldsSchema); err != nil {
		return Template{}, fmt.Errorf("templates: parse fields_schema: %w", err)
	}
	return t, nil
}

func (s *Store) List(ctx context.Context) ([]Template, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, tracker_type_name, fields_schema, ai_instructions, created_at, updated_at FROM templates ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("templates: list: %w", err)
	}
	defer rows.Close()

	out := []Template{}
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, fmt.Errorf("templates: scan: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) Get(ctx context.Context, id string) (Template, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, tracker_type_name, fields_schema, ai_instructions, created_at, updated_at FROM templates WHERE id = ?`, id)
	t, err := scanTemplate(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Template{}, ErrNotFound
	}
	if err != nil {
		return Template{}, fmt.Errorf("templates: get %q: %w", id, err)
	}
	return t, nil
}

func (s *Store) Create(ctx context.Context, t Template) (Template, error) {
	fieldsJSON, err := json.Marshal(t.FieldsSchema)
	if err != nil {
		return Template{}, fmt.Errorf("templates: marshal fields_schema: %w", err)
	}
	id := uuid.NewString()

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO templates (id, name, tracker_type_name, fields_schema, ai_instructions) VALUES (?, ?, ?, ?, ?)`,
		id, t.Name, t.TrackerTypeName, string(fieldsJSON), t.AIInstructions)
	if err != nil {
		return Template{}, fmt.Errorf("templates: create: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *Store) Update(ctx context.Context, t Template) (Template, error) {
	fieldsJSON, err := json.Marshal(t.FieldsSchema)
	if err != nil {
		return Template{}, fmt.Errorf("templates: marshal fields_schema: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		`UPDATE templates SET name = ?, tracker_type_name = ?, fields_schema = ?, ai_instructions = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		t.Name, t.TrackerTypeName, string(fieldsJSON), t.AIInstructions, t.ID)
	if err != nil {
		return Template{}, fmt.Errorf("templates: update %q: %w", t.ID, err)
	}
	return s.Get(ctx, t.ID)
}

func (s *Store) Delete(ctx context.Context, id string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM templates WHERE id = ?`, id); err != nil {
		return fmt.Errorf("templates: delete %q: %w", id, err)
	}
	return nil
}
