package link

import (
	"context"
	"errors"

	"taskt111-shortlink/internal/store"
)

type CleanupReport struct {
	Candidates int `json:"candidates"`
	Deleted    int `json:"deleted"`
}

// CleanupExpired removes expired links only after the caller explicitly asks
// for maintenance. Click history remains untouched so historical reports keep
// their evidence and orphan detection can be performed separately.
//
// 删除时若链接已被并发删除（DeleteLink 返回 store.ErrLinkNotFound），
// 视作已清理并继续处理后续候选，避免单个缺失项中断整批维护。
func (s *Service) CleanupExpired(ctx context.Context, limit int) (CleanupReport, error) {
	rows, err := s.store.ExpiredLinks(ctx, limit)
	if err != nil {
		return CleanupReport{}, err
	}
	out := CleanupReport{Candidates: len(rows)}
	for _, row := range rows {
		if err := s.store.DeleteLink(ctx, row.Code); err != nil {
			if errors.Is(err, store.ErrLinkNotFound) {
				continue
			}
			return out, err
		}
		out.Deleted++
	}
	return out, nil
}
