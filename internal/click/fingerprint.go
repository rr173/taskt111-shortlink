package click

import "strings"

// NormalizeFingerprint keeps deduplication keys stable across harmless header
// whitespace differences while retaining the caller's identity boundary.
func NormalizeFingerprint(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}
