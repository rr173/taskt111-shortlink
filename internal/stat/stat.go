// Package stat 实现访问统计聚合：按天明细、来源分布、热门排行与总量。
package stat

import (
	"context"
	"fmt"
	"time"

	"taskt111-shortlink/internal/store"
)

// Service 聚合统计相关操作。
type Service struct {
	store *store.Store
}

// New 构造统计服务。
func New(s *store.Store) *Service { return &Service{store: s} }

// DailyBreakdown 返回 [from, to] 闭区间内每一天的点击数，缺失的天以 0 填充，
// 结果按日期升序。这是一个「主数据 + 派生状态」必须保持一致的不变量：
// 输出天数必须等于区间天数。
func (s *Service) DailyBreakdown(ctx context.Context, code, from, to string) ([]store.DayStat, error) {
	start, err := time.Parse("2006-01-02", from)
	if err != nil {
		return nil, fmt.Errorf("parse from %q: %w", from, err)
	}
	end, err := time.Parse("2006-01-02", to)
	if err != nil {
		return nil, fmt.Errorf("parse to %q: %w", to, err)
	}
	if end.Before(start) {
		return nil, fmt.Errorf("from %q after to %q", from, to)
	}
	rows, err := s.store.DailyClicks(ctx, code, from, to)
	if err != nil {
		return nil, err
	}
	return FillMissingDays(rows, start, end), nil
}

// TopLinks 返回热门短码排行；ctx 被显式传递并在已取消时立即返回。
func (s *Service) TopLinks(ctx context.Context, owner string, limit int) ([]store.TopLink, error) {
	return s.store.TopLinks(ctx, owner, limit)
}

// Referers 返回按来源聚合的点击分布。
func (s *Service) Referers(ctx context.Context, code string, limit int) ([]store.RefererStat, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.store.RefererClicks(ctx, code, limit)
}

// TotalClicks 返回某短码的总点击数。
func (s *Service) TotalClicks(ctx context.Context, code string) (int, error) {
	return s.store.CountClicks(ctx, code)
}
