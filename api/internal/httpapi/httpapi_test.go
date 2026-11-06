package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"leadcapture/api/internal/db"
)

const validLead = `{
	"first_name": " Zoë ",
	"last_name": "Martin",
	"email": "ZOE@Example.COM",
	"finance_amount": 38500.5,
	"salary": null,
	"state": "Queensland",
	"vehicle": null
}`

func newServer(t *testing.T) http.Handler {
	t.Helper()
	store, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "leads.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	frontend := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<!doctype html>app shell")}}
	return New(store, frontend)
}

func do(t *testing.T, handler http.Handler, method, target, body string) (int, map[string]any) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, reader)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	handler.ServeHTTP(recorder, request)

	result := map[string]any{}
	response := recorder.Result()
	if response.Header.Get("Content-Type") == "application/json; charset=utf-8" {
		raw, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(string(raw)) != "[]" {
			if err := json.Unmarshal(raw, &result); err != nil {
				t.Fatalf("decode %s: %v", raw, err)
			}
		}
	}
	return response.StatusCode, result
}

func TestListLeadsEmptyReturnsArray(t *testing.T) {
	recorder := httptest.NewRecorder()
	newServer(t).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/leads", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", recorder.Code)
	}
	if got := strings.TrimSpace(recorder.Body.String()); got != "[]" {
		t.Fatalf("body %s, want []", got)
	}
}

func TestCreateLeadReturnsSavedLead(t *testing.T) {
	handler := newServer(t)
	status, created := do(t, handler, http.MethodPost, "/api/leads", validLead)
	if status != http.StatusCreated {
		t.Fatalf("status %d, want 201: %v", status, created)
	}
	if created["id"] != float64(1) {
		t.Fatalf("id %v, want 1", created["id"])
	}
	if created["first_name"] != "Zoë" || created["email"] != "zoe@example.com" || created["state"] != "QLD" {
		t.Fatalf("expected normalized lead, got %v", created)
	}
	if created["salary"] != nil || created["vehicle"] != nil {
		t.Fatalf("optional fields must stay null, got %v", created)
	}
	createdAt, err := time.Parse(time.RFC3339, created["created_at"].(string))
	if err != nil || createdAt.IsZero() {
		t.Fatalf("created_at %v is not RFC3339", created["created_at"])
	}
}

func TestCreateLeadWithPartialVehicle(t *testing.T) {
	body := `{"first_name":"A","last_name":"B","email":"a@example.com","finance_amount":1,
		"state":"NSW","salary":60000.5,"vehicle":{"make":"Kia","model":"EV6","year":null}}`
	status, created := do(t, newServer(t), http.MethodPost, "/api/leads", body)
	if status != http.StatusCreated {
		t.Fatalf("status %d, want 201: %v", status, created)
	}
	vehicle, ok := created["vehicle"].(map[string]any)
	if !ok || vehicle["make"] != "Kia" || vehicle["model"] != "EV6" || vehicle["year"] != nil {
		t.Fatalf("unexpected vehicle %v", created["vehicle"])
	}
	if created["salary"] != 60000.5 {
		t.Fatalf("salary %v, want 60000.5", created["salary"])
	}
}

func TestCreateLeadMalformedJSON(t *testing.T) {
	for _, body := range []string{"", `{`, `{"first_name": }`, `[1, 2]`, `"just a string"`} {
		status, response := do(t, newServer(t), http.MethodPost, "/api/leads", body)
		if status != http.StatusBadRequest {
			t.Errorf("body %q: status %d, want 400", body, status)
		}
		if _, reported := response["error"]; !reported {
			t.Errorf("body %q: missing error message: %v", body, response)
		}
	}
}

func TestCreateLeadValidationErrors(t *testing.T) {
	body := `{"first_name":"","last_name":"","email":"","finance_amount":-1,
		"salary":0,"state":"ZZZ","vehicle":{"model":"EV6","year":1899}}`
	status, response := do(t, newServer(t), http.MethodPost, "/api/leads", body)
	if status != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, want 422: %v", status, response)
	}
	fields, ok := response["errors"].(map[string]any)
	if !ok {
		t.Fatalf("missing errors object: %v", response)
	}
	for field, want := range map[string]string{
		"first_name":     "is required",
		"last_name":      "is required",
		"email":          "is required",
		"finance_amount": "must be greater than 0",
		"salary":         "must be greater than 0",
		"state":          "must be a valid Australian state or territory",
		"vehicle.make":   "is required when a model or year is given",
		"vehicle.year":   "must be between 1990 and " + strconv.Itoa(time.Now().Year()+1),
	} {
		if got := fields[field]; got != want {
			t.Errorf("%s = %v, want %q", field, got, want)
		}
	}
}

func TestCreateLeadDuplicateEmail(t *testing.T) {
	handler := newServer(t)
	if status, _ := do(t, handler, http.MethodPost, "/api/leads", validLead); status != http.StatusCreated {
		t.Fatal("setup create failed")
	}
	status, response := do(t, handler, http.MethodPost, "/api/leads", validLead)
	if status != http.StatusConflict {
		t.Fatalf("status %d, want 409: %v", status, response)
	}
	fields, ok := response["errors"].(map[string]any)
	if !ok || fields["email"] == "" {
		t.Fatalf("expected email error, got %v", response)
	}
}

func TestListLeadsAfterCreate(t *testing.T) {
	handler := newServer(t)
	if status, _ := do(t, handler, http.MethodPost, "/api/leads", validLead); status != http.StatusCreated {
		t.Fatal("setup create failed")
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/leads", nil))
	var leads []map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &leads); err != nil {
		t.Fatalf("decode %s: %v", recorder.Body, err)
	}
	if len(leads) != 1 || leads[0]["email"] != "zoe@example.com" {
		t.Fatalf("unexpected leads %v", leads)
	}
}

func TestUnknownAPIPath(t *testing.T) {
	status, response := do(t, newServer(t), http.MethodGet, "/api/nothing", "")
	if status != http.StatusNotFound {
		t.Fatalf("status %d, want 404: %v", status, response)
	}
}

func TestFrontendServed(t *testing.T) {
	handler := newServer(t)
	for _, path := range []string{"/", "/leads/42"} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "app shell") {
			t.Errorf("GET %s: status %d, body %s", path, recorder.Code, recorder.Body)
		}
	}
}
