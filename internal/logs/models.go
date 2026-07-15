// Package logs is the audit trail of every generate/create/edit/error action.
package logs

import "time"

type LogEntry struct {
	ID               string    `json:"id"`
	Timestamp        time.Time `json:"timestamp"`
	Action           string    `json:"action"` // "generate" | "refine" | "create" | "edit" | "error"
	ConnectionID     string    `json:"connectionId,omitempty"`
	TemplateID       string    `json:"templateId,omitempty"`
	RawInput         string    `json:"rawInput,omitempty"`
	GeneratedContent string    `json:"generatedContent,omitempty"`
	FinalContent     string    `json:"finalContent,omitempty"`
	ResultTicketID   string    `json:"resultTicketId,omitempty"`
	ResultTicketURL  string    `json:"resultTicketUrl,omitempty"`
	Status           string    `json:"status"` // "success" | "failure"
	ErrorMessage     string    `json:"errorMessage,omitempty"`
}

// Filter narrows List results. Zero-value fields are not applied as filters.
type Filter struct {
	Action       string `json:"action,omitempty"`
	Status       string `json:"status,omitempty"`
	ConnectionID string `json:"connectionId,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	Offset       int    `json:"offset,omitempty"`
}
