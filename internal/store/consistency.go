package store

import (
	"context"
	"fmt"
)

type ConsistencyReport struct {
	Links        int  `json:"links"`
	Clicks       int  `json:"clicks"`
	OrphanClicks int  `json:"orphan_clicks"`
	ExpiredLinks int  `json:"expired_links"`
	Healthy      bool `json:"healthy"`
}

// Consistency checks the relationships needed by redirect and statistics
// paths. It is intentionally read-only and can be run after restore.
func (s *Store) Consistency(ctx context.Context) (ConsistencyReport, error) {
	var out ConsistencyReport
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM links`).Scan(&out.Links); err != nil {
		return out, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM clicks`).Scan(&out.Clicks); err != nil {
		return out, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM clicks c LEFT JOIN links l ON l.code=c.code WHERE l.code IS NULL`).Scan(&out.OrphanClicks); err != nil {
		return out, fmt.Errorf("orphan clicks: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM links WHERE expires_at>0 AND expires_at<=strftime('%s','now')*1000`).Scan(&out.ExpiredLinks); err != nil {
		return out, err
	}
	out.Healthy = out.OrphanClicks == 0
	return out, nil
}
