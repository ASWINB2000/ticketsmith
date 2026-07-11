package openproject

import (
	"context"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUploadAttachmentSendsMultipartFileAndMetadata(t *testing.T) {
	var gotFileName, gotFileContent, gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/work_packages/42/attachments" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mediaType != "multipart/form-data" {
			t.Fatalf("Content-Type = %q, err = %v", r.Header.Get("Content-Type"), err)
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			part, err := mr.NextPart()
			if err != nil {
				break
			}
			if part.FormName() == "file" {
				gotFileName = part.FileName()
				gotContentType = part.Header.Get("Content-Type")
				buf := make([]byte, 1024)
				n, _ := part.Read(buf)
				gotFileContent = string(buf[:n])
			}
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	err := c.UploadAttachment(context.Background(), "42", "screenshot.png", "image/png", []byte("fake-png-bytes"))
	if err != nil {
		t.Fatalf("UploadAttachment: %v", err)
	}
	if gotFileName != "screenshot.png" {
		t.Errorf("file name = %q, want screenshot.png", gotFileName)
	}
	if gotFileContent != "fake-png-bytes" {
		t.Errorf("file content = %q, want fake-png-bytes", gotFileContent)
	}
	if gotContentType != "image/png" {
		t.Errorf("part Content-Type = %q, want image/png", gotContentType)
	}
}

func TestUploadAttachmentWrapsErrorOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"message":"file too large"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	err := c.UploadAttachment(context.Background(), "1", "video.mp4", "video/mp4", []byte("data"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got == "" {
		t.Errorf("error message empty")
	}
}
