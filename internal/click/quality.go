package click

import "taskt111-shortlink/internal/store"

type Quality struct {
	Total           int `json:"total"`
	IdentityPercent int `json:"identity_percent"`
	ContextPercent  int `json:"context_percent"`
}

func QualityFrom(row store.ClickQuality) Quality {
	identity, context := 0, 0
	if row.Total > 0 {
		identity = row.WithFingerprint * 100 / row.Total
		context = row.WithReferer * 100 / row.Total
	}
	return Quality{Total: row.Total, IdentityPercent: identity, ContextPercent: context}
}
