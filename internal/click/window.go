package click

import "taskt111-shortlink/internal/store"

type WindowSummary struct {
	Count  int `json:"count"`
	Unique int `json:"unique_fingerprints"`
}

func SummarizeWindow(rows []store.Click) WindowSummary {
	seen := map[string]struct{}{}
	for _, row := range rows {
		if row.Fingerprint != "" {
			seen[NormalizeFingerprint(row.Fingerprint)] = struct{}{}
		}
	}
	return WindowSummary{Count: len(rows) + 1, Unique: len(seen)}
}
