// Package click 负责访问点击的采集与去重。
package click

import (
	"context"
	"time"

	"taskt111-shortlink/internal/store"
)

// Service 聚合点击相关操作。
type Service struct {
	store *store.Store
}

// New 构造点击服务。
func New(s *store.Store) *Service { return &Service{store: s} }

// Record 写入一条点击记录，返回落库后的完整记录。
// fingerprint 用于同一浏览器在短时间内重复刷新时的去重。
func (s *Service) Record(ctx context.Context, code, referer, ua, ip, fingerprint string) (store.Click, error) {
	return s.store.InsertClick(ctx, store.Click{
		Code:        code,
		Referer:     referer,
		UserAgent:   ua,
		IP:          ip,
		Fingerprint: NormalizeFingerprint(fingerprint),
		ClickedAt:   time.Now().UnixMilli(),
	})
}

// Recent 返回最近的点击记录。
func (s *Service) Recent(ctx context.Context, limit int) ([]store.Click, error) {
	limit = NormalizeLimit(limit)
	return s.store.RecentClicks(ctx, limit)
}
