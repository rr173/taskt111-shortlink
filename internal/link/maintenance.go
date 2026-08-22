package link

import (
	"context"
)

type CleanupReport struct {
	Candidates int `json:"candidates"`
	Deleted    int `json:"deleted"`
}

// CleanupExpired removes expired links only after the caller explicitly asks
// for maintenance. Click history remains untouched so historical reports keep
// their evidence and orphan detection can be performed separately.
func (s *Service) CleanupExpired(ctx context.Context, limit int) (CleanupReport, error) {
	rows, err := s.store.ExpiredLinks(ctx, limit)
	if err != nil {
		return CleanupReport{}, err
	}
	out := CleanupReport{Candidates: len(rows)}
	for _, row := range rows {
		if err := s.store.DeleteLink(ctx, row.Code); err != nil {
			return out, err
		}
		out.Deleted++
	}
	return out, nil
}
