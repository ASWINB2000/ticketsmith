package updater

import (
	"os"
	"path/filepath"
)

// updateExePath locates Update.exe, which Velopack installs one directory
// above the running app: {packId}\current\ticketsmith.exe -> {packId}\Update.exe.
func updateExePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), "..", "Update.exe"), nil
}
