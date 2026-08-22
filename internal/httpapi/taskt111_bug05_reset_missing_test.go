package httpapi

import ("net/http"; "net/http/httptest"; "testing")

func TestResetMissingLinkReturnsNotFound(t *testing.T) {
    h := newServer(t); rec := httptest.NewRecorder()
    h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/links/missing/reset", nil))
    if rec.Code != http.StatusNotFound { t.Fatalf("reset status = %d, want 404", rec.Code) }
}
