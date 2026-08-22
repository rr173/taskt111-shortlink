package httpapi

import "net/http"

func (h *Handler) domains(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.DomainStats(r.Context(), atoiDefault(r.URL.Query().Get("limit"), 100))
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"domains": rows, "count": len(rows)})
}
