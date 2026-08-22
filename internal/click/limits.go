package click

func NormalizeLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 500 {
		return 500
	}
	return limit
}

func LimitWindow(rows []int, limit int) []int {
	limit = NormalizeLimit(limit)
	if len(rows) <= limit {
		return append([]int(nil), rows...)
	}
	return append([]int(nil), rows[len(rows)-limit:]...)
}
