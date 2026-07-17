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

// EditExample pairs what the AI generated for a ticket with what the user
// actually filed after hand-editing it — the raw material for tuning a
// template's instructions from real usage.
type EditExample struct {
	Generated StructuredTicket `json:"generated"`
	Final     StructuredTicket `json:"final"`
}

// TuningSuggestion is the AI's analysis of a template's edit history: what
// patterns the user's edits show, and a rewritten instruction set that would
// have produced the edited versions directly.
type TuningSuggestion struct {
	Summary               string `json:"summary"`
	SuggestedInstructions string `json:"suggestedInstructions"`
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

	// SuggestInstructions studies before/after pairs of tickets the user
	// edited by hand before filing and proposes an updated version of the
	// template's AI instructions that would have produced the edited
	// versions directly.
	SuggestInstructions(ctx context.Context, template templates.Template, examples []EditExample) (TuningSuggestion, error)
}
