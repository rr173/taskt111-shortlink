// Package click 负责访问点击的采集与去重。
package click

import (
	"context"
	"errors"
	"time"

	"taskt111-shortlink/internal/store"
)

// 领域错误。click 不能反向依赖 link（link 已依赖 click 的聚合类型），
// 因此这里独立定义语义对等的哨兵错误，由 httpapi 层统一映射为 HTTP 状态码。
var (
	// ErrNotFound 表示被记录点击的短链不存在，必须拒绝写入以免产生孤立点击。
	ErrNotFound = errors.New("link not found")
	// ErrLimitReached 表示短链已达点击上限。
	ErrLimitReached = errors.New("click limit reached")
)

// Service 聚合点击相关操作。
type Service struct {
	store *store.Store
}

// New 构造点击服务。
func New(s *store.Store) *Service { return &Service{store: s} }

// Record 写入一条点击记录，返回落库后的完整记录。
// fingerprint 用于同一浏览器在短时间内重复刷新时的去重。
// 未知短链（code 不存在）会被明确拒绝，绝不向 clicks 表写入孤立记录。
func (s *Service) Record(ctx context.Context, code, referer, ua, ip, fingerprint string) (store.Click, error) {
	l, err := s.store.GetLinkByCode(ctx, code)
	if err != nil {
		return store.Click{}, err
	}
	// 未知短链必须被明确拒绝，否则会向 clicks 表写入无法关联到任何链接的
	// 孤立记录，既污染统计，又会让一致性校验报告 orphan_clicks 失真。
	if l.Code == "" {
		return store.Click{}, ErrNotFound
	}
	if l.MaxClicks > 0 {
		n, err := s.store.CountClicks(ctx, code)
		if err != nil {
			return store.Click{}, err
		}
		if n >= l.MaxClicks {
			return store.Click{}, ErrLimitReached
		}
	}
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
