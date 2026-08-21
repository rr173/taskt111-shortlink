package stat

import "taskt111-shortlink/internal/store"

type Anomaly struct {
	Day    string `json:"day"`
	Count  int    `json:"count"`
	Reason string `json:"reason"`
}

// DetectAnomalies flags days that are materially above the observed average.
// The rule is intentionally explainable and is only a derived view over the
// durable click facts; it never changes redirect behavior.
func DetectAnomalies(rows []store.DayStat) []Anomaly {
	trend := SummarizeTrend(rows)
	if trend.Average == 0 {
		return []Anomaly{}
	}
	threshold := trend.Average * 3
	out := make([]Anomaly, 0)
	for _, row := range rows {
		if row.Count > threshold {
			out = append(out, Anomaly{Day: row.Day, Count: row.Count, Reason: "above_three_day_average"})
		}
	}
	return out
}
