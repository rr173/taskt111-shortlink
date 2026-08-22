package click

import (
	"sort"
	"taskt111-shortlink/internal/store"
)

type VisitorSession struct {
	Fingerprint string `json:"fingerprint"`
	Clicks      int    `json:"clicks"`
	Last        int64  `json:"last"`
}

func Sessions(rows []store.Click) []VisitorSession {
	byID := map[string]*VisitorSession{}
	for _, row := range rows {
		id := NormalizeFingerprint(row.Fingerprint)
		if id == "" {
			continue
		}
		item := byID[id]
		if item == nil {
			item = &VisitorSession{Fingerprint: id}
			byID[id] = item
		}
		item.Clicks++
		if row.ClickedAt > item.Last {
			item.Last = row.ClickedAt
		}
	}
	out := make([]VisitorSession, 0, len(byID))
	for _, item := range byID {
		out = append(out, *item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Clicks > out[j].Clicks })
	return out
}
