package link

import (
	"errors"
	"strings"

	"taskt111-shortlink/internal/idgen"
)

// ValidateCreate applies business limits before a request reaches SQLite.
func ValidateCreate(req CreateReq) error {
	if err := validURL(req.TargetURL); err != nil {
		return err
	}
	if req.ExpiresAt < 0 || req.MaxClicks < 0 {
		return errors.New("expiry and max_clicks must be non-negative")
	}
	if req.CustomCode != "" && !idgen.ValidCode(req.CustomCode) {
		return errors.New("custom code contains unsupported characters")
	}
	return nil
}

func NormalizeOwner(owner string) string { return strings.TrimSpace(owner) }
