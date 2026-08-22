package httpapi

import (
	"net/http"
	"taskt111-shortlink/internal/click"
)

func (h *Handler) ownerReport(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")
	report, err := h.link.OwnerReport(r.Context(), owner)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, report)
}

func (h *Handler) clickQuality(w http.ResponseWriter, r *http.Request) {
	quality, err := h.store.ClickQuality(r.Context(), r.PathValue("code"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, click.QualityFrom(quality))
}

func (h *Handler) clickWindow(w http.ResponseWriter, r *http.Request) {
	rows, err := h.clk.Recent(r.Context(), atoiDefault(r.URL.Query().Get("limit"), 20))
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"summary": click.SummarizeWindow(rows), "sessions": click.Sessions(rows)})
}
