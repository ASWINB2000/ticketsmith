// Package notes holds the zero-friction capture inbox: freeform text notes
// that can be merged (AI-rephrased into one) or converted into a ticket via
// the existing Generate pipeline. See docs/NOTES_PLAN.md §1-10.
package notes

import "time"

// Note is a single freeform capture. Status is "active" (shown in the
// default list) or "merged" (hidden but never deleted — MergedFromIDs on
// the note that superseded it preserves lineage back to it).
type Note struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Content       string    `json:"content"`
	Status        string    `json:"status"`
	MergedFromIDs []string  `json:"mergedFromIds,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
