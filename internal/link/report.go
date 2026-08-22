package link

import "context"

// OwnerReport is a persisted owner-level view used by quota and cleanup jobs.
type OwnerReport struct {
	Owner       string `json:"owner"`
	Links       int    `json:"links"`
	Clicks      int    `json:"clicks"`
	ActiveLinks int    `json:"active_links"`
}

func (s *Service) OwnerReport(ctx context.Context, owner string) (OwnerReport, error) {
	summary, err := s.store.OwnerSummary(ctx, owner)
	if err != nil {
		return OwnerReport{}, err
	}
	return OwnerReport{Owner: owner, Links: summary.Links, Clicks: summary.Clicks, ActiveLinks: summary.ActiveLinks}, nil
}
