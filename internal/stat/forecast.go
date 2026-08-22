package stat

import "taskt111-shortlink/internal/store"

type Forecast struct {
	CurrentAverage  int `json:"current_average"`
	PreviousAverage int `json:"previous_average"`
	DeltaPercent    int `json:"delta_percent"`
}

func CompareForecast(current, previous []store.DayStat) Forecast {
	currentTrend, previousTrend := SummarizeTrend(current), SummarizeTrend(previous)
	percent := 0
	if previousTrend.Average != 0 {
		percent = (currentTrend.Average - previousTrend.Average) * 100 / previousTrend.Average
	}
	return Forecast{CurrentAverage: currentTrend.Average, PreviousAverage: previousTrend.Average, DeltaPercent: percent}
}
