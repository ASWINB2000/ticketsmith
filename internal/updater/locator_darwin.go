package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// updateExePath locates UpdateMac inside the running .app bundle:
// AppName.app/Contents/MacOS/ticketsmith -> AppName.app/Contents/MacOS/UpdateMac.
func updateExePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	const marker = ".app/Contents/MacOS/"
	idx := strings.Index(exe, marker)
	if idx == -1 {
		return "", fmt.Errorf("updater: running executable %q is not inside a .app bundle", exe)
	}
	return filepath.Join(exe[:idx+len(marker)], "UpdateMac"), nil
}
