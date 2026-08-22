package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"taskt111-shortlink/internal/store"
)

// TestRecordClickRejectsAfterLimit 验证通过 HTTP 接口：达到访问上限后，
// 后续点击返回 410 Gone，且数据库总记录数不超过上限。
func TestRecordClickRejectsAfterLimit(t *testing.T) {
	mux := newServer(t)

	// 创建一个 max_clicks=2 的短链
	req := httptest.NewRequest(http.MethodPost, "/api/links", strings.NewReader(`{"target_url":"https://x.com","max_clicks":2}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create = %d", rec.Code)
	}
	var l store.Link
	if err := json.Unmarshal(rec.Body.Bytes(), &l); err != nil {
		t.Fatal(err)
	}

	// 两次点击应成功（201）
	for i := 0; i < 2; i++ {
		r := httptest.NewRequest(http.MethodPost, "/api/links/"+l.Code+"/click", strings.NewReader("{}"))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != http.StatusCreated {
			t.Fatalf("click %d = %d, want 201", i, w.Code)
		}
	}

	// 第三次点击应被拒绝（410 Gone）
	r := httptest.NewRequest(http.MethodPost, "/api/links/"+l.Code+"/click", strings.NewReader("{}"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusGone {
		t.Fatalf("third click = %d, want 410 Gone", w.Code)
	}

	// 继续尝试多次，仍被拒绝
	for i := 0; i < 3; i++ {
		r := httptest.NewRequest(http.MethodPost, "/api/links/"+l.Code+"/click", strings.NewReader("{}"))
		mux.ServeHTTP(httptest.NewRecorder(), r)
	}

	// 通过 stats 接口核对总数不超过上限
	statReq := httptest.NewRequest(http.MethodGet, "/api/links/"+l.Code+"/stats", nil)
	statRec := httptest.NewRecorder()
	mux.ServeHTTP(statRec, statReq)
	if statRec.Code != http.StatusOK {
		t.Fatalf("stats = %d", statRec.Code)
	}
	var s struct {
		TotalClicks int `json:"total_clicks"`
	}
	if err := json.Unmarshal(statRec.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	if s.TotalClicks != 2 {
		t.Fatalf("total_clicks = %d, want 2 (not exceeding max)", s.TotalClicks)
	}
}
