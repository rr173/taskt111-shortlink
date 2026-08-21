package stat

import "taskt111-shortlink/internal/store"

type Bucket struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// Buckets groups daily observations into stable low/medium/high bands. It is
// used by quota and traffic reviews where a percentage is less useful than a
// consistent operational classification.
func Buckets(rows []store.DayStat) []Bucket {
	if len(rows) == 0 {
		return []Bucket{{Name: "low", Count: 0}, {Name: "medium", Count: 0}, {Name: "high", Count: 0}}
	}
	peak := 0
	for _, row := range rows {
		if row.Count > peak {
			peak = row.Count
		}
	}
	low, high := peak/3, peak*2/3
	out := []Bucket{{Name: "low"}, {Name: "medium"}, {Name: "high"}}
	for _, row := range rows {
		switch {
		case row.Count <= low:
			out[0].Count++
		case row.Count <= high:
			out[1].Count++
		default:
			out[2].Count++
		}
	}
	return out
}
