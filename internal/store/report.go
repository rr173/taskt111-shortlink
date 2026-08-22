package store

import (
	"context"
	"fmt"
	"time"
)

type OwnerSummary struct {
	Links       int
	Clicks      int
	ActiveLinks int
}

func (s *Store) OwnerSummary(ctx context.Context, owner string) (OwnerSummary, error) {
	var out OwnerSummary
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM links WHERE owner=?`, owner).Scan(&out.Links); err != nil {
		return out, fmt.Errorf("count owner links: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM clicks c JOIN links l ON l.code=c.code WHERE l.owner=?`, owner).Scan(&out.Clicks); err != nil {
		return out, fmt.Errorf("count owner clicks: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM links WHERE owner=? AND (expires_at=0 OR expires_at>?)`, owner, time.Now().UnixMilli()).Scan(&out.ActiveLinks); err != nil {
		return out, fmt.Errorf("count active links: %w", err)
	}
	return out, nil
}
