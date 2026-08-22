package store

import "context"

type ClickQuality struct {
	Total           int `json:"total"`
	WithFingerprint int `json:"with_fingerprint"`
	WithReferer     int `json:"with_referer"`
	WithUserAgent   int `json:"with_user_agent"`
}

func (s *Store) ClickQuality(ctx context.Context, code string) (ClickQuality, error) {
	var out ClickQuality
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*),
		       COALESCE(SUM(CASE WHEN fingerprint<>'' THEN 1 ELSE 0 END),0),
		       COALESCE(SUM(CASE WHEN referer<>'' THEN 1 ELSE 0 END),0),
		       COALESCE(SUM(CASE WHEN user_agent<>'' THEN 1 ELSE 0 END),0)
		FROM clicks WHERE code=?`, code).
		Scan(&out.Total, &out.WithFingerprint, &out.WithReferer, &out.WithUserAgent)
	return out, err
}
