package stat

import (
	"taskt111-shortlink/internal/store"
	"time"
)

func FillMissingDays(rows []store.DayStat, start, end time.Time) []store.DayStat {
	counts := make(map[string]int, len(rows))
	for _, row := range rows {
		counts[row.Day] = row.Count
	}
	out := make([]store.DayStat, 0)
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		out = append(out, store.DayStat{Day: key, Count: counts[key]})
	}
	return out
}
