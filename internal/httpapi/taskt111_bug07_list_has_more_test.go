package httpapi

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
)

func TestListReportsHasMoreAfterTruncation(t *testing.T) {
    h := newServer(t)
    for _, target := range []string{"https://example.com/a", "https://example.com/b"} {
        rec := httptest.NewRecorder()
        h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/links", strings.NewReader(`{"target_url":"`+target+`"}`)))
        if rec.Code != http.StatusCreated { t.Fatalf("create status = %d", rec.Code) }
    }
    rec := httptest.NewRecorder()
    h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/links?limit=1", nil))
    if rec.Code != http.StatusOK { t.Fatalf("list status = %d", rec.Code) }
    var body struct { Links []map[string]any `json:"links"`; HasMore bool `json:"has_more"` }
    if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil { t.Fatal(err) }
    if len(body.Links) != 1 || !body.HasMore { t.Fatalf("page = %+v", body) }
}
