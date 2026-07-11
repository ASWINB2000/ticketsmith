package openproject

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"ticketsmith/internal/tracker"
)

type halElement struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
}

type halCollection struct {
	Embedded struct {
		Elements []halElement `json:"elements"`
	} `json:"_embedded"`
}

// GetTypes fetches and caches the OpenProject instance's work package types.
func (c *Client) GetTypes(ctx context.Context) ([]tracker.TicketType, error) {
	c.mu.RLock()
	if c.typesCache != nil {
		cached := c.typesCache
		c.mu.RUnlock()
		return cached, nil
	}
	c.mu.RUnlock()

	var col halCollection
	if err := c.doRequest(ctx, http.MethodGet, "/api/v3/types", nil, &col); err != nil {
		return nil, err
	}

	types := make([]tracker.TicketType, 0, len(col.Embedded.Elements))
	for _, e := range col.Embedded.Elements {
		types = append(types, tracker.TicketType{ID: strconv.Itoa(e.ID), Name: e.Name})
	}

	c.mu.Lock()
	c.typesCache = types
	c.mu.Unlock()
	return types, nil
}

// GetProjects fetches and caches the OpenProject instance's projects.
func (c *Client) GetProjects(ctx context.Context) ([]tracker.Project, error) {
	c.mu.RLock()
	if c.projectsCache != nil {
		cached := c.projectsCache
		c.mu.RUnlock()
		return cached, nil
	}
	c.mu.RUnlock()

	var col halCollection
	if err := c.doRequest(ctx, http.MethodGet, "/api/v3/projects", nil, &col); err != nil {
		return nil, err
	}

	projects := make([]tracker.Project, 0, len(col.Embedded.Elements))
	for _, e := range col.Embedded.Elements {
		projects = append(projects, tracker.Project{ID: strconv.Itoa(e.ID), Name: e.Name, Identifier: e.Identifier})
	}

	c.mu.Lock()
	c.projectsCache = projects
	c.mu.Unlock()
	return projects, nil
}

// GetPriorities fetches and caches the OpenProject instance's ticket priorities.
func (c *Client) GetPriorities(ctx context.Context) ([]tracker.Priority, error) {
	c.mu.RLock()
	if c.prioritiesCache != nil {
		cached := c.prioritiesCache
		c.mu.RUnlock()
		return cached, nil
	}
	c.mu.RUnlock()

	var col halCollection
	if err := c.doRequest(ctx, http.MethodGet, "/api/v3/priorities", nil, &col); err != nil {
		return nil, err
	}

	priorities := make([]tracker.Priority, 0, len(col.Embedded.Elements))
	for _, e := range col.Embedded.Elements {
		priorities = append(priorities, tracker.Priority{ID: strconv.Itoa(e.ID), Name: e.Name})
	}

	c.mu.Lock()
	c.prioritiesCache = priorities
	c.mu.Unlock()
	return priorities, nil
}

// GetAssignees fetches and caches (per projectID) the users assignable within a project.
func (c *Client) GetAssignees(ctx context.Context, projectID string) ([]tracker.User, error) {
	c.mu.RLock()
	if cached, ok := c.assigneesCache[projectID]; ok {
		c.mu.RUnlock()
		return cached, nil
	}
	c.mu.RUnlock()

	var col halCollection
	path := fmt.Sprintf("/api/v3/projects/%s/available_assignees", projectID)
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &col); err != nil {
		return nil, err
	}

	users := make([]tracker.User, 0, len(col.Embedded.Elements))
	for _, e := range col.Embedded.Elements {
		users = append(users, tracker.User{ID: strconv.Itoa(e.ID), Name: e.Name})
	}

	c.mu.Lock()
	c.assigneesCache[projectID] = users
	c.mu.Unlock()
	return users, nil
}
