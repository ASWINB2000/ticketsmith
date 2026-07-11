// Package tracker defines the provider-agnostic interface for project
// tracker backends (OpenProject, and later Jira/Azure DevOps).
//
// The concrete kind -> implementation dispatch lives at the composition root
// (app.go) rather than in this package, since adapter packages (e.g.
// internal/tracker/openproject) import these shared model types and a
// factory here would create an import cycle.
package tracker

import "context"

type TicketType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Project struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Identifier string `json:"identifier,omitempty"`
}

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Priority struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// FieldValue is one template extraction field's value, carrying its display
// label so trackers can render a human-readable heading instead of the raw
// field name. Order matters: it follows the template's declared field order.
type FieldValue struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type TicketInput struct {
	Subject     string       `json:"subject"`
	Description string       `json:"description"`
	Fields      []FieldValue `json:"fields,omitempty"`
	ParentID    string       `json:"parentId,omitempty"`
	AssigneeID  string       `json:"assigneeId,omitempty"`
	PriorityID  string       `json:"priorityId,omitempty"`
	StartDate   string       `json:"startDate,omitempty"` // YYYY-MM-DD
	DueDate     string       `json:"dueDate,omitempty"`   // YYYY-MM-DD
}

type Ticket struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type Tracker interface {
	GetTypes(ctx context.Context) ([]TicketType, error)
	GetProjects(ctx context.Context) ([]Project, error)
	GetAssignees(ctx context.Context, projectID string) ([]User, error)
	GetPriorities(ctx context.Context) ([]Priority, error)
	CreateTicket(ctx context.Context, projectID, typeID string, input TicketInput) (Ticket, error)
	// UploadAttachment attaches a file (already on disk, read by the caller)
	// to an existing ticket. Optional — called after CreateTicket succeeds,
	// once per selected file.
	UploadAttachment(ctx context.Context, ticketID, filename, contentType string, data []byte) error
}
