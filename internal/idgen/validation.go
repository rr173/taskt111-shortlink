package idgen

func ValidCode(code string) bool {
	if len(code) < 2 || len(code) > 32 {
		return false
	}
	for _, r := range code {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}
