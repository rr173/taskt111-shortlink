package link

import (
	"context"
	"fmt"
	"taskt111-shortlink/internal/idgen"
	"taskt111-shortlink/internal/store"
)

type ValidationSummary struct {
	Checked int      `json:"checked"`
	Valid   int      `json:"valid"`
	Invalid []string `json:"invalid"`
}

// ValidatePersistedData audits all link rows after migration or restore. It
// does not modify the database and returns concrete codes for operator repair.
func (s *Service) ValidatePersistedData(ctx context.Context) (ValidationSummary, error) {
	rows, err := s.store.ListLinks(ctx, "", 10000, 0)
	if err != nil {
		return ValidationSummary{}, err
	}
	out := ValidationSummary{Checked: len(rows), Invalid: []string{}}
	for _, row := range rows {
		if err := validatePersistedLink(row); err != nil {
			out.Invalid = append(out.Invalid, fmt.Sprintf("%s: %v", row.Code, err))
			continue
		}
		out.Valid++
	}
	return out, nil
}

func validatePersistedLink(row store.Link) error {
	if row.Code == "" || !idgen.ValidCode(row.Code) {
		return fmt.Errorf("invalid code")
	}
	if err := validURL(row.TargetURL); err != nil {
		return err
	}
	if row.ExpiresAt < 0 || row.MaxClicks < 0 {
		return fmt.Errorf("negative retention limit")
	}
	return nil
}
