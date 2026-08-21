package httpapi

import (
	"net/http"
	"taskt111-shortlink/internal/click"
	"taskt111-shortlink/internal/stat"
)

func (h *Handler) activity(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	view, err := h.link.ActivityReport(r.Context(), code, atoiDefault(r.URL.Query().Get("limit"), 20))
	if err != nil {
		h.writeError(w, err)
		return
	}
	rows, err := h.clk.Recent(r.Context(), 100)
	if err != nil {
		h.writeError(w, err)
		return
	}
	view.Window = click.SummarizeWindow(rows)
	h.writeJSON(w, http.StatusOK, view)
}

func (h *Handler) expired(w http.ResponseWriter, r *http.Request) {
	rows, err := h.link.Expired(r.Context(), atoiDefault(r.URL.Query().Get("limit"), 100))
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"links": rows, "count": len(rows)})
}

func (h *Handler) trend(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	rows, err := h.stat.DailyBreakdown(r.Context(), code, from, to)
	if err != nil {
		h.writeError(w, err)
		return
	}
	values := make([]int, 0, len(rows))
	for _, row := range rows {
		values = append(values, row.Count)
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"trend": stat.SummarizeTrend(rows), "buckets": stat.Buckets(rows), "anomalies": stat.DetectAnomalies(rows), "percentiles": stat.CountPercentiles(values), "days": rows})
}
