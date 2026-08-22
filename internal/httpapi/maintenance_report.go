package httpapi

import "net/http"

func (h *Handler) consistency(w http.ResponseWriter, r *http.Request) {
	report, err := h.store.Consistency(r.Context())
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, report)
}

func (h *Handler) validateData(w http.ResponseWriter, r *http.Request) {
	report, err := h.link.ValidatePersistedData(r.Context())
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, report)
}

func (h *Handler) cleanupExpired(w http.ResponseWriter, r *http.Request) {
	report, err := h.link.CleanupExpired(r.Context(), atoiDefault(r.URL.Query().Get("limit"), 100))
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, report)
}
