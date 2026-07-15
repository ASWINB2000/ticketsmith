package openproject

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"ticketsmith/internal/tracker"
)

type wpLinkRef struct {
	Href string `json:"href"`
}

type wpLinks struct {
	Project  *wpLinkRef `json:"project,omitempty"`
	Type     *wpLinkRef `json:"type,omitempty"`
	Assignee *wpLinkRef `json:"assignee,omitempty"`
	Parent   *wpLinkRef `json:"parent,omitempty"`
	Priority *wpLinkRef `json:"priority,omitempty"`
}

type wpDescription struct {
	Format string `json:"format"`
	Raw    string `json:"raw"`
}

type createWorkPackageRequest struct {
	Subject     string        `json:"subject"`
	Description wpDescription `json:"description"`
	StartDate   string        `json:"startDate,omitempty"`
	DueDate     string        `json:"dueDate,omitempty"`
	Links       wpLinks       `json:"_links"`
}

type workPackageResponse struct {
	ID    int `json:"id"`
	Links struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"_links"`
}

// CreateTicket creates a work package. Each template field is matched
// (case-insensitively, by label) against the target project+type's real
// custom fields; a match's value is posted to that custom field, otherwise
// the field is appended to the description as a markdown section, in the
// template's declared field order. Custom field discovery failing (e.g. an
// older OpenProject version) is non-fatal: every field just falls back to
// the description, matching this adapter's original v1 behavior.
func (c *Client) CreateTicket(ctx context.Context, projectID, typeID string, input tracker.TicketInput) (tracker.Ticket, error) {
	description := input.Description
	customFieldValues := map[string]interface{}{}

	if len(input.Fields) > 0 {
		schema, err := c.GetCustomFields(ctx, projectID, typeID)
		if err != nil {
			schema = nil
		}

		var b strings.Builder
		b.WriteString(description)
		for _, f := range input.Fields {
			if cf, ok := matchCustomField(schema, f.Label); ok {
				if cf.Type == "Formattable" {
					customFieldValues[cf.Key] = wpDescription{Format: "markdown", Raw: f.Value}
				} else {
					customFieldValues[cf.Key] = f.Value
				}
				continue
			}
			fmt.Fprintf(&b, "\n\n#### %s\n%s", f.Label, f.Value)
		}
		description = b.String()
	}

	links := wpLinks{
		Project: &wpLinkRef{Href: "/api/v3/projects/" + projectID},
		Type:    &wpLinkRef{Href: "/api/v3/types/" + typeID},
	}
	if input.AssigneeID != "" {
		links.Assignee = &wpLinkRef{Href: "/api/v3/users/" + input.AssigneeID}
	}
	if input.ParentID != "" {
		links.Parent = &wpLinkRef{Href: "/api/v3/work_packages/" + input.ParentID}
	}
	if input.PriorityID != "" {
		links.Priority = &wpLinkRef{Href: "/api/v3/priorities/" + input.PriorityID}
	}

	body := createWorkPackageRequest{
		Subject:     input.Subject,
		Description: wpDescription{Format: "markdown", Raw: description},
		StartDate:   input.StartDate,
		DueDate:     input.DueDate,
		Links:       links,
	}

	var reqBody interface{} = body
	if len(customFieldValues) > 0 {
		merged, err := mergeCustomFields(body, customFieldValues)
		if err != nil {
			return tracker.Ticket{}, err
		}
		reqBody = merged
	}

	var resp workPackageResponse
	if err := c.doRequest(ctx, http.MethodPost, "/api/v3/work_packages", reqBody, &resp); err != nil {
		return tracker.Ticket{}, err
	}

	id := strconv.Itoa(resp.ID)
	return tracker.Ticket{
		ID:  id,
		URL: c.baseURL + "/work_packages/" + id,
	}, nil
}

// matchCustomField finds the schema entry whose name case-insensitively
// matches a template field's label, if any.
func matchCustomField(schema []tracker.CustomFieldSchema, label string) (tracker.CustomFieldSchema, bool) {
	for _, cf := range schema {
		if strings.EqualFold(cf.Name, label) {
			return cf, true
		}
	}
	return tracker.CustomFieldSchema{}, false
}

// mergeCustomFields folds customFieldN values into the work package request
// body. createWorkPackageRequest is a fixed struct (custom field keys are
// per-instance and dynamic), so the merge goes through a JSON round-trip
// into a plain map.
func mergeCustomFields(body createWorkPackageRequest, customFields map[string]interface{}) (map[string]interface{}, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("openproject: marshal work package request: %w", err)
	}
	merged := map[string]interface{}{}
	if err := json.Unmarshal(raw, &merged); err != nil {
		return nil, fmt.Errorf("openproject: remarshal work package request: %w", err)
	}
	for k, v := range customFields {
		merged[k] = v
	}
	return merged, nil
}
