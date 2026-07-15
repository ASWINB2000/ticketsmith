package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"ticketsmith/internal/ai"
	"ticketsmith/internal/ai/openaicompat"
	"ticketsmith/internal/connections"
	"ticketsmith/internal/db"
	"ticketsmith/internal/logs"
	"ticketsmith/internal/notes"
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
	notesStore       *notes.Store

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
	a.notesStore = notes.NewStore(sqlDB)
	a.trackerCache = map[string]tracker.Tracker{}

	a.seedDefaultConnection()
	a.seedDefaultTemplates()

	go a.checkForUpdatesOnStartup()
}

// shutdown closes the database cleanly on app exit.
func (a *App) shutdown(ctx context.Context) {
	if a.db != nil {
		a.db.Close()
	}
}

// WindowReady is called once by the frontend after its first render, to
// reveal the window (started hidden via StartHidden in main.go). This avoids
// showing a blank or stale WebView2 frame while the bundle loads and React
// mounts its default screen.
func (a *App) WindowReady() {
	wailsruntime.WindowShow(a.ctx)
}

// checkForUpdatesOnStartup checks GitHub Releases for a newer version in the
// background on every launch. It never downloads or applies anything on its
// own, but if it finds an update it does push the full UpdateInfo via an
// "update:available" event, which the frontend uses to pop the update dialog
// once per launch (and to keep the sidebar's "Update available" state lit
// until the user actually installs, even if they dismiss the dialog with
// "Not now"). Silent if already up to date or if the check fails.
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

// GetLatestReleaseNotes returns the notes for the latest published GitHub
// release, regardless of whether it's newer than the running version — bound
// for the sidebar's version badge so a user can read "what's new" on demand.
func (a *App) GetLatestReleaseNotes() (*updater.ReleaseNotesInfo, error) {
	return updater.LatestReleaseNotes(a.ctx, a.updaterConfig())
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

// seedDefaultTemplates ensures each starter template in defaultTemplates
// exists by name, creating whichever ones are missing. Runs on every
// startup (cheap: one List query) so a preset the user hasn't touched keeps
// showing up even after a schema change adds a new one, but any template
// whose name is already taken (including a user's own edited copy) is left
// alone.
func (a *App) seedDefaultTemplates() {
	existing, err := a.templatesStore.List(a.ctx)
	if err != nil {
		return
	}

	byName := make(map[string]bool, len(existing))
	for _, t := range existing {
		byName[t.Name] = true
	}

	for _, t := range defaultTemplates {
		if byName[t.Name] {
			continue
		}
		if _, err := a.templatesStore.Create(a.ctx, t); err != nil {
			log.Printf("ticketsmith: seed default templates: %v", err)
		}
	}
}

var defaultTemplates = []templates.Template{
	{
		Name:            "User Story",
		TrackerTypeName: "User story",
		FieldsSchema: []templates.Field{
			{Name: "precondition", Label: "Pre-Condition", Type: "textarea"},
			{Name: "synopsis", Label: "Synopsis", Type: "textarea"},
			{Name: "acceptanceCriteria", Label: "Acceptance Criteria", Type: "textarea"},
		},
		AIInstructions: "Extract a thorough, well-elaborated user story from the raw input — never " +
			"compress it into a single line. Write a short, descriptive subject (e.g. \"Managing " +
			"Products\"). The description should open with the story in \"As a <role>, I should be able " +
			"to <capability> so that <benefit>\" form, followed by a detailed breakdown of the " +
			"requirements and behavior as a bulleted or numbered list — cover every distinct capability " +
			"or rule mentioned in the input, not just the headline one. Fill in Pre-Condition with every " +
			"piece of state that must already be true for the story to apply — required role/signup/login " +
			"state, existing data, prior setup, any dependent feature or configuration the input implies — " +
			"as a multi-sentence explanation or bulleted list, not a single term or clause; if the input " +
			"only implies one precondition, still spell out its full implication in 2–3 sentences rather " +
			"than a fragment. Synopsis must be a substantive multi-sentence paragraph (never a single " +
			"sentence) that restates the story's goal, who it's for, the value/benefit to the user, and any " +
			"context on where or how it fits into the broader product that the input touches on. Acceptance " +
			"Criteria should be a numbered list of concrete, testable conditions — each one a complete " +
			"sentence specific enough that someone could verify it without re-reading the raw input, and " +
			"covering every distinct behavior or rule the input mentioned rather than merging several into " +
			"one bullet. Leave a field blank rather than guessing if the input doesn't cover it at all.",
	},
	{
		Name:            "Bug",
		TrackerTypeName: "Bug",
		FieldsSchema: []templates.Field{
			{Name: "preconditions", Label: "Preconditions", Type: "textarea"},
			{Name: "stepsToReproduce", Label: "Steps to Reproduce", Type: "textarea"},
			{Name: "testData", Label: "Test Data", Type: "textarea"},
			{Name: "expectedResult", Label: "Expected Result", Type: "textarea"},
			{Name: "actualResult", Label: "Actual Result", Type: "textarea"},
		},
		AIInstructions: "Extract a clear, thorough, reproducible bug report from the raw " +
			"input, written with the density and precision of a professional QA bug " +
			"ticket — never compress it down to a single line. Write a concise subject " +
			"naming the defect and the affected area (e.g. \"Archived Count Not " +
			"Displayed in Dashboard Summary Cards\"). The description should be a full " +
			"paragraph that explains what's wrong, where it happens, what behavior was " +
			"expected instead, and any relevant context or impact — not just a one-line " +
			"label. Fill in Preconditions with the state that must already be true for " +
			"the bug to be observed (user role, existing data/setup, screen/viewport, " +
			"etc.), Steps to Reproduce as a detailed numbered list precise enough for " +
			"someone unfamiliar with the report to reproduce it exactly, Test Data with " +
			"any specific field values, components, or configurations called out in the " +
			"input, and Expected Result / Actual Result as full contrasting sentences " +
			"(not fragments or single words) describing correct vs. observed behavior. " +
			"Leave a field blank rather than guessing if the input doesn't cover it.",
	},
	{
		Name:            "Task",
		TrackerTypeName: "Task",
		FieldsSchema: []templates.Field{
			{Name: "scope", Label: "Scope", Type: "textarea"},
			{Name: "expectedBehavior", Label: "Expected Behavior", Type: "textarea"},
		},
		AIInstructions: "Extract a clear, thorough, actionable engineering task from the raw input — " +
			"never compress it into a single line. Write a concise subject describing the work to be " +
			"done (e.g. \"Product Browsing & Discovery UI Implementation - WEB\"). The description " +
			"should be a full paragraph (or more) explaining the feature/module, its functionality, and " +
			"any context or motivation given in the input — not a one-line summary. Fill in Scope with a " +
			"detailed account of the module(s) and functionality boundaries covered by the task, " +
			"including what's explicitly out of scope if the input says so, and Expected Behavior with a " +
			"thorough description of what the system should do once the task is complete — as a numbered " +
			"or bulleted breakdown covering every distinct behavior mentioned in the input, each written " +
			"as a full sentence rather than a fragment. Leave a field blank rather than guessing if the " +
			"input doesn't cover it.",
	},
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

func (a *App) GetTrackerPriorities(connectionID string) ([]tracker.Priority, error) {
	t, err := a.trackerFor(connectionID)
	if err != nil {
		return nil, err
	}
	return t.GetPriorities(a.ctx)
}

func (a *App) GetTrackerCustomFields(connectionID, projectID, typeID string) ([]tracker.CustomFieldSchema, error) {
	t, err := a.trackerFor(connectionID)
	if err != nil {
		return nil, err
	}
	return t.GetCustomFields(a.ctx, projectID, typeID)
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
// actually work. Prefers a free GET /models check; only falls back to
// spending a completion token if the provider doesn't support that.
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

// AIUsage fetches best-effort rate-limit/usage info for the given (or
// partially-saved) AI provider settings. Not every OpenAI-compatible
// provider reports this — check Usage.Supported before trusting the
// numeric fields.
func (a *App) AIUsage(baseURL, model, apiKey string) (openaicompat.Usage, error) {
	key, err := a.resolveAIKey(apiKey)
	if err != nil {
		return openaicompat.Usage{}, err
	}
	return openaicompat.New(baseURL, key, model).Usage(a.ctx)
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

// RefineTicket re-elaborates the current (possibly user-edited) draft
// instead of regenerating from rawInput alone, so manual edits/added points
// survive and get built out rather than discarded — see
// ai.Provider.RefineTicket. Writes its own audit log row, same as
// GenerateTicket.
func (a *App) RefineTicket(connectionID, templateID, rawInput string, current ai.StructuredTicket) (GenerateResult, error) {
	tmpl, err := a.templatesStore.Get(a.ctx, templateID)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("app: get template: %w", err)
	}

	provider, err := a.aiProvider()
	if err != nil {
		return GenerateResult{}, err
	}

	ticket, refineErr := provider.RefineTicket(a.ctx, tmpl, rawInput, current)

	status := "success"
	errMsg := ""
	generatedJSON := ""
	if refineErr != nil {
		status = "failure"
		errMsg = refineErr.Error()
	} else if b, err := json.Marshal(ticket); err == nil {
		generatedJSON = string(b)
	}

	entry, logErr := a.logsStore.Create(a.ctx, logs.LogEntry{
		Action:           "refine",
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

	if refineErr != nil {
		return GenerateResult{LogID: entry.ID}, refineErr
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
// priorityID, startDate, and dueDate are optional (blank means "leave to
// tracker default").
func (a *App) CreateTicket(logID, connectionID, projectID, typeID string, ticket ai.StructuredTicket, parentID, assigneeID, priorityID, startDate, dueDate string) (tracker.Ticket, error) {
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
		PriorityID:  priorityID,
		StartDate:   startDate,
		DueDate:     dueDate,
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

// PickAttachments opens a native "choose files" dialog restricted to common
// image/video types and returns the selected absolute paths (empty if the
// user cancels). Reading the files happens later in UploadAttachments — this
// only collects the paths so the frontend can list what's staged.
func (a *App) PickAttachments() ([]string, error) {
	paths, err := wailsruntime.OpenMultipleFilesDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Attach images or videos",
		Filters: []wailsruntime.FileFilter{
			{
				DisplayName: "Images and videos",
				Pattern:     "*.png;*.jpg;*.jpeg;*.gif;*.webp;*.heic;*.bmp;*.mp4;*.mov;*.webm;*.avi;*.mkv",
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("app: pick attachments: %w", err)
	}
	if paths == nil {
		return []string{}, nil
	}
	return paths, nil
}

// maxAttachmentPreviewBytes bounds GetAttachmentPreview so a large picked
// file doesn't get fully base64-encoded and shipped across the IPC bridge
// just to render a thumbnail.
const maxAttachmentPreviewBytes = 8 * 1024 * 1024

// GetAttachmentPreview reads an image file already staged in the attachments
// list and returns it as a data: URL for the frontend to render as a
// thumbnail. Only meant to be called for image paths — the frontend already
// knows from the extension whether a path is a video and shows a generic
// icon for those instead, so this doesn't need to distinguish kinds itself.
func (a *App) GetAttachmentPreview(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("app: stat attachment: %w", err)
	}
	if info.Size() > maxAttachmentPreviewBytes {
		return "", fmt.Errorf("app: %q is too large to preview", filepath.Base(path))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("app: read attachment: %w", err)
	}
	contentType := mime.TypeByExtension(filepath.Ext(path))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

// UploadAttachments reads each file at paths from disk and attaches it to an
// already-created ticket. Called after CreateTicket succeeds; attachments are
// optional, so a failure here is reported but doesn't undo the ticket. Errors
// from individual files are joined so the caller can report all failures at
// once rather than stopping at the first. Temp files created by
// SaveClipboardAttachment are cleaned up after a successful upload.
func (a *App) UploadAttachments(connectionID, ticketID string, paths []string) error {
	t, err := a.trackerFor(connectionID)
	if err != nil {
		return err
	}

	var errs []error
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			errs = append(errs, fmt.Errorf("read %q: %w", filepath.Base(p), err))
			continue
		}
		contentType := mime.TypeByExtension(filepath.Ext(p))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		if err := t.UploadAttachment(a.ctx, ticketID, filepath.Base(p), contentType, data); err != nil {
			errs = append(errs, fmt.Errorf("upload %q: %w", filepath.Base(p), err))
			continue
		}
		a.discardIfClipboardTemp(p)
	}
	if len(errs) > 0 {
		return fmt.Errorf("app: upload attachments: %w", errors.Join(errs...))
	}
	return nil
}

// clipboardAttachmentPrefix marks temp files SaveClipboardAttachment creates,
// so cleanup logic can recognize and remove them without ever touching a
// real file-picker path that happens to live elsewhere on disk.
const clipboardAttachmentPrefix = "ticketsmith-clip-"

// SaveClipboardAttachment writes a pasted image/video (base64-encoded by the
// frontend from clipboard data, which has no path on disk) to a temp file and
// returns its path, so it can be staged in the same attachments list as
// files picked via PickAttachments and uploaded the same way.
func (a *App) SaveClipboardAttachment(base64Data, suggestedName string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", fmt.Errorf("app: decode clipboard attachment: %w", err)
	}

	ext := filepath.Ext(suggestedName)
	if ext == "" {
		ext = ".png"
	}
	f, err := os.CreateTemp("", clipboardAttachmentPrefix+"*"+ext)
	if err != nil {
		return "", fmt.Errorf("app: create temp file: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return "", fmt.Errorf("app: write temp file: %w", err)
	}
	return f.Name(), nil
}

// DiscardClipboardAttachment removes a temp file created by
// SaveClipboardAttachment — called when the user removes a pasted attachment
// before creating the ticket. A no-op for file-picker paths.
func (a *App) DiscardClipboardAttachment(path string) {
	a.discardIfClipboardTemp(path)
}

func (a *App) discardIfClipboardTemp(path string) {
	if strings.HasPrefix(filepath.Base(path), clipboardAttachmentPrefix) {
		os.Remove(path)
	}
}

// ----- Logs screen -----

func (a *App) ListLogs(filter logs.Filter) ([]logs.LogEntry, error) {
	return a.logsStore.List(a.ctx, filter)
}

func (a *App) GetLog(id string) (logs.LogEntry, error) {
	return a.logsStore.Get(a.ctx, id)
}

// ----- Notes screen -----

func (a *App) ListNotes() ([]notes.Note, error) {
	return a.notesStore.List(a.ctx)
}

func (a *App) CreateNote(title, content string) (notes.Note, error) {
	return a.notesStore.Create(a.ctx, title, content)
}

func (a *App) UpdateNote(id, title, content string) (notes.Note, error) {
	return a.notesStore.Update(a.ctx, id, title, content)
}

func (a *App) DeleteNote(id string) error {
	return a.notesStore.Delete(a.ctx, id)
}

// MergeNotes runs Rephrase over the given notes' current content and
// returns the draft for the frontend's editable preview. It does not
// persist anything — see ConfirmMerge for that.
func (a *App) MergeNotes(noteIDs []string) (string, error) {
	if len(noteIDs) == 0 {
		return "", fmt.Errorf("app: merge requires at least one note")
	}

	contents := make([]string, 0, len(noteIDs))
	for _, id := range noteIDs {
		n, err := a.notesStore.Get(a.ctx, id)
		if err != nil {
			return "", fmt.Errorf("app: get note %q: %w", id, err)
		}
		contents = append(contents, n.Content)
	}

	provider, err := a.aiProvider()
	if err != nil {
		return "", err
	}
	draft, err := provider.Rephrase(a.ctx, contents)
	if err != nil {
		return "", fmt.Errorf("app: rephrase notes: %w", err)
	}
	return draft, nil
}

// ConfirmMerge persists a merge: creates a new note with finalContent and
// merged_from_ids set to noteIDs, and marks each source note 'merged'.
func (a *App) ConfirmMerge(noteIDs []string, title, finalContent string) (notes.Note, error) {
	if len(noteIDs) == 0 {
		return notes.Note{}, fmt.Errorf("app: merge requires at least one note")
	}
	return a.notesStore.CreateMerged(a.ctx, noteIDs, title, finalContent)
}
