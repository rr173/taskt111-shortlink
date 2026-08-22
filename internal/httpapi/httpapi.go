// Package httpapi 暴露短链服务的 HTTP 接口，使用标准库 net/http 的
// 模式匹配路由（Go 1.22+），不引入第三方 Web 框架。
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"taskt111-shortlink/internal/click"
	"taskt111-shortlink/internal/link"
	"taskt111-shortlink/internal/stat"
	"taskt111-shortlink/internal/store"
)

// Handler 聚合所有 HTTP 处理。
type Handler struct {
	store *store.Store
	link  *link.Service
	clk   *click.Service
	stat  *stat.Service
}

// NewHandler 构造 HTTP Handler。
func NewHandler(s *store.Store) *Handler {
	return &Handler{
		store: s,
		link:  link.New(s),
		clk:   click.New(s),
		stat:  stat.New(s),
	}
}

// Routes 返回注册好全部路由的 http.Handler。
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/links", h.create)
	mux.HandleFunc("POST /api/links/bulk", h.bulk)
	mux.HandleFunc("GET /api/links", h.list)
	mux.HandleFunc("GET /api/links/search", h.search)
	mux.HandleFunc("GET /api/links/{code}/info", h.info)
	mux.HandleFunc("GET /api/links/{code}/exists", h.exists)
	mux.HandleFunc("GET /api/links/{code}/stats", h.statsTotal)
	mux.HandleFunc("GET /api/links/{code}/stats/daily", h.statsDaily)
	mux.HandleFunc("GET /api/links/{code}/stats/referers", h.statsReferers)
	mux.HandleFunc("GET /api/links/{code}/activity", h.activity)
	mux.HandleFunc("GET /api/links/{code}/quality", h.clickQuality)
	mux.HandleFunc("GET /api/links/{code}/trend", h.trend)
	mux.HandleFunc("GET /api/links/{code}/status", h.status)
	mux.HandleFunc("PUT /api/links/{code}", h.update)
	mux.HandleFunc("DELETE /api/links/{code}", h.delete)
	mux.HandleFunc("POST /api/links/{code}/click", h.recordClick)
	mux.HandleFunc("POST /api/links/{code}/reset", h.reset)
	mux.HandleFunc("POST /api/links/{code}/expire", h.expire)
	mux.HandleFunc("GET /api/stats/top", h.top)
	mux.HandleFunc("GET /api/stats/recent", h.recent)
	mux.HandleFunc("GET /api/health", h.health)
	mux.HandleFunc("GET /api/health/ready", h.ready)
	mux.HandleFunc("GET /api/metrics", h.metrics)
	mux.HandleFunc("GET /api/admin/links/expired", h.expired)
	mux.HandleFunc("GET /api/admin/consistency", h.consistency)
	mux.HandleFunc("GET /api/admin/validate", h.validateData)
	mux.HandleFunc("GET /api/admin/quota", h.quota)
	mux.HandleFunc("GET /api/admin/domains", h.domains)
	mux.HandleFunc("POST /api/admin/links/cleanup", h.cleanupExpired)
	mux.HandleFunc("GET /api/reports/owner", h.ownerReport)
	mux.HandleFunc("GET /api/reports/click-window", h.clickWindow)
	mux.HandleFunc("POST /api/admin/clicks/purge", h.purgeClicks)
	mux.HandleFunc("GET /{code}", h.redirect)
	return mux
}

// ---- 创建 / 批量 ----

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TargetURL   string `json:"target_url"`
		Owner       string `json:"owner"`
		Description string `json:"description"`
		CustomCode  string `json:"custom_code"`
		ExpiresAt   int64  `json:"expires_at"`
		MaxClicks   int    `json:"max_clicks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeJSON(w, http.StatusBadRequest, errResp("invalid json"))
		return
	}
	l, err := h.link.Create(r.Context(), link.CreateReq{
		TargetURL:   body.TargetURL,
		Owner:       body.Owner,
		Description: body.Description,
		CustomCode:  body.CustomCode,
		ExpiresAt:   body.ExpiresAt,
		MaxClicks:   body.MaxClicks,
	})
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, l)
}

func (h *Handler) bulk(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Links []struct {
			TargetURL   string `json:"target_url"`
			Owner       string `json:"owner"`
			Description string `json:"description"`
			CustomCode  string `json:"custom_code"`
			ExpiresAt   int64  `json:"expires_at"`
			MaxClicks   int    `json:"max_clicks"`
		} `json:"links"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeJSON(w, http.StatusBadRequest, errResp("invalid json"))
		return
	}
	reqs := make([]link.CreateReq, 0, len(body.Links))
	for _, it := range body.Links {
		reqs = append(reqs, link.CreateReq{
			TargetURL:   it.TargetURL,
			Owner:       it.Owner,
			Description: it.Description,
			CustomCode:  it.CustomCode,
			ExpiresAt:   it.ExpiresAt,
			MaxClicks:   it.MaxClicks,
		})
	}
	created, err := h.link.BulkCreate(r.Context(), reqs)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, map[string]any{"created": len(created), "links": created})
}

// ---- 列表 / 搜索 ----

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	limit := atoiDefault(r.URL.Query().Get("limit"), 20)
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}
	offset := atoiDefault(r.URL.Query().Get("offset"), 0)
	if offset < 0 {
		offset = 0
	}
	owner := r.URL.Query().Get("owner")
	links, err := h.store.ListLinks(r.Context(), owner, limit+1, offset)
	if err != nil {
		h.writeError(w, err)
		return
	}
	if len(links) > limit {
		hasMore := true
		links = links[:limit]
		h.writeJSON(w, http.StatusOK, map[string]any{
			"links":    links,
			"has_more": hasMore,
			"limit":    limit,
			"offset":   offset,
		})
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"links":    links,
		"has_more": false,
		"limit":    limit,
		"offset":   offset,
	})
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limit := atoiDefault(r.URL.Query().Get("limit"), 20)
	links, err := h.link.Search(r.Context(), q, limit)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"links": links, "count": len(links)})
}

// ---- 单链接查询 / 状态 ----

func (h *Handler) info(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	l, err := h.store.GetLinkByCode(r.Context(), code)
	if err != nil {
		h.writeError(w, err)
		return
	}
	if l.Code == "" {
		h.writeJSON(w, http.StatusNotFound, errResp("link not found"))
		return
	}
	h.writeJSON(w, http.StatusOK, l)
}

func (h *Handler) exists(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	l, err := h.store.GetLinkByCode(r.Context(), code)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"exists": l.Code != ""})
}

func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	st, err := h.link.Status(r.Context(), code)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, st)
}

// ---- 更新 / 删除 ----

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	var body struct {
		TargetURL   string `json:"target_url"`
		Description string `json:"description"`
		ExpiresAt   int64  `json:"expires_at"`
		MaxClicks   int    `json:"max_clicks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeJSON(w, http.StatusBadRequest, errResp("invalid json"))
		return
	}
	if err := h.link.Update(r.Context(), code, body.TargetURL, body.Description, body.ExpiresAt, body.MaxClicks); err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"code": code, "updated": true})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if err := h.link.Delete(r.Context(), code); err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"code": code, "deleted": true})
}

// ---- 点击 ----

func (h *Handler) recordClick(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	c, err := h.clk.Record(r.Context(), code, r.Header.Get("Referer"), r.Header.Get("User-Agent"), clientIP(r), fingerprint(r))
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, c)
}

func (h *Handler) reset(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	l, err := h.store.GetLinkByCode(r.Context(), code)
	if err != nil {
		h.writeError(w, err)
		return
	}
	if l.Code == "" {
		h.writeError(w, link.ErrNotFound)
		return
	}
	if err := h.store.ResetClicks(r.Context(), code); err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"code": code, "reset": true})
}

// ---- 过期 ----

func (h *Handler) expire(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	l, err := h.store.GetLinkByCode(r.Context(), code)
	if err != nil {
		h.writeError(w, err)
		return
	}
	if l.Code == "" {
		h.writeError(w, link.ErrNotFound)
		return
	}
	if err := h.link.Update(r.Context(), code, "", "", time.Now().UnixMilli(), 0); err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"code": code, "expired": true})
}

// ---- 统计 ----

func (h *Handler) statsTotal(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	n, err := h.stat.TotalClicks(r.Context(), code)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"code": code, "total_clicks": n})
}

func (h *Handler) statsDaily(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		h.writeJSON(w, http.StatusBadRequest, errResp("from and to are required (YYYY-MM-DD)"))
		return
	}
	rows, err := h.stat.DailyBreakdown(r.Context(), code, from, to)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"code": code, "from": from, "to": to, "days": rows})
}

func (h *Handler) statsReferers(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	limit := atoiDefault(r.URL.Query().Get("limit"), 20)
	rows, err := h.stat.Referers(r.Context(), code, limit)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"code": code, "referers": rows})
}

func (h *Handler) top(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")
	limit := atoiDefault(r.URL.Query().Get("limit"), 10)
	rows, err := h.stat.TopLinks(r.Context(), owner, limit)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"top": rows})
}

func (h *Handler) recent(w http.ResponseWriter, r *http.Request) {
	limit := atoiDefault(r.URL.Query().Get("limit"), 20)
	rows, err := h.clk.Recent(r.Context(), limit)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"recent": rows})
}

// ---- 健康检查 ----

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "time": time.Now().UnixMilli()})
}

func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	if err := h.store.DBHealth(); err != nil {
		h.writeJSON(w, http.StatusServiceUnavailable, errResp(err.Error()))
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ready"})
}

func (h *Handler) metrics(w http.ResponseWriter, r *http.Request) {
	total, err := h.store.CountLinks(r.Context(), "")
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"links": total})
}

// ---- 公开重定向 ----

func (h *Handler) redirect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" || code == "api" || code == "favicon.ico" {
		http.NotFound(w, r)
		return
	}
	l, err := h.link.Resolve(r.Context(), code)
	if err != nil {
		h.writeError(w, err)
		return
	}
	w.Header().Set("Location", l.TargetURL)
	w.WriteHeader(http.StatusFound)
	_, _ = h.clk.Record(r.Context(), code, r.Header.Get("Referer"), r.Header.Get("User-Agent"), clientIP(r), fingerprint(r))
}

// ---- 工具 ----

func (h *Handler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, link.ErrOwnerQuotaExceeded):
		h.writeJSON(w, http.StatusConflict, errResp(err.Error()))
	case errors.Is(err, link.ErrNotFound):
		h.writeJSON(w, http.StatusNotFound, errResp("link not found"))
	case errors.Is(err, link.ErrExpired), errors.Is(err, link.ErrLimitReached):
		h.writeJSON(w, http.StatusGone, errResp(err.Error()))
	default:
		h.writeJSON(w, http.StatusInternalServerError, errResp(err.Error()))
	}
}

func errResp(msg string) map[string]any {
	return map[string]any{"error": msg}
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.Split(fwd, ",")[0]
	}
	return r.RemoteAddr
}

func fingerprint(r *http.Request) string {
	ua := r.Header.Get("User-Agent")
	if len(ua) > 32 {
		ua = ua[:32]
	}
	return clientIP(r) + "|" + ua
}
