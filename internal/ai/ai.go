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

	// RefineTicket re-elaborates an already-generated (and possibly
	// user-edited) ticket, using the original raw input as supporting
	// context. Unlike GenerateTicket, it treats current's content as
	// ground truth to preserve and expand on rather than re-deriving
	// from rawInput alone — so a manual edit/added point survives and
	// gets built out rather than discarded on the next pass.
	RefineTicket(ctx context.Context, template templates.Template, rawInput string, current StructuredTicket) (StructuredTicket, error)

	// Rephrase combines one or more freeform notes into a single coherent
	// draft — plain text in, plain text out, no structured schema.
	Rephrase(ctx context.Context, notes []string) (string, error)
}
