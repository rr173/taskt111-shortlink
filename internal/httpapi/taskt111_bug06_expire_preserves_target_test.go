package httpapi

import ("encoding/json"; "net/http"; "net/http/httptest"; "strings"; "testing"; "taskt111-shortlink/internal/store")

func TestExpireKeepsTargetURL(t *testing.T) {
    h := newServer(t); create := httptest.NewRecorder()
    h.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/links", strings.NewReader(`{"target_url":"https://example.com/keep"}`)))
    var l store.Link; if err := json.Unmarshal(create.Body.Bytes(), &l); err != nil { t.Fatal(err) }
    expire := httptest.NewRecorder(); h.ServeHTTP(expire, httptest.NewRequest(http.MethodPost, "/api/links/"+l.Code+"/expire", nil))
    if expire.Code != http.StatusOK { t.Fatalf("expire status = %d", expire.Code) }
    info := httptest.NewRecorder(); h.ServeHTTP(info, httptest.NewRequest(http.MethodGet, "/api/links/"+l.Code+"/info", nil))
    var got store.Link; if err := json.Unmarshal(info.Body.Bytes(), &got); err != nil { t.Fatal(err) }
    if got.TargetURL != "https://example.com/keep" { t.Fatalf("target after expire = %q", got.TargetURL) }
}
