package updater

import "golang.org/x/mod/semver"

// normalize ensures a version string has the "v" prefix semver.Compare requires.
func normalize(v string) string {
	if len(v) == 0 || v[0] != 'v' {
		return "v" + v
	}
	return v
}

// IsNewer reports whether candidate is a strictly newer semantic version than current.
func IsNewer(current, candidate string) bool {
	return semver.Compare(normalize(candidate), normalize(current)) > 0
}
