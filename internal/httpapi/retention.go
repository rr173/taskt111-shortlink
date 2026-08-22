package httpapi

import (
	"net/http"
	"strconv"
	"time"
)

func (h *Handler) purgeClicks(w http.ResponseWriter, r *http.Request) {
	ms, err := strconv.ParseInt(r.URL.Query().Get("before_ms"), 10, 64)
	if err != nil || ms <= 0 {
		h.writeJSON(w, http.StatusBadRequest, errResp("before_ms required"))
		return
	}
	n, err := h.store.DeleteClicksBefore(r.Context(), time.UnixMilli(ms))
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"deleted": n})
}
