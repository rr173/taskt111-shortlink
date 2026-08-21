package idgen

import "strings"

func CanonicalCode(code string) string { return strings.TrimSpace(code) }

func LooksGenerated(code string) bool {
	if len(code) < 6 || len(code) > 8 {
		return false
	}
	return ValidCode(code)
}
