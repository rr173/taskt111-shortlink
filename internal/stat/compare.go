package stat

import "taskt111-shortlink/internal/store"

type DayDelta struct {
	Day      string `json:"day"`
	Current  int    `json:"current"`
	Previous int    `json:"previous"`
	Delta    int    `json:"delta"`
}

func CompareDays(current, previous []store.DayStat) []DayDelta {
	pm := map[string]int{}
	for _, row := range previous {
		pm[row.Day] = row.Count
	}
	out := make([]DayDelta, 0, len(current))
	for _, row := range current {
		before := pm[row.Day]
		out = append(out, DayDelta{Day: row.Day, Current: row.Count, Previous: before, Delta: row.Count - before})
	}
	return out
}
