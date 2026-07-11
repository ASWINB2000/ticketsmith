package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const githubAPI = "https://api.github.com"

// Asset is one entry in a Velopack releases.{channel}.json feed.
type Asset struct {
	PackageId string `json:"PackageId"`
	Version   string `json:"Version"`
	Type      string `json:"Type"` // "Full" or "Delta"
	FileName  string `json:"FileName"`
	SHA1      string `json:"SHA1"`
	Size      int64  `json:"Size"`
}

// Feed is the parsed contents of a Velopack releases.{channel}.json file.
type Feed struct {
	Assets []Asset `json:"Assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type ghRelease struct {
	TagName     string    `json:"tag_name"`
	Body        string    `json:"body"` // markdown release notes, as written when the GitHub release was published
	HTMLURL     string    `json:"html_url"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []ghAsset `json:"assets"`
}

// FeedFetcher fetches the Velopack release feed for a GitHub-hosted repo.
type FeedFetcher struct {
	HTTPClient *http.Client
	Owner      string
	Repo       string
}

func (f *FeedFetcher) latestRelease(ctx context.Context) (*ghRelease, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", githubAPI, f.Owner, f.Repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("updater: build request: %w", err)
	}
	resp, err := f.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("updater: fetch latest release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("updater: latest release request failed: status %d", resp.StatusCode)
	}
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("updater: decode latest release: %w", err)
	}
	return &rel, nil
}

// FetchFeed returns the parsed releases.{channel}.json feed and the full
// GitHub release metadata it was found in (needed to resolve an asset's
// download URL later).
func (f *FeedFetcher) FetchFeed(ctx context.Context, channel string) (*Feed, *ghRelease, error) {
	rel, err := f.latestRelease(ctx)
	if err != nil {
		return nil, nil, err
	}

	feedName := fmt.Sprintf("releases.%s.json", channel)
	var feedURL string
	for _, a := range rel.Assets {
		if a.Name == feedName {
			feedURL = a.BrowserDownloadURL
			break
		}
	}
	if feedURL == "" {
		return nil, nil, fmt.Errorf("updater: no %s asset found in latest release %s", feedName, rel.TagName)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("updater: build feed request: %w", err)
	}
	resp, err := f.HTTPClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("updater: fetch feed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("updater: feed request failed: status %d", resp.StatusCode)
	}
	var feed Feed
	if err := json.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, nil, fmt.Errorf("updater: decode feed: %w", err)
	}
	return &feed, rel, nil
}

// AssetDownloadURL finds the browser_download_url for a named asset within
// a previously-fetched release.
func AssetDownloadURL(rel *ghRelease, fileName string) (string, error) {
	if rel == nil {
		return "", fmt.Errorf("updater: no release provided for asset %q", fileName)
	}
	for _, a := range rel.Assets {
		if a.Name == fileName {
			return a.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("updater: asset %q not found in release %s", fileName, rel.TagName)
}
