// Package secrets stores and retrieves API tokens/keys via the OS credential
// store (Windows Credential Manager / macOS Keychain / Linux Secret Service).
// Values passed through here must never be written to SQLite.
package secrets

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const serviceName = "ticketsmith"

// Set stores value under key in the OS keychain.
func Set(key, value string) error {
	if err := keyring.Set(serviceName, key, value); err != nil {
		return fmt.Errorf("secrets: set %q: %w", key, err)
	}
	return nil
}

// Get retrieves the value stored under key from the OS keychain.
func Get(key string) (string, error) {
	value, err := keyring.Get(serviceName, key)
	if err != nil {
		return "", fmt.Errorf("secrets: get %q: %w", key, err)
	}
	return value, nil
}

// Delete removes the value stored under key from the OS keychain.
func Delete(key string) error {
	if err := keyring.Delete(serviceName, key); err != nil {
		return fmt.Errorf("secrets: delete %q: %w", key, err)
	}
	return nil
}
