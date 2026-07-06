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

type TicketInput struct {
	Subject     string            `json:"subject"`
	Description string            `json:"description"`
	Fields      map[string]string `json:"fields,omitempty"`
	ParentID    string            `json:"parentId,omitempty"`
	AssigneeID  string            `json:"assigneeId,omitempty"`
}

type Ticket struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type Tracker interface {
	GetTypes(ctx context.Context) ([]TicketType, error)
	GetProjects(ctx context.Context) ([]Project, error)
	GetAssignees(ctx context.Context, projectID string) ([]User, error)
	CreateTicket(ctx context.Context, projectID, typeID string, input TicketInput) (Ticket, error)
}
