package openproject

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetCustomFieldsFiltersToKnownTypes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/projects/1/work_packages/form" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Write([]byte(`{"_embedded":{"schema":{
			"subject":{"type":"String","name":"Subject"},
			"customField5":{"type":"String","name":"Steps to reproduce"},
			"customField7":{"type":"Formattable","name":"Acceptance Criteria"},
			"customField9":{"type":"List","name":"Environment"}
		}}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	fields, err := c.GetCustomFields(context.Background(), "1", "2")
	if err != nil {
		t.Fatalf("GetCustomFields: %v", err)
	}

	if len(fields) != 2 {
		t.Fatalf("got %d fields, want 2 (subject and the List field should be excluded): %+v", len(fields), fields)
	}
	byKey := map[string]string{}
	for _, f := range fields {
		byKey[f.Key] = f.Name
	}
	if byKey["customField5"] != "Steps to reproduce" {
		t.Errorf("customField5 name = %q", byKey["customField5"])
	}
	if byKey["customField7"] != "Acceptance Criteria" {
		t.Errorf("customField7 name = %q", byKey["customField7"])
	}
}

func TestGetCustomFieldsCachesPerProjectAndType(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write([]byte(`{"_embedded":{"schema":{"customField1":{"type":"String","name":"Foo"}}}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	if _, err := c.GetCustomFields(context.Background(), "1", "2"); err != nil {
		t.Fatalf("GetCustomFields: %v", err)
	}
	if _, err := c.GetCustomFields(context.Background(), "1", "2"); err != nil {
		t.Fatalf("GetCustomFields (cached): %v", err)
	}
	if _, err := c.GetCustomFields(context.Background(), "1", "3"); err != nil {
		t.Fatalf("GetCustomFields (other type): %v", err)
	}
	if calls != 2 {
		t.Errorf("server was called %d times, want 2 (one per distinct project+type)", calls)
	}

	c.InvalidateCache()
	if _, err := c.GetCustomFields(context.Background(), "1", "2"); err != nil {
		t.Fatalf("GetCustomFields after invalidate: %v", err)
	}
	if calls != 3 {
		t.Errorf("server was called %d times after invalidate, want 3", calls)
	}
}

// OpenProject's schema map mixes field entries with schema metadata like
// "_type":"Schema" (a plain string, not a {type,name} object) — this must
// not blow up the whole decode.
func TestGetCustomFieldsIgnoresNonFieldSchemaEntries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"_embedded":{"schema":{
			"_type":"Schema",
			"_dependencies":[],
			"subject":{"type":"String","name":"Subject"},
			"customField5":{"type":"String","name":"Synopsis"}
		}}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	fields, err := c.GetCustomFields(context.Background(), "1", "2")
	if err != nil {
		t.Fatalf("GetCustomFields: %v", err)
	}
	if len(fields) != 1 || fields[0].Key != "customField5" || fields[0].Name != "Synopsis" {
		t.Fatalf("unexpected fields: %+v", fields)
	}
}

func TestGetCustomFieldsReturnsErrorOnFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"boom"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	if _, err := c.GetCustomFields(context.Background(), "1", "2"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
