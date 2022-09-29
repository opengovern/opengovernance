package types

type HistoricalCount struct {
	LastDayValue     int64 `json:"lastDayValue"`
	LastMonthValue   int64 `json:"lastMonthValue"`
	LastQuarterValue int64 `json:"lastQuarterValue"`
	LastYearValue    int64 `json:"lastYearValue"`
}
