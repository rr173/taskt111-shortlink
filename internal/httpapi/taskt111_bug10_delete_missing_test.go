package httpapi

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestDeleteMissingLinkReturnsNotFoundThroughHTTP(t *testing.T) {
    h := newServer(t)
    rec := httptest.NewRecorder()
    h.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/links/missing", nil))
    if rec.Code != http.StatusNotFound { t.Fatalf("delete status = %d, want 404", rec.Code) }
}
