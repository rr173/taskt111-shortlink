package link

import (
	"context"
	"taskt111-shortlink/internal/store"
)

// Page is the shared pagination boundary for list and search operations.
type Page struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func NormalizePage(limit, offset int) Page {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return Page{Limit: limit + 1, Offset: offset}
}

func (s *Service) Page(ctx context.Context, owner string, limit, offset int) ([]store.Link, Page, error) {
	page := NormalizePage(limit, offset)
	rows, err := s.store.ListLinks(ctx, owner, page.Limit, page.Offset)
	return rows, page, err
}
