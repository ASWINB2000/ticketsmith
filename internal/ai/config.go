package ai

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"ticketsmith/internal/secrets"
)

// Config is the user's AI provider configuration. The API key itself never
// appears here — only KeyringKey, a reference into the OS keychain.
type Config struct {
	BaseURL    string
	Model      string
	KeyringKey string
}

// ConfigStore persists the single-row "default" AI provider configuration.
type ConfigStore struct {
	db *sql.DB
}

func NewConfigStore(db *sql.DB) *ConfigStore {
	return &ConfigStore{db: db}
}

// Get returns the current AI config, or a zero-value Config if none has been saved yet.
func (s *ConfigStore) Get(ctx context.Context) (Config, error) {
	var cfg Config
	err := s.db.QueryRowContext(ctx,
		`SELECT base_url, model, keyring_key FROM ai_settings WHERE id = 'default'`,
	).Scan(&cfg.BaseURL, &cfg.Model, &cfg.KeyringKey)
	if errors.Is(err, sql.ErrNoRows) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("ai: get config: %w", err)
	}
	return cfg, nil
}

// Save upserts the AI config. If apiKey is empty, the existing key (if any)
// is left untouched; otherwise it's written to the OS keychain under a
// reused-or-newly-generated keyring key.
func (s *ConfigStore) Save(ctx context.Context, baseURL, model, apiKey string) error {
	cfg, err := s.Get(ctx)
	if err != nil {
		return err
	}

	keyringKey := cfg.KeyringKey
	if apiKey != "" {
		if keyringKey == "" {
			keyringKey = uuid.NewString()
		}
		if err := secrets.Set(keyringKey, apiKey); err != nil {
			return fmt.Errorf("ai: save api key: %w", err)
		}
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO ai_settings (id, base_url, model, keyring_key, updated_at)
		VALUES ('default', ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			base_url = excluded.base_url,
			model = excluded.model,
			keyring_key = excluded.keyring_key,
			updated_at = CURRENT_TIMESTAMP
	`, baseURL, model, keyringKey)
	if err != nil {
		return fmt.Errorf("ai: save config: %w", err)
	}
	return nil
}
