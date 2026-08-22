package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type OwnerActivity struct {
	Owner       string `json:"owner"`
	Links       int    `json:"links"`
	Clicks      int    `json:"clicks"`
	ActiveLinks int    `json:"active_links"`
	LastClick   int64  `json:"last_click"`
}

type FingerprintStat struct {
	Fingerprint string `json:"fingerprint"`
	Clicks      int    `json:"clicks"`
	FirstSeen   int64  `json:"first_seen"`
	LastSeen    int64  `json:"last_seen"`
}

// OwnerActivityReport is calculated from links and click events, not cached
// counters, so it remains correct after a process restart or a restore.
func (s *Store) OwnerActivityReport(ctx context.Context, owner string) (OwnerActivity, error) {
	var out OwnerActivity
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*),
		       COALESCE(SUM(CASE WHEN expires_at=0 OR expires_at>? THEN 1 ELSE 0 END),0)
		FROM links WHERE owner=?`, time.Now().UnixMilli(), owner).
		Scan(&out.Links, &out.ActiveLinks); err != nil {
		return out, fmt.Errorf("owner link activity: %w", err)
	}
	var last sql.NullInt64
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*), MAX(c.clicked_at)
		FROM clicks c JOIN links l ON l.code=c.code WHERE l.owner=?`, owner).
		Scan(&out.Clicks, &last); err != nil {
		return out, fmt.Errorf("owner click activity: %w", err)
	}
	if last.Valid {
		out.LastClick = last.Int64
	}
	return out, nil
}

func (s *Store) FingerprintStats(ctx context.Context, code string, limit int) ([]FingerprintStat, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT fingerprint, COUNT(*), MIN(clicked_at), MAX(clicked_at)
		FROM clicks WHERE code=? AND fingerprint<>''
		GROUP BY fingerprint ORDER BY COUNT(*) DESC, fingerprint LIMIT ?`, code, limit)
	if err != nil {
		return nil, fmt.Errorf("fingerprint stats: %w", err)
	}
	defer rows.Close()
	out := make([]FingerprintStat, 0)
	for rows.Next() {
		var row FingerprintStat
		if err := rows.Scan(&row.Fingerprint, &row.Clicks, &row.FirstSeen, &row.LastSeen); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) ExpiredLinks(ctx context.Context, limit int) ([]Link, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, code, target_url, owner, description, created_at, expires_at, max_clicks, custom_alias
		FROM links WHERE expires_at>0 AND expires_at<=? ORDER BY expires_at, id LIMIT ?`, time.Now().UnixMilli(), limit)
	if err != nil {
		return nil, fmt.Errorf("expired links: %w", err)
	}
	defer rows.Close()
	out := make([]Link, 0)
	for rows.Next() {
		var l Link
		var alias int
		if err := rows.Scan(&l.ID, &l.Code, &l.TargetURL, &l.Owner, &l.Description, &l.CreatedAt, &l.ExpiresAt, &l.MaxClicks, &alias); err != nil {
			return nil, err
		}
		l.CustomAlias = alias != 0
		out = append(out, l)
	}
	return out, rows.Err()
}
