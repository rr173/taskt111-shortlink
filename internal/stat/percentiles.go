package stat

import "sort"

type Percentiles struct {
	P50 int `json:"p50"`
	P90 int `json:"p90"`
	P99 int `json:"p99"`
}

func ClampPercent(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func NormalizeValues(values []int) []int {
	out := make([]int, len(values))
	for i, value := range values {
		if value < 0 {
			value = 0
		}
		out[i] = value
	}
	return out
}

func CountPercentiles(values []int) Percentiles {
	if len(values) == 0 {
		return Percentiles{}
	}
	ordered := NormalizeValues(values)
	sort.Ints(ordered)
	pick := func(p int) int {
		idx := (len(ordered)*ClampPercent(p) + 99) / 100
		if idx < 1 {
			idx = 1
		}
		return ordered[idx-1]
	}
	return Percentiles{P50: pick(50), P90: pick(90), P99: pick(99)}
}
