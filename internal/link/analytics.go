package link

import (
	"context"
	"taskt111-shortlink/internal/click"
	"taskt111-shortlink/internal/store"
)

type ActivityReport struct {
	Link          store.Link              `json:"link"`
	OwnerActivity store.OwnerActivity     `json:"owner_activity"`
	Fingerprints  []store.FingerprintStat `json:"fingerprints"`
	Window        click.WindowSummary     `json:"recent_window"`
}

func (s *Service) ActivityReport(ctx context.Context, code string, limit int) (ActivityReport, error) {
	l, err := s.store.GetLinkByCode(ctx, code)
	if err != nil {
		return ActivityReport{}, err
	}
	if l.Code == "" {
		return ActivityReport{}, ErrNotFound
	}
	activity, err := s.store.OwnerActivityReport(ctx, l.Owner)
	if err != nil {
		return ActivityReport{}, err
	}
	fps, err := s.store.FingerprintStats(ctx, code, limit)
	if err != nil {
		return ActivityReport{}, err
	}
	return ActivityReport{Link: l, OwnerActivity: activity, Fingerprints: fps}, nil
}

func (s *Service) Expired(ctx context.Context, limit int) ([]store.Link, error) {
	return s.store.ExpiredLinks(ctx, limit)
}
