package openproject

import (
	"context"
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

// CreateTicket creates a work package. Template fields are appended to the
// description as markdown sections, in the template's declared field order
// (v1 simplification — real per-type custom field mapping needs schema
// discovery not covered by this adapter yet).
func (c *Client) CreateTicket(ctx context.Context, projectID, typeID string, input tracker.TicketInput) (tracker.Ticket, error) {
	description := input.Description
	if len(input.Fields) > 0 {
		var b strings.Builder
		b.WriteString(description)
		for _, f := range input.Fields {
			fmt.Fprintf(&b, "\n\n## %s\n%s", f.Label, f.Value)
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

	var resp workPackageResponse
	if err := c.doRequest(ctx, http.MethodPost, "/api/v3/work_packages", body, &resp); err != nil {
		return tracker.Ticket{}, err
	}

	id := strconv.Itoa(resp.ID)
	return tracker.Ticket{
		ID:  id,
		URL: c.baseURL + "/work_packages/" + id,
	}, nil
}
