package logs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("logs: not found")

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func scanLogEntry(row interface {
	Scan(dest ...interface{}) error
}) (LogEntry, error) {
	var e LogEntry
	var connectionID, templateID, rawInput, generatedContent, finalContent sql.NullString
	var resultTicketID, resultTicketURL, errorMessage sql.NullString
	err := row.Scan(&e.ID, &e.Timestamp, &e.Action, &connectionID, &templateID,
		&rawInput, &generatedContent, &finalContent,
		&resultTicketID, &resultTicketURL, &e.Status, &errorMessage)
	if err != nil {
		return LogEntry{}, err
	}
	e.ConnectionID = connectionID.String
	e.TemplateID = templateID.String
	e.RawInput = rawInput.String
	e.GeneratedContent = generatedContent.String
	e.FinalContent = finalContent.String
	e.ResultTicketID = resultTicketID.String
	e.ResultTicketURL = resultTicketURL.String
	e.ErrorMessage = errorMessage.String
	return e, nil
}

const selectColumns = `id, timestamp, action, connection_id, template_id, raw_input, generated_content, final_content, result_ticket_id, result_ticket_url, status, error_message`

func (s *Store) Create(ctx context.Context, e LogEntry) (LogEntry, error) {
	id := uuid.NewString()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO logs (id, action, connection_id, template_id, raw_input, generated_content, final_content, result_ticket_id, result_ticket_url, status, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, e.Action, e.ConnectionID, e.TemplateID, e.RawInput, e.GeneratedContent, e.FinalContent,
		e.ResultTicketID, e.ResultTicketURL, e.Status, e.ErrorMessage)
	if err != nil {
		return LogEntry{}, fmt.Errorf("logs: create: %w", err)
	}
	return s.Get(ctx, id)
}

// Update mutates the same log row in place (the generate->create flow shares
// one row, per the schema's generated_content/final_content pairing).
func (s *Store) Update(ctx context.Context, id, action, finalContent, resultTicketID, resultTicketURL, status, errorMessage string) (LogEntry, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE logs SET action = ?, final_content = ?, result_ticket_id = ?, result_ticket_url = ?, status = ?, error_message = ?
		WHERE id = ?`,
		action, finalContent, resultTicketID, resultTicketURL, status, errorMessage, id)
	if err != nil {
		return LogEntry{}, fmt.Errorf("logs: update %q: %w", id, err)
	}
	return s.Get(ctx, id)
}

func (s *Store) Get(ctx context.Context, id string) (LogEntry, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+selectColumns+` FROM logs WHERE id = ?`, id)
	e, err := scanLogEntry(row)
	if errors.Is(err, sql.ErrNoRows) {
		return LogEntry{}, ErrNotFound
	}
	if err != nil {
		return LogEntry{}, fmt.Errorf("logs: get %q: %w", id, err)
	}
	return e, nil
}

func (s *Store) List(ctx context.Context, f Filter) ([]LogEntry, error) {
	query := `SELECT ` + selectColumns + ` FROM logs`
	var conditions []string
	var args []interface{}

	if f.Action != "" {
		conditions = append(conditions, "action = ?")
		args = append(args, f.Action)
	}
	if f.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, f.Status)
	}
	if f.ConnectionID != "" {
		conditions = append(conditions, "connection_id = ?")
		args = append(args, f.ConnectionID)
	}
	if f.TemplateID != "" {
		conditions = append(conditions, "template_id = ?")
		args = append(args, f.TemplateID)
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	// timestamp has only second resolution; rowid (insertion order) breaks ties deterministically.
	query += " ORDER BY timestamp DESC, rowid DESC"

	if f.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, f.Limit)
		if f.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, f.Offset)
		}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("logs: list: %w", err)
	}
	defer rows.Close()

	out := []LogEntry{}
	for rows.Next() {
		e, err := scanLogEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("logs: scan: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
