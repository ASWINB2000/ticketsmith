// Package connections manages saved tracker connections (base URL, tracker
// kind). Tokens are never stored here — only a keyring_key reference into
// the OS keychain (see internal/secrets).
package connections

import "time"

// Connection is a saved tracker instance the user has configured.
type Connection struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	TrackerKind string    `json:"trackerKind"`
	BaseURL     string    `json:"baseUrl"`
	KeyringKey  string    `json:"-"`
	CreatedAt   time.Time `json:"createdAt"`
}
