// Package link 实现短链的核心领域逻辑：创建、解析（过期/点击上限判定）、
// 批量创建与状态计算。它依赖 store 完成持久化，自身不持有任何外部状态。
package link

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"taskt111-shortlink/internal/idgen"
	"taskt111-shortlink/internal/store"
)

// 领域错误。
var (
	ErrNotFound     = errors.New("link not found")
	ErrExpired      = errors.New("link expired")
	ErrLimitReached = errors.New("click limit reached")
	ErrInvalidURL   = errors.New("invalid target url")
	ErrEmptyCode    = errors.New("custom code is empty")
)

// Service 聚合短链领域操作。
type Service struct {
	store *store.Store
}

// New 构造短链服务。
func New(s *store.Store) *Service { return &Service{store: s} }

// CreateReq 是创建短链的输入。
type CreateReq struct {
	TargetURL   string
	Owner       string
	Description string
	CustomCode  string
	ExpiresAt   int64 // unix 毫秒，0 表示不过期
	MaxClicks   int   // 0 表示不限
}

// Create 创建一条短链。未指定 CustomCode 时自动生成唯一短码；
// 指定 CustomCode 时要求全局唯一。
func (s *Service) Create(ctx context.Context, req CreateReq) (store.Link, error) {
	if err := ValidateCreate(req); err != nil {
		return store.Link{}, err
	}
	req.Owner = NormalizeOwner(req.Owner)
	code := req.CustomCode
	code = idgen.CanonicalCode(code)
	if code == "" {
		c, err := idgen.UniqueCode(6, 8, func(c string) bool {
			l, e := s.store.GetLinkByCode(ctx, c)
			if e != nil {
				return true
			}
			return l.Code != ""
		})
		if err != nil {
			return store.Link{}, fmt.Errorf("generate code: %w", err)
		}
		code = c
	} else {
		existing, err := s.store.GetLinkByCode(ctx, code)
		if err != nil {
			return store.Link{}, err
		}
		if existing.Code != "" {
			return store.Link{}, fmt.Errorf("code %q already exists", code)
		}
	}
	l, err := s.store.InsertLink(ctx, store.Link{
		Code:        code,
		TargetURL:   req.TargetURL,
		Owner:       req.Owner,
		Description: req.Description,
		ExpiresAt:   req.ExpiresAt,
		MaxClicks:   req.MaxClicks,
		CustomAlias: req.CustomCode != "",
	})
	if err != nil {
		return store.Link{}, err
	}
	return l, nil
}

// BulkCreate 批量创建短链。任一子项失败则立即返回错误。
func (s *Service) BulkCreate(ctx context.Context, reqs []CreateReq) ([]store.Link, error) {
	if _, err := PlanBatch(reqs); err != nil {
		return nil, err
	}
	out := make([]store.Link, 0, len(reqs))
	for _, r := range reqs {
		l, err := s.Create(ctx, r)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, nil
}

// Resolve 解析短码并返回可跳转的链接；不存在、过期或达点击上限时返回对应错误。
func (s *Service) Resolve(ctx context.Context, code string) (store.Link, error) {
	l, err := s.store.GetLinkByCode(ctx, code)
	if err != nil {
		return store.Link{}, err
	}
	if l.Code == "" {
		return store.Link{}, ErrNotFound
	}
	now := time.Now().UnixMilli()
	if l.ExpiresAt > 0 && now >= l.ExpiresAt {
		return store.Link{}, ErrExpired
	}
	if l.MaxClicks > 0 {
		n, err := s.store.CountClicks(ctx, code)
		if err != nil {
			return store.Link{}, err
		}
		if n >= l.MaxClicks {
			return store.Link{}, ErrLimitReached
		}
	}
	return l, nil
}

// LinkStatus 描述一条链接的当前可跳转状态。
type LinkStatus struct {
	Code      string `json:"code"`
	Active    bool   `json:"active"`
	Reason    string `json:"reason"`
	Remaining int    `json:"remaining"` // 剩余可点击次数；不限时为 -1
}

// Status 计算链接的实时状态。
func (s *Service) Status(ctx context.Context, code string) (LinkStatus, error) {
	l, err := s.store.GetLinkByCode(ctx, code)
	if err != nil {
		return LinkStatus{}, err
	}
	if l.Code == "" {
		return LinkStatus{}, ErrNotFound
	}
	st := LinkStatus{Code: code, Active: true, Remaining: -1}
	now := time.Now().UnixMilli()
	if l.ExpiresAt > 0 && now >= l.ExpiresAt {
		st.Active = false
		st.Reason = "expired"
	}
	if l.MaxClicks > 0 {
		n, err := s.store.CountClicks(ctx, code)
		if err != nil {
			return LinkStatus{}, err
		}
		st.Remaining = l.MaxClicks - n
		if n >= l.MaxClicks {
			st.Active = false
			if st.Reason == "" {
				st.Reason = "limit"
			}
		}
	}
	return st, nil
}

// Update 更新链接的目标地址、描述、过期时间与最大点击数。
func (s *Service) Update(ctx context.Context, code, targetURL, description string, expiresAt int64, maxClicks int) error {
	if targetURL != "" {
		if err := validURL(targetURL); err != nil {
			return err
		}
	}
	return s.store.UpdateLink(ctx, code, targetURL, description, expiresAt, maxClicks)
}

// Delete 删除链接及其点击记录。
func (s *Service) Delete(ctx context.Context, code string) error {
	l, err := s.store.GetLinkByCode(ctx, code)
	if err != nil {
		return err
	}
	if l.Code == "" {
		return ErrNotFound
	}
	return s.store.DeleteLink(ctx, code)
}

// List 分页列出链接。
func (s *Service) List(ctx context.Context, owner string, limit, offset int) ([]store.Link, error) {
	rows, _, err := s.Page(ctx, owner, limit, offset)
	return rows, err
}

// Search 模糊搜索链接。
func (s *Service) Search(ctx context.Context, q string, limit int) ([]store.Link, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.store.SearchLinks(ctx, q, limit)
}

func validURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return ErrInvalidURL
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ErrInvalidURL
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrInvalidURL
	}
	if u.Host == "" {
		return ErrInvalidURL
	}
	return nil
}
