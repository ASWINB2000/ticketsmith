// Package ai defines the provider-agnostic interface for AI-based ticket
// generation. Concrete provider dispatch lives at the composition root
// (app.go), mirroring internal/tracker's structure.
package ai

import (
	"context"

	"ticketsmith/internal/templates"
)

// StructuredTicket is the AI's structured output for a single generation request.
type StructuredTicket struct {
	Subject     string            `json:"subject"`
	Description string            `json:"description"`
	Fields      map[string]string `json:"fields"`
}

// Provider generates a StructuredTicket from a template and freeform raw input.
type Provider interface {
	GenerateTicket(ctx context.Context, template templates.Template, rawInput string) (StructuredTicket, error)
}
