package store

import (
	"context"
	"fmt"
)

type DomainStat struct {
	Host   string `json:"host"`
	Links  int    `json:"links"`
	Clicks int    `json:"clicks"`
}

// DomainStats supports abuse review and traffic concentration checks without
// loading every link into application memory.
func (s *Store) DomainStats(ctx context.Context, limit int) ([]DomainStat, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT substr(target_url, instr(target_url, '//')+2), COUNT(*),
		       COALESCE((SELECT COUNT(*) FROM clicks c WHERE c.code IN
		          (SELECT code FROM links l2 WHERE l2.target_url LIKE '%' || substr(l1.target_url, instr(l1.target_url, '//')+2) || '%')),0)
		FROM links l1 GROUP BY substr(target_url, instr(target_url, '//')+2)
		ORDER BY COUNT(*) DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("domain stats: %w", err)
	}
	defer rows.Close()
	out := make([]DomainStat, 0)
	for rows.Next() {
		var row DomainStat
		if err := rows.Scan(&row.Host, &row.Links, &row.Clicks); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
