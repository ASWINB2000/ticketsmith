// Package templates holds user-defined ticket templates: which tracker type
// they target, which fields to extract from freeform input, and the AI
// instructions used to generate a structured ticket from that input.
package templates

import "time"

// Field describes one extraction field a template asks the AI to populate.
type Field struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Type        string `json:"type"` // "text" | "textarea"
	Description string `json:"description,omitempty"`
}

// Template is a user-defined ticket template.
type Template struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	TrackerTypeName string    `json:"trackerTypeName"`
	FieldsSchema    []Field   `json:"fieldsSchema"`
	AIInstructions  string    `json:"aiInstructions"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}
