package openproject

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
)

type attachmentMetadata struct {
	FileName    string `json:"fileName"`
	Description struct {
		Raw string `json:"raw"`
	} `json:"description"`
}

// UploadAttachment attaches a file to an existing work package via
// OpenProject's multipart attachments endpoint (metadata + file parts) —
// distinct from doRequest's plain-JSON requests, since this one is
// multipart/form-data.
func (c *Client) UploadAttachment(ctx context.Context, ticketID, filename, contentType string, data []byte) error {
	meta := attachmentMetadata{FileName: filename}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	metaPart, err := w.CreateFormField("metadata")
	if err != nil {
		return fmt.Errorf("openproject: create metadata part: %w", err)
	}
	if err := json.NewEncoder(metaPart).Encode(meta); err != nil {
		return fmt.Errorf("openproject: encode metadata: %w", err)
	}

	filePartHeader := make(textproto.MIMEHeader)
	filePartHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, filename))
	filePartHeader.Set("Content-Type", contentType)
	filePart, err := w.CreatePart(filePartHeader)
	if err != nil {
		return fmt.Errorf("openproject: create file part: %w", err)
	}
	if _, err := filePart.Write(data); err != nil {
		return fmt.Errorf("openproject: write file part: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("openproject: close multipart writer: %w", err)
	}

	path := "/api/v3/work_packages/" + ticketID + "/attachments"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, &buf)
	if err != nil {
		return fmt.Errorf("openproject: build attachment request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	auth := base64.StdEncoding.EncodeToString([]byte("apikey:" + c.token))
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("openproject POST %s: %w", path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("openproject POST %s: read response: %w", path, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := string(respBody)
		var opErr opErrorBody
		if json.Unmarshal(respBody, &opErr) == nil && opErr.Message != "" {
			msg = opErr.Message
		}
		return fmt.Errorf("openproject POST %s: %s (status %d)", path, msg, resp.StatusCode)
	}
	return nil
}
