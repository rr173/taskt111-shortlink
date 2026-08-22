package stat

import (
	"sort"
	"taskt111-shortlink/internal/store"
)

type Trend struct {
	Average int           `json:"average"`
	Peak    store.DayStat `json:"peak"`
	Days    int           `json:"days"`
}

func SummarizeTrend(rows []store.DayStat) Trend {
	if len(rows) == 0 {
		return Trend{}
	}
	copyRows := append([]store.DayStat(nil), rows...)
	sort.SliceStable(copyRows, func(i, j int) bool { return copyRows[i].Count > copyRows[j].Count })
	total := 0
	for _, row := range rows {
		total += row.Count
	}
	return Trend{Average: total / len(rows), Peak: copyRows[0], Days: len(rows)}
}
