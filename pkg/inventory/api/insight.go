package api

type ListInsightResultsRequest struct {
	DescriptionFilter *string  `json:"descriptionFilter"`
	Labels            []string `json:"labels"`
	SourceIDs         []string `json:"sourceIDs"`
	ExecutedAt        *int64   `json:"executedAt"`
}

type ListInsightResultsResponse struct {
	Results []InsightResult `json:"results"`
}

type InsightResult struct {
	SmartQueryID     uint   `json:"smartQueryID"`
	Query            string `json:"query"`
	Category         string `json:"category"`
	Provider         string `json:"provider"`
	SourceID         string `json:"sourceID"`
	Description      string `json:"description"`
	ExecutedAt       int64  `json:"executedAt"`
	Result           int64  `json:"result"`
	LastDayValue     *int64 `json:"lastDayValue"`
	LastWeekValue    *int64 `json:"lastWeekValue"`
	LastMonthValue   *int64 `json:"lastMonthValue"`
	LastQuarterValue *int64 `json:"lastQuarterValue"`
	LastYearValue    *int64 `json:"lastYearValue"`
}
