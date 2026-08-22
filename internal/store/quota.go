package store

import "context"

type Quota struct {
	Owner     string `json:"owner"`
	Used      int    `json:"used"`
	Clicks    int    `json:"clicks"`
	Limit     int    `json:"limit"`
	Remaining int    `json:"remaining"`
}

func (s *Store) OwnerQuota(ctx context.Context, owner string, limit int) (Quota, error) {
	activity, err := s.OwnerActivityReport(ctx, owner)
	if err != nil {
		return Quota{}, err
	}
	remaining := limit - activity.Links
	if remaining < 0 {
		remaining = 0
	}
	return Quota{Owner: owner, Used: activity.Links, Clicks: activity.Clicks, Limit: limit, Remaining: remaining}, nil
}
