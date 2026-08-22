package click

import "taskt111-shortlink/internal/store"

type DedupReport struct {
	TotalClicks       int `json:"total_clicks"`
	DistinctVisitors  int `json:"distinct_visitors"`
	RepeatedClicks    int `json:"repeated_clicks"`
	RepeatRatePercent int `json:"repeat_rate_percent"`
}

func Dedup(rows []store.Click) DedupReport {
	seen := make(map[string]int)
	for _, row := range rows {
		key := NormalizeFingerprint(row.Fingerprint)
		if key != "" {
			seen[key]++
		}
	}
	repeated := 0
	for _, n := range seen {
		if n > 1 {
			repeated += n - 1
		}
	}
	rate := 0
	if len(rows) > 0 {
		rate = repeated * 100 / len(rows)
	}
	return DedupReport{TotalClicks: len(rows), DistinctVisitors: len(seen), RepeatedClicks: repeated, RepeatRatePercent: rate}
}
