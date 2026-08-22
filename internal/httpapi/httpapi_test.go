package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"taskt111-shortlink/internal/store"
)

func newServer(t *testing.T) http.Handler {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "h.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return NewHandler(s).Routes()
}

func TestCreateAndInfo(t *testing.T) {
	h := newServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/links", strings.NewReader(`{"target_url":"https://x.com","owner":"o"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create code = %d", rec.Code)
	}
	var l store.Link
	if err := json.Unmarshal(rec.Body.Bytes(), &l); err != nil {
		t.Fatal(err)
	}
	if l.Code == "" {
		t.Fatal("empty code")
	}
	req2 := httptest.NewRequest(http.MethodGet, "/api/links/"+l.Code+"/info", nil)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("info code = %d", rec2.Code)
	}
}

func TestResolveExistingRedirect(t *testing.T) {
	h := newServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/links", strings.NewReader(`{"target_url":"https://x.com"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var l store.Link
	if err := json.Unmarshal(rec.Body.Bytes(), &l); err != nil {
		t.Fatal(err)
	}
	req2 := httptest.NewRequest(http.MethodGet, "/"+l.Code, nil)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusFound {
		t.Fatalf("redirect code = %d", rec2.Code)
	}
	if loc := rec2.Header().Get("Location"); loc != "https://x.com" {
		t.Fatalf("location = %q", loc)
	}
}

func TestListLargeLimit(t *testing.T) {
	h := newServer(t)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/links", strings.NewReader(`{"target_url":"https://x.com/i"}`))
		h.ServeHTTP(httptest.NewRecorder(), req)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/links?limit=50", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list code = %d", rec.Code)
	}
	var resp struct {
		Links   []store.Link `json:"links"`
		HasMore bool         `json:"has_more"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Links) != 2 {
		t.Fatalf("links = %d, want 2", len(resp.Links))
	}
	if resp.HasMore {
		t.Fatal("has_more should be false when limit exceeds total")
	}
}

// TestRedirectLimitBoundary verifies that recording, counting and the redirect
// decision agree at the critical count: once MaxClicks is reached, the next
// redirect is blocked (410) instead of being allowed through.
func TestRedirectLimitBoundary(t *testing.T) {
	h := newServer(t)
	// Create a link with a click limit of 2.
	req := httptest.NewRequest(http.MethodPost, "/api/links", strings.NewReader(`{"target_url":"https://x.com","max_clicks":2}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create code = %d", rec.Code)
	}
	var l store.Link
	if err := json.Unmarshal(rec.Body.Bytes(), &l); err != nil {
		t.Fatal(err)
	}

	// Two redirects exhaust the allowed count; both must succeed (302).
	for i := 1; i <= 2; i++ {
		r := httptest.NewRequest(http.MethodGet, "/"+l.Code, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusFound {
			t.Fatalf("redirect #%d code = %d, want %d", i, w.Code, http.StatusFound)
		}
	}

	// The third redirect must be blocked at the boundary.
	r := httptest.NewRequest(http.MethodGet, "/"+l.Code, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusGone {
		t.Fatalf("redirect over limit code = %d, want %d (410 Gone)", w.Code, http.StatusGone)
	}
}

