package httpapi

import "net/http"

func (h *Handler) quota(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")
	limit := atoiDefault(r.URL.Query().Get("limit"), 100)
	quota, err := h.link.CheckOwnerQuota(r.Context(), owner, limit)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, quota)
}
