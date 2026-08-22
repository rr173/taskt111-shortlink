// Package click 负责访问点击的采集与去重。
package click

import (
	"context"
	"errors"
	"time"

	"taskt111-shortlink/internal/store"
)

// 领域错误：短链不存在、已过期或已达访问上限。
// 与 internal/link 中的同名错误语义一致，httpapi 在映射 HTTP 状态时一并识别。
var (
	ErrNotFound     = errors.New("link not found")
	ErrExpired      = errors.New("link expired")
	ErrLimitReached = errors.New("click limit reached")
)

// Service 聚合点击相关操作。
type Service struct {
	store *store.Store
}

// New 构造点击服务。
func New(s *store.Store) *Service { return &Service{store: s} }

// Record 写入一条点击记录，返回落库后的完整记录。
// 短链不存在、已过期或已达访问上限时返回对应错误，且不写入记录，从而保证
// 数据库中的总记录数不超过该短链设置的访问上限。
// fingerprint 用于同一浏览器在短时间内重复刷新时的去重。
func (s *Service) Record(ctx context.Context, code, referer, ua, ip, fingerprint string) (store.Click, error) {
	l, err := s.store.GetLinkByCode(ctx, code)
	if err != nil {
		return store.Click{}, err
	}
	if l.Code == "" {
		return store.Click{}, ErrNotFound
	}
	if l.ExpiresAt > 0 {
		if now := time.Now().UnixMilli(); now >= l.ExpiresAt {
			return store.Click{}, ErrExpired
		}
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
