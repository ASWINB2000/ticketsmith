package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	goruntime "runtime"
	"sync"
	"time"

	"github.com/joho/godotenv"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"ticketsmith/internal/ai"
	"ticketsmith/internal/ai/openaicompat"
	"ticketsmith/internal/connections"
	"ticketsmith/internal/db"
	"ticketsmith/internal/logs"
	"ticketsmith/internal/prefs"
	"ticketsmith/internal/secrets"
	"ticketsmith/internal/templates"
	"ticketsmith/internal/tracker"
	"ticketsmith/internal/tracker/openproject"
	"ticketsmith/internal/updater"

	"database/sql"
)

// App struct is the single Wails-bound backend for Ticketsmith.
type App struct {
	ctx     context.Context
	db      *sql.DB
	version string

	connectionsStore *connections.Store
	templatesStore   *templates.Store
	logsStore        *logs.Store
	aiConfigStore    *ai.ConfigStore
	prefsStore       *prefs.Store

	trackerMu    sync.Mutex
	trackerCache map[string]tracker.Tracker
}

// AISettingsView is what the frontend sees for AI provider config — never
// the plaintext key, only whether one has been saved.
type AISettingsView struct {
	BaseURL string `json:"baseUrl"`
	Model   string `json:"model"`
	HasKey  bool   `json:"hasKey"`
}

// GenerateResult is the response to GenerateTicket: the log row it was
// recorded under, plus the AI's structured output.
type GenerateResult struct {
	LogID  string              `json:"logId"`
	Ticket ai.StructuredTicket `json:"ticket"`
}

// NewApp creates a new App application struct
func NewApp(version string) *App {
	return &App{version: version}
}

// startup is called when the app starts. The context is saved so we can call
// the runtime methods, and all backend state (DB, stores) is wired here.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Dev convenience only — no-op if no .env is present (e.g. packaged builds).
	_ = godotenv.Load()

	dbPath, err := db.DefaultPath()
	if err != nil {
		log.Fatalf("ticketsmith: resolve database path: %v", err)
	}
	sqlDB, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("ticketsmith: open database: %v", err)
	}
	a.db = sqlDB

	a.connectionsStore = connections.NewStore(sqlDB)
	a.templatesStore = templates.NewStore(sqlDB)
	a.logsStore = logs.NewStore(sqlDB)
	a.aiConfigStore = ai.NewConfigStore(sqlDB)
	a.prefsStore = prefs.NewStore(sqlDB)
	a.trackerCache = map[string]tracker.Tracker{}

	a.seedDefaultConnection()

	go a.checkForUpdatesOnStartup()
}

// shutdown closes the database cleanly on app exit.
func (a *App) shutdown(ctx context.Context) {
	if a.db != nil {
		a.db.Close()
	}
}

// checkForUpdatesOnStartup checks GitHub Releases for a newer version in the
// background on every launch. It never downloads or applies anything on its
// own — if it finds an update, it pushes an "update:available" event so the
// frontend can show the same confirmation dialog the manual "Check for
// updates" button uses (see CheckForUpdates below). Silent if already up to
// date or if the check fails.
func (a *App) checkForUpdatesOnStartup() {
	info, err := updater.Check(a.ctx, a.updaterConfig())
	if err != nil {
		log.Printf("ticketsmith: update check failed: %v", err)
		return
	}
	if info != nil {
		wailsruntime.EventsEmit(a.ctx, "update:available", info)
	}
}

// Version returns the running app version, for display in the sidebar footer.
func (a *App) Version() string {
	return a.version
}

// CheckForUpdates is bound for the frontend's manual "Check for updates"
// button — it only reports whether an update exists so the frontend can
// prompt the user first.
func (a *App) CheckForUpdates() (*updater.UpdateInfo, error) {
	return updater.Check(a.ctx, a.updaterConfig())
}

// DownloadUpdate downloads the package described by info, emitting
// "update:download-progress" events (fraction 0.0-1.0) as it goes. Called
// only after the user confirms "Update now" in the frontend dialog.
func (a *App) DownloadUpdate(info updater.UpdateInfo) (string, error) {
	return updater.Download(a.ctx, a.updaterConfig(), &info, func(fraction float64) {
		wailsruntime.EventsEmit(a.ctx, "update:download-progress", fraction)
	})
}

// InstallUpdate applies a previously-downloaded package and restarts the
// app. Called only after the user confirms "Install & Restart" in the
// frontend's "ready to install" dialog.
func (a *App) InstallUpdate(pkgPath string) error {
	if err := updater.Install(pkgPath); err != nil {
		return err
	}
	wailsruntime.EventsEmit(a.ctx, "update:applying")
	time.Sleep(500 * time.Millisecond) // let the frontend show the toast before we exit
	wailsruntime.Quit(a.ctx)
	return nil
}

func (a *App) updaterConfig() updater.Config {
	return updater.Config{
		Owner:          "ASWINB2000",
		Repo:           "ticketsmith",
		Channel:        updateChannel(),
		CurrentVersion: a.version,
	}
}

func updateChannel() string {
	if goruntime.GOOS == "darwin" {
		return "osx"
	}
	return "win"
}

// seedDefaultConnection auto-creates a "Default" connection from .env
// credentials on first launch, purely as a dev convenience — never triggers
// once a real connection exists or in a packaged build with no .env.
func (a *App) seedDefaultConnection() {
	existing, err := a.connectionsStore.List(a.ctx)
	if err != nil || len(existing) > 0 {
		return
	}
	baseURL := os.Getenv("OPENPROJECT_BASE_URL")
	token := os.Getenv("OPENPROJECT_API_TOKEN")
	if baseURL == "" || token == "" {
		return
	}
	if _, err := a.connectionsStore.Create(a.ctx, "Default", "openproject", baseURL, token); err != nil {
		log.Printf("ticketsmith: seed default connection: %v", err)
	}
}

// trackerFor returns a cached (or newly built) Tracker for a connection.
// This is where tracker "kind" dispatch happens, since adapter packages
// import internal/tracker's shared types and can't be imported back into it.
func (a *App) trackerFor(connectionID string) (tracker.Tracker, error) {
	a.trackerMu.Lock()
	defer a.trackerMu.Unlock()

	if t, ok := a.trackerCache[connectionID]; ok {
		return t, nil
	}

	conn, err := a.connectionsStore.Get(a.ctx, connectionID)
	if err != nil {
		return nil, fmt.Errorf("app: get connection %q: %w", connectionID, err)
	}
	token, err := secrets.Get(conn.KeyringKey)
	if err != nil {
		return nil, fmt.Errorf("app: get token for connection %q: %w", connectionID, err)
	}

	var t tracker.Tracker
	switch conn.TrackerKind {
	case "openproject":
		t = openproject.NewClient(conn.BaseURL, token)
	default:
		return nil, fmt.Errorf("app: unsupported tracker kind %q", conn.TrackerKind)
	}

	a.trackerCache[connectionID] = t
	return t, nil
}

// invalidateTracker drops a cached Tracker so the next use rebuilds it
// (e.g. after credentials/base URL change).
func (a *App) invalidateTracker(connectionID string) {
	a.trackerMu.Lock()
	defer a.trackerMu.Unlock()
	delete(a.trackerCache, connectionID)
}

func (a *App) aiProvider() (ai.Provider, error) {
	cfg, err := a.aiConfigStore.Get(a.ctx)
	if err != nil {
		return nil, fmt.Errorf("app: get AI config: %w", err)
	}
	if cfg.BaseURL == "" || cfg.KeyringKey == "" {
		return nil, fmt.Errorf("AI provider is not configured — add it on the Connect screen")
	}
	apiKey, err := secrets.Get(cfg.KeyringKey)
	if err != nil {
		return nil, fmt.Errorf("app: get AI api key: %w", err)
	}
	return openaicompat.New(cfg.BaseURL, apiKey, cfg.Model), nil
}

// ----- Connect screen: tracker connections -----

func (a *App) ListConnections() ([]connections.Connection, error) {
	return a.connectionsStore.List(a.ctx)
}

func (a *App) CreateConnection(name, trackerKind, baseURL, token string) (connections.Connection, error) {
	return a.connectionsStore.Create(a.ctx, name, trackerKind, baseURL, token)
}

func (a *App) UpdateConnection(id, name, baseURL, token string) (connections.Connection, error) {
	conn, err := a.connectionsStore.Update(a.ctx, id, name, baseURL, token)
	if err == nil {
		a.invalidateTracker(id)
	}
	return conn, err
}

func (a *App) DeleteConnection(id string) error {
	err := a.connectionsStore.Delete(a.ctx, id)
	if err == nil {
		a.invalidateTracker(id)
	}
	return err
}

// TestConnection forces a fresh (uncached) live check against the tracker.
func (a *App) TestConnection(id string) error {
	a.invalidateTracker(id)
	t, err := a.trackerFor(id)
	if err != nil {
		return err
	}
	if _, err := t.GetProjects(a.ctx); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	return nil
}

// ----- Connect/Generate screens: tracker metadata -----

func (a *App) GetTrackerTypes(connectionID string) ([]tracker.TicketType, error) {
	t, err := a.trackerFor(connectionID)
	if err != nil {
		return nil, err
	}
	return t.GetTypes(a.ctx)
}

func (a *App) GetTrackerProjects(connectionID string) ([]tracker.Project, error) {
	t, err := a.trackerFor(connectionID)
	if err != nil {
		return nil, err
	}
	return t.GetProjects(a.ctx)
}

func (a *App) GetTrackerAssignees(connectionID, projectID string) ([]tracker.User, error) {
	t, err := a.trackerFor(connectionID)
	if err != nil {
		return nil, err
	}
	return t.GetAssignees(a.ctx, projectID)
}

// ----- Connect screen: AI provider settings -----

func (a *App) GetAISettings() (AISettingsView, error) {
	cfg, err := a.aiConfigStore.Get(a.ctx)
	if err != nil {
		return AISettingsView{}, err
	}
	return AISettingsView{BaseURL: cfg.BaseURL, Model: cfg.Model, HasKey: cfg.KeyringKey != ""}, nil
}

func (a *App) SaveAISettings(baseURL, model, apiKey string) error {
	return a.aiConfigStore.Save(a.ctx, baseURL, model, apiKey)
}

// resolveAIKey returns apiKey if non-empty, otherwise the currently-saved
// key — so callers can fetch models / test a connection without retyping a
// key that's already stored in the keychain.
func (a *App) resolveAIKey(apiKey string) (string, error) {
	if apiKey != "" {
		return apiKey, nil
	}
	cfg, err := a.aiConfigStore.Get(a.ctx)
	if err != nil {
		return "", err
	}
	if cfg.KeyringKey == "" {
		return "", fmt.Errorf("no API key saved yet — enter one first")
	}
	return secrets.Get(cfg.KeyringKey)
}

// ListAIModels fetches available model IDs from an OpenAI-compatible
// endpoint. If apiKey is blank, the currently-saved key is used.
func (a *App) ListAIModels(baseURL, apiKey string) ([]string, error) {
	key, err := a.resolveAIKey(apiKey)
	if err != nil {
		return nil, err
	}
	return openaicompat.ListModels(a.ctx, baseURL, key)
}

// TestAISettings verifies the given (or partially-saved) AI provider settings
// actually work, without spending a completion token.
func (a *App) TestAISettings(baseURL, model, apiKey string) error {
	key, err := a.resolveAIKey(apiKey)
	if err != nil {
		return err
	}
	if err := openaicompat.New(baseURL, key, model).Ping(a.ctx); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	return nil
}

// ----- Templates screen -----

func (a *App) ListTemplates() ([]templates.Template, error) {
	return a.templatesStore.List(a.ctx)
}

func (a *App) GetTemplate(id string) (templates.Template, error) {
	return a.templatesStore.Get(a.ctx, id)
}

func (a *App) CreateTemplate(t templates.Template) (templates.Template, error) {
	return a.templatesStore.Create(a.ctx, t)
}

func (a *App) UpdateTemplate(t templates.Template) (templates.Template, error) {
	return a.templatesStore.Update(a.ctx, t)
}

func (a *App) DeleteTemplate(id string) error {
	return a.templatesStore.Delete(a.ctx, id)
}

// ----- Generate screen -----

// GetGenerateDestination returns the last connection/project the user
// configured on the Generate screen, so it can be restored across restarts.
func (a *App) GetGenerateDestination() (prefs.GenerateDestination, error) {
	return a.prefsStore.GetGenerateDestination(a.ctx)
}

// SaveGenerateDestination remembers the connection/project the user just
// configured on the Generate screen.
func (a *App) SaveGenerateDestination(connectionID, projectID string) error {
	return a.prefsStore.SaveGenerateDestination(a.ctx, connectionID, projectID)
}

// GenerateTicket runs AI generation and writes an audit log row (even on
// failure). connectionID is recorded on the log entry so Logs can filter by
// it, though the AI call itself is tracker-agnostic.
func (a *App) GenerateTicket(connectionID, templateID, rawInput string) (GenerateResult, error) {
	tmpl, err := a.templatesStore.Get(a.ctx, templateID)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("app: get template: %w", err)
	}

	provider, err := a.aiProvider()
	if err != nil {
		return GenerateResult{}, err
	}

	ticket, genErr := provider.GenerateTicket(a.ctx, tmpl, rawInput)

	status := "success"
	errMsg := ""
	generatedJSON := ""
	if genErr != nil {
		status = "failure"
		errMsg = genErr.Error()
	} else if b, err := json.Marshal(ticket); err == nil {
		generatedJSON = string(b)
	}

	entry, logErr := a.logsStore.Create(a.ctx, logs.LogEntry{
		Action:           "generate",
		ConnectionID:     connectionID,
		TemplateID:       templateID,
		RawInput:         rawInput,
		GeneratedContent: generatedJSON,
		Status:           status,
		ErrorMessage:     errMsg,
	})
	if logErr != nil {
		return GenerateResult{}, fmt.Errorf("app: write log entry: %w", logErr)
	}

	if genErr != nil {
		return GenerateResult{LogID: entry.ID}, genErr
	}
	return GenerateResult{LogID: entry.ID, Ticket: ticket}, nil
}

// orderedFieldValues resolves each extraction field's display label and
// declared order from the template recorded on the log entry created during
// generation. Fields no longer present on that template (e.g. it was edited
// or deleted since generation) are still included, using the raw field name
// as a fallback label, appended after the ones the template still declares.
func (a *App) orderedFieldValues(logID string, fields map[string]string) ([]tracker.FieldValue, error) {
	if len(fields) == 0 {
		return nil, nil
	}

	entry, err := a.logsStore.Get(a.ctx, logID)
	if err != nil {
		return nil, fmt.Errorf("app: get log entry: %w", err)
	}

	out := make([]tracker.FieldValue, 0, len(fields))
	seen := make(map[string]bool, len(fields))

	if entry.TemplateID != "" {
		if tmpl, err := a.templatesStore.Get(a.ctx, entry.TemplateID); err == nil {
			for _, f := range tmpl.FieldsSchema {
				v, ok := fields[f.Name]
				if !ok {
					continue
				}
				label := f.Label
				if label == "" {
					label = f.Name
				}
				out = append(out, tracker.FieldValue{Label: label, Value: v})
				seen[f.Name] = true
			}
		}
	}

	for name, v := range fields {
		if seen[name] {
			continue
		}
		out = append(out, tracker.FieldValue{Label: name, Value: v})
	}

	return out, nil
}

// CreateTicket submits the (possibly user-edited) structured ticket to the
// tracker and mutates the same log row in place with the final result.
func (a *App) CreateTicket(logID, connectionID, projectID, typeID string, ticket ai.StructuredTicket, parentID, assigneeID string) (tracker.Ticket, error) {
	t, err := a.trackerFor(connectionID)
	if err != nil {
		return tracker.Ticket{}, err
	}

	fields, err := a.orderedFieldValues(logID, ticket.Fields)
	if err != nil {
		return tracker.Ticket{}, err
	}

	input := tracker.TicketInput{
		Subject:     ticket.Subject,
		Description: ticket.Description,
		Fields:      fields,
		ParentID:    parentID,
		AssigneeID:  assigneeID,
	}

	result, createErr := t.CreateTicket(a.ctx, projectID, typeID, input)

	status := "success"
	errMsg := ""
	if createErr != nil {
		status = "failure"
		errMsg = createErr.Error()
	}
	finalJSON, _ := json.Marshal(ticket)

	if _, logErr := a.logsStore.Update(a.ctx, logID, "create", string(finalJSON), result.ID, result.URL, status, errMsg); logErr != nil && createErr == nil {
		return tracker.Ticket{}, fmt.Errorf("app: write log update: %w", logErr)
	}

	if createErr != nil {
		return tracker.Ticket{}, createErr
	}
	return result, nil
}

// ----- Logs screen -----

func (a *App) ListLogs(filter logs.Filter) ([]logs.LogEntry, error) {
	return a.logsStore.List(a.ctx, filter)
}

func (a *App) GetLog(id string) (logs.LogEntry, error) {
	return a.logsStore.Get(a.ctx, id)
}
