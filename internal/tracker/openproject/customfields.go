package openproject

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"ticketsmith/internal/tracker"
)

var customFieldKeyPattern = regexp.MustCompile(`^customField\d+$`)

// customFieldTypeAllowList are the schema "type" tags whose values this
// adapter knows how to populate from a plain extraction-field string.
// Other types (List, User, Date, Boolean, ...) are left unmatched so their
// template fields fall back to the description section.
var customFieldTypeAllowList = map[string]bool{
	"String":      true,
	"Formattable": true,
}

type wpFormRequest struct {
	Links struct {
		Type wpLinkRef `json:"type"`
	} `json:"_links"`
}

type schemaFieldEntry struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// wpFormResponse's schema map holds a mix of shapes: field entries like
// {"type":"String","name":"..."} alongside plain scalars for schema
// metadata (e.g. "_type":"Schema"), so its values are decoded lazily
// (json.RawMessage) rather than eagerly into a fixed struct.
type wpFormResponse struct {
	Embedded struct {
		Schema map[string]json.RawMessage `json:"schema"`
	} `json:"_embedded"`
}

// GetCustomFields fetches and caches (per projectID+typeID) the custom
// fields available on work packages of the given type in the given
// project, via OpenProject's work package "form" endpoint — the documented
// mechanism for discovering per-instance custom field schemas, since custom
// fields are configured per project+type and have no stable cross-instance
// identifier. Only fields whose declared type this adapter knows how to
// populate from a plain string (see customFieldTypeAllowList) are returned.
func (c *Client) GetCustomFields(ctx context.Context, projectID, typeID string) ([]tracker.CustomFieldSchema, error) {
	cacheKey := projectID + ":" + typeID
	c.mu.RLock()
	if cached, ok := c.customFieldsCache[cacheKey]; ok {
		c.mu.RUnlock()
		return cached, nil
	}
	c.mu.RUnlock()

	var body wpFormRequest
	body.Links.Type.Href = "/api/v3/types/" + typeID

	var resp wpFormResponse
	path := fmt.Sprintf("/api/v3/projects/%s/work_packages/form", projectID)
	if err := c.doRequest(ctx, http.MethodPost, path, body, &resp); err != nil {
		return nil, err
	}

	fields := make([]tracker.CustomFieldSchema, 0, len(resp.Embedded.Schema))
	for key, raw := range resp.Embedded.Schema {
		if !customFieldKeyPattern.MatchString(key) {
			continue
		}
		var entry schemaFieldEntry
		if err := json.Unmarshal(raw, &entry); err != nil || !customFieldTypeAllowList[entry.Type] {
			continue
		}
		fields = append(fields, tracker.CustomFieldSchema{Key: key, Name: entry.Name, Type: entry.Type})
	}

	c.mu.Lock()
	c.customFieldsCache[cacheKey] = fields
	c.mu.Unlock()
	return fields, nil
}
