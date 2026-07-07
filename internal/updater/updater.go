package updater

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config controls a single update check-and-apply run.
type Config struct {
	Owner          string
	Repo           string
	Channel        string // "win" or "osx"
	CurrentVersion string
	HTTPClient     *http.Client
}

// UpdateInfo describes an available update surfaced to the user before
// they choose to download it. Asset and Release must stay exported even
// though they're internal plumbing: this struct round-trips through the
// Wails JS<->Go boundary (returned by Check, then passed back into
// Download), and encoding/json silently drops unexported fields, which
// would leave Download with a nil Release on the way back from the frontend.
type UpdateInfo struct {
	Version      string
	ReleaseNotes string // raw markdown from the GitHub release body

	Asset   Asset
	Release *ghRelease
}

// Check looks for a newer "Full" release on cfg.Channel without
// downloading anything. Returns (nil, nil) if already up to date. Used by
// the manual "Check for updates" button.
func Check(ctx context.Context, cfg Config) (*UpdateInfo, error) {
	client := httpClient(cfg)
	fetcher := &FeedFetcher{HTTPClient: client, Owner: cfg.Owner, Repo: cfg.Repo}
	feed, rel, err := fetcher.FetchFeed(ctx, cfg.Channel)
	if err != nil {
		return nil, err
	}
	best := latestFullAsset(feed, cfg.CurrentVersion)
	if best == nil {
		return nil, nil
	}
	return &UpdateInfo{Version: best.Version, ReleaseNotes: rel.Body, Asset: *best, Release: rel}, nil
}

// ProgressFunc is called periodically during Download with the fraction
// (0.0-1.0) of the package downloaded so far.
type ProgressFunc func(fraction float64)

// Download fetches the package described by info to a temp file, invoking
// onProgress as bytes arrive, and returns the local package path. The
// download's SHA1 is verified against the feed's recorded checksum before
// the path is returned, since this package is about to replace the running
// application and a truncated/corrupted download must never be applied.
func Download(ctx context.Context, cfg Config, info *UpdateInfo, onProgress ProgressFunc) (string, error) {
	client := httpClient(cfg)
	downloadURL, err := AssetDownloadURL(info.Release, info.Asset.FileName)
	if err != nil {
		return "", err
	}
	return downloadToTemp(ctx, client, downloadURL, info.Asset, onProgress)
}

// Install hands the downloaded package at pkgPath to the bundled Velopack
// updater to apply and relaunch. The caller must exit immediately after
// this returns successfully so Update.exe/UpdateMac can replace the
// "current" directory. --waitPid tells the updater to wait for this
// process to actually exit before it touches files this process may still
// have open/locked; restarting into the new version afterward is already
// the default "apply" behavior (there is no separate "--restart" flag).
func Install(pkgPath string) error {
	exePath, err := updateExePath()
	if err != nil {
		return fmt.Errorf("updater: locate updater binary: %w", err)
	}
	cmd := exec.Command(exePath, "apply", "--package", pkgPath, "--waitPid", strconv.Itoa(os.Getpid()))
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("updater: launch updater: %w", err)
	}
	return nil
}

func httpClient(cfg Config) *http.Client {
	if cfg.HTTPClient != nil {
		return cfg.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

// latestFullAsset returns the newest "Full" asset in feed that is newer
// than currentVersion, or nil if there is none.
func latestFullAsset(feed *Feed, currentVersion string) *Asset {
	var best *Asset
	for i := range feed.Assets {
		a := &feed.Assets[i]
		if a.Type != "Full" || !IsNewer(currentVersion, a.Version) {
			continue
		}
		if best == nil || IsNewer(best.Version, a.Version) {
			best = a
		}
	}
	return best
}

// progressReader wraps an io.Reader, invoking onProgress with the running
// fraction of total bytes read so far.
type progressReader struct {
	io.Reader
	total      int64
	read       int64
	onProgress ProgressFunc
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.Reader.Read(b)
	p.read += int64(n)
	if p.onProgress != nil && p.total > 0 {
		p.onProgress(float64(p.read) / float64(p.total))
	}
	return n, err
}

func downloadToTemp(ctx context.Context, client *http.Client, url string, asset Asset, onProgress ProgressFunc) (string, error) {
	dst := filepath.Join(os.TempDir(), asset.FileName)

	// Reuse an already-downloaded copy if one exists and its checksum still
	// matches — e.g. the user checked, downloaded, then clicked "Later" and
	// is now clicking "Update now" again. The destination path is stable
	// per version (os.TempDir()/<versioned filename>), so a prior download
	// naturally lands here again without any extra bookkeeping.
	if asset.SHA1 != "" && fileMatchesSHA1(dst, asset.SHA1) {
		if onProgress != nil {
			onProgress(1.0)
		}
		return dst, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("updater: build download request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("updater: download package: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("updater: download failed: status %d", resp.StatusCode)
	}

	f, err := os.Create(dst)
	if err != nil {
		return "", fmt.Errorf("updater: create temp package file: %w", err)
	}
	defer f.Close()

	var src io.Reader = resp.Body
	if onProgress != nil && asset.Size > 0 {
		src = &progressReader{Reader: resp.Body, total: asset.Size, onProgress: onProgress}
	}

	hasher := sha1.New()
	if _, err := io.Copy(io.MultiWriter(f, hasher), src); err != nil {
		return "", fmt.Errorf("updater: write temp package file: %w", err)
	}

	if asset.SHA1 != "" {
		sum := hex.EncodeToString(hasher.Sum(nil))
		if !strings.EqualFold(sum, asset.SHA1) {
			os.Remove(dst)
			return "", fmt.Errorf("updater: downloaded package checksum mismatch: got %s, expected %s", sum, asset.SHA1)
		}
	}

	return dst, nil
}

// fileMatchesSHA1 reports whether the file at path exists and its SHA1
// matches expected, without holding the whole file in memory.
func fileMatchesSHA1(path, expected string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	hasher := sha1.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return false
	}
	return strings.EqualFold(hex.EncodeToString(hasher.Sum(nil)), expected)
}
