package announcement

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testServer(t *testing.T, body string) Config {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return Config{URL: srv.URL, HTTPClient: srv.Client()}
}

func TestCheckReturnsManifestWhenIDIsNew(t *testing.T) {
	cfg := testServer(t, `{"ID":"ann-1","Title":"Heads up","Body":"Something changed","Level":"warning","URL":"https://example.com"}`)

	m, err := Check(context.Background(), cfg, "")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if m == nil {
		t.Fatal("Check() = nil, want manifest")
	}
	if m.ID != "ann-1" || m.Title != "Heads up" || m.Level != "warning" {
		t.Errorf("Check() = %+v, unexpected fields", m)
	}
}

func TestCheckReturnsNilWhenAlreadyDismissed(t *testing.T) {
	cfg := testServer(t, `{"ID":"ann-1","Title":"Heads up","Body":"Something changed"}`)

	m, err := Check(context.Background(), cfg, "ann-1")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if m != nil {
		t.Errorf("Check() = %+v, want nil (already dismissed)", m)
	}
}

func TestCheckReturnsNilForEmptyManifest(t *testing.T) {
	cfg := testServer(t, `{"ID":"","Title":"","Body":""}`)

	m, err := Check(context.Background(), cfg, "")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if m != nil {
		t.Errorf("Check() = %+v, want nil (no ID set)", m)
	}
}

func TestCheckErrorsOnNonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := Check(context.Background(), Config{URL: srv.URL, HTTPClient: srv.Client()}, "")
	if err == nil {
		t.Fatal("Check() error = nil, want error on 404")
	}
}
