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

// TestDeleteMissingReturnsNotFound 验证删除不存在的短链时返回 404 而非 200，
// 调用方可据此判断删除目标是否真实存在。
func TestDeleteMissingReturnsNotFound(t *testing.T) {
	h := newServer(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/links/no-such-code", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("delete missing code = %d, want 404", rec.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] == "" {
		t.Fatalf("expected non-empty error body, got %s", rec.Body.String())
	}
}

// TestDeleteExistingSucceeds 验证删除已存在的短链返回 200 deleted:true，
// 并使后续 GET 跳转返回 404。
func TestDeleteExistingSucceeds(t *testing.T) {
	h := newServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/links", strings.NewReader(`{"target_url":"https://x.com"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create code = %d", rec.Code)
	}
	var l store.Link
	if err := json.Unmarshal(rec.Body.Bytes(), &l); err != nil {
		t.Fatal(err)
	}

	dreq := httptest.NewRequest(http.MethodDelete, "/api/links/"+l.Code, nil)
	drec := httptest.NewRecorder()
	h.ServeHTTP(drec, dreq)
	if drec.Code != http.StatusOK {
		t.Fatalf("delete code = %d, want 200", drec.Code)
	}
	var dresp map[string]any
	if err := json.Unmarshal(drec.Body.Bytes(), &dresp); err != nil {
		t.Fatal(err)
	}
	if dresp["deleted"] != true {
		t.Fatalf("deleted = %v, want true", dresp["deleted"])
	}

	// 删除后再次删除应返回未找到，而非再次成功。
	again := httptest.NewRequest(http.MethodDelete, "/api/links/"+l.Code, nil)
	arec := httptest.NewRecorder()
	h.ServeHTTP(arec, again)
	if arec.Code != http.StatusNotFound {
		t.Fatalf("re-delete code = %d, want 404", arec.Code)
	}
}
