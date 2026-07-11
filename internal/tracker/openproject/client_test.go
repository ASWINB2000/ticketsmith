package openproject

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetTypesSendsBasicAuthAndCaches(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("apikey:mytoken"))
		if got := r.Header.Get("Authorization"); got != wantAuth {
			t.Errorf("Authorization header = %q, want %q", got, wantAuth)
		}
		if r.URL.Path != "/api/v3/types" {
			t.Errorf("path = %q, want /api/v3/types", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"_embedded":{"elements":[{"id":1,"name":"Bug"},{"id":2,"name":"Task"}]}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "mytoken")

	types, err := c.GetTypes(context.Background())
	if err != nil {
		t.Fatalf("GetTypes: %v", err)
	}
	if len(types) != 2 || types[0].Name != "Bug" || types[1].ID != "2" {
		t.Fatalf("unexpected types: %+v", types)
	}

	// Second call should be served from cache, not hit the server again.
	if _, err := c.GetTypes(context.Background()); err != nil {
		t.Fatalf("second GetTypes: %v", err)
	}
	if calls != 1 {
		t.Errorf("server was called %d times, want 1 (cache not used)", calls)
	}

	c.InvalidateCache()
	if _, err := c.GetTypes(context.Background()); err != nil {
		t.Fatalf("GetTypes after invalidate: %v", err)
	}
	if calls != 2 {
		t.Errorf("server was called %d times after invalidate, want 2", calls)
	}
}

func TestGetProjectsParsesIdentifier(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"_embedded":{"elements":[{"id":5,"name":"Demo","identifier":"demo"}]}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	projects, err := c.GetProjects(context.Background())
	if err != nil {
		t.Fatalf("GetProjects: %v", err)
	}
	if len(projects) != 1 || projects[0].Identifier != "demo" {
		t.Fatalf("unexpected projects: %+v", projects)
	}
}

func TestGetPrioritiesParsesAndCaches(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/v3/priorities" {
			t.Errorf("path = %q, want /api/v3/priorities", r.URL.Path)
		}
		w.Write([]byte(`{"_embedded":{"elements":[{"id":7,"name":"Low"},{"id":8,"name":"High"}]}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	priorities, err := c.GetPriorities(context.Background())
	if err != nil {
		t.Fatalf("GetPriorities: %v", err)
	}
	if len(priorities) != 2 || priorities[0].Name != "Low" || priorities[1].ID != "8" {
		t.Fatalf("unexpected priorities: %+v", priorities)
	}

	if _, err := c.GetPriorities(context.Background()); err != nil {
		t.Fatalf("second GetPriorities: %v", err)
	}
	if calls != 1 {
		t.Errorf("server was called %d times, want 1 (cache not used)", calls)
	}
}

func TestGetAssigneesCachesPerProject(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write([]byte(`{"_embedded":{"elements":[{"id":9,"name":"Alice"}]}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	if _, err := c.GetAssignees(context.Background(), "1"); err != nil {
		t.Fatalf("GetAssignees: %v", err)
	}
	if _, err := c.GetAssignees(context.Background(), "1"); err != nil {
		t.Fatalf("GetAssignees (cached): %v", err)
	}
	if _, err := c.GetAssignees(context.Background(), "2"); err != nil {
		t.Fatalf("GetAssignees (other project): %v", err)
	}
	if calls != 2 {
		t.Errorf("server was called %d times, want 2 (one per distinct project)", calls)
	}
}

func TestDoRequestWrapsErrorOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"invalid token"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "bad-token")
	_, err := c.GetTypes(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "invalid token") || !strings.Contains(got, "401") {
		t.Errorf("error message = %q, want it to contain \"invalid token\" and \"401\"", got)
	}
}
