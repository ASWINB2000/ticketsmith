package ai

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"ticketsmith/internal/secrets"
)

var ErrProfileNotFound = errors.New("ai: profile not found")

// Profile is one saved AI provider configuration (e.g. "Groq", "Gemini",
// "Local Ollama"). Exactly one profile is active at a time — that's the one
// aiProvider() builds a client from. Like tracker connections, the API key
// itself lives in the OS keychain; only KeyringKey is stored here.
type Profile struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	BaseURL    string `json:"baseUrl"`
	Model      string `json:"model"`
	KeyringKey string `json:"-"`
	Active     bool   `json:"active"`
	HasKey     bool   `json:"hasKey"`
}

// ProfileStore persists AI provider profiles.
type ProfileStore struct {
	db *sql.DB
}

func NewProfileStore(db *sql.DB) *ProfileStore {
	return &ProfileStore{db: db}
}

const profileColumns = `id, name, base_url, model, keyring_key, is_active`

func scanProfile(row interface {
	Scan(dest ...interface{}) error
}) (Profile, error) {
	var p Profile
	var active int
	if err := row.Scan(&p.ID, &p.Name, &p.BaseURL, &p.Model, &p.KeyringKey, &active); err != nil {
		return Profile{}, err
	}
	p.Active = active != 0
	p.HasKey = p.KeyringKey != ""
	return p, nil
}

func (s *ProfileStore) List(ctx context.Context) ([]Profile, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+profileColumns+` FROM ai_profiles ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("ai: list profiles: %w", err)
	}
	defer rows.Close()

	out := []Profile{}
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, fmt.Errorf("ai: scan profile: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *ProfileStore) Get(ctx context.Context, id string) (Profile, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+profileColumns+` FROM ai_profiles WHERE id = ?`, id)
	p, err := scanProfile(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Profile{}, ErrProfileNotFound
	}
	if err != nil {
		return Profile{}, fmt.Errorf("ai: get profile %q: %w", id, err)
	}
	return p, nil
}

// GetActive returns the currently-active profile, or ErrProfileNotFound if
// no profile is active (including when no profiles exist at all).
func (s *ProfileStore) GetActive(ctx context.Context) (Profile, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+profileColumns+` FROM ai_profiles WHERE is_active = 1`)
	p, err := scanProfile(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Profile{}, ErrProfileNotFound
	}
	if err != nil {
		return Profile{}, fmt.Errorf("ai: get active profile: %w", err)
	}
	return p, nil
}

// Create saves a new profile. If no profile is currently active (first-ever
// profile, or every previous one was deleted), the new one becomes active,
// so a fresh setup works without a separate "activate" step. apiKey may be
// empty (e.g. a local Ollama endpoint that needs none).
func (s *ProfileStore) Create(ctx context.Context, name, baseURL, model, apiKey string) (Profile, error) {
	id := uuid.NewString()
	keyringKey := ""
	if apiKey != "" {
		keyringKey = uuid.NewString()
		if err := secrets.Set(keyringKey, apiKey); err != nil {
			return Profile{}, fmt.Errorf("ai: store api key: %w", err)
		}
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO ai_profiles (id, name, base_url, model, keyring_key, is_active)
		VALUES (?, ?, ?, ?, ?, NOT EXISTS (SELECT 1 FROM ai_profiles WHERE is_active = 1))`,
		id, name, baseURL, model, keyringKey)
	if err != nil {
		return Profile{}, fmt.Errorf("ai: create profile: %w", err)
	}
	return s.Get(ctx, id)
}

// Update edits a profile's name/baseURL/model, and rotates its API key if a
// non-empty key is supplied. An empty apiKey leaves the existing secret
// untouched.
func (s *ProfileStore) Update(ctx context.Context, id, name, baseURL, model, apiKey string) (Profile, error) {
	existing, err := s.Get(ctx, id)
	if err != nil {
		return Profile{}, err
	}

	keyringKey := existing.KeyringKey
	if apiKey != "" {
		if keyringKey == "" {
			keyringKey = uuid.NewString()
		}
		if err := secrets.Set(keyringKey, apiKey); err != nil {
			return Profile{}, fmt.Errorf("ai: rotate api key: %w", err)
		}
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE ai_profiles SET name = ?, base_url = ?, model = ?, keyring_key = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, name, baseURL, model, keyringKey, id)
	if err != nil {
		return Profile{}, fmt.Errorf("ai: update profile %q: %w", id, err)
	}
	return s.Get(ctx, id)
}

// Delete removes a profile and its keychain secret. If the deleted profile
// was active, the oldest remaining profile (if any) becomes active, so the
// app never silently loses its AI provider while alternatives exist.
func (s *ProfileStore) Delete(ctx context.Context, id string) error {
	existing, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if existing.KeyringKey != "" {
		if err := secrets.Delete(existing.KeyringKey); err != nil {
			return fmt.Errorf("ai: delete api key: %w", err)
		}
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM ai_profiles WHERE id = ?`, id); err != nil {
		return fmt.Errorf("ai: delete profile %q: %w", id, err)
	}

	if existing.Active {
		_, err = s.db.ExecContext(ctx, `
			UPDATE ai_profiles SET is_active = 1
			WHERE id = (SELECT id FROM ai_profiles ORDER BY created_at ASC LIMIT 1)`)
		if err != nil {
			return fmt.Errorf("ai: reassign active profile: %w", err)
		}
	}
	return nil
}

// SetActive makes the given profile the active one (and every other profile
// inactive) in a single transaction.
func (s *ProfileStore) SetActive(ctx context.Context, id string) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ai: set active profile: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE ai_profiles SET is_active = 0 WHERE is_active = 1`); err != nil {
		return fmt.Errorf("ai: clear active profile: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE ai_profiles SET is_active = 1 WHERE id = ?`, id); err != nil {
		return fmt.Errorf("ai: set active profile %q: %w", id, err)
	}
	return tx.Commit()
}

// MigrateLegacyConfig moves the pre-profiles single-row ai_settings config
// into a "Default" profile, exactly once: it only fires when no profiles
// exist yet and a configured legacy row is present, and it clears the legacy
// row afterwards so it can never resurrect a since-deleted profile. The
// keychain entry is reused as-is (ownership transfers to the profile), so
// the user's saved API key survives the upgrade without a keychain rewrite.
func (s *ProfileStore) MigrateLegacyConfig(ctx context.Context) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ai_profiles`).Scan(&count); err != nil {
		return fmt.Errorf("ai: migrate legacy config: %w", err)
	}
	if count > 0 {
		return nil
	}

	var baseURL, model, keyringKey string
	err := s.db.QueryRowContext(ctx,
		`SELECT base_url, model, keyring_key FROM ai_settings WHERE id = 'default'`,
	).Scan(&baseURL, &model, &keyringKey)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("ai: read legacy config: %w", err)
	}
	if baseURL == "" && keyringKey == "" {
		return nil
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO ai_profiles (id, name, base_url, model, keyring_key, is_active)
		VALUES (?, 'Default', ?, ?, ?, 1)`,
		uuid.NewString(), baseURL, model, keyringKey)
	if err != nil {
		return fmt.Errorf("ai: migrate legacy config: %w", err)
	}

	if _, err := s.db.ExecContext(ctx, `DELETE FROM ai_settings WHERE id = 'default'`); err != nil {
		return fmt.Errorf("ai: clear legacy config: %w", err)
	}
	return nil
}
