package link

import (
	"context"
	"errors"
	"taskt111-shortlink/internal/store"
)

var ErrOwnerQuotaExceeded = errors.New("owner link quota exceeded")

func (s *Service) CheckOwnerQuota(ctx context.Context, owner string, limit int) (store.Quota, error) {
	quota, err := s.store.OwnerQuota(ctx, owner, limit)
	if err != nil {
		return store.Quota{}, err
	}
	if quota.Remaining == 0 && quota.Used >= quota.Limit {
		return quota, ErrOwnerQuotaExceeded
	}
	return quota, nil
}
