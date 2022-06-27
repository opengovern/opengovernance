package api

type ListInsightResultsRequest struct {
	DescriptionFilter *string  `json:"descriptionFilter"`
	Labels            []string `json:"labels"`
}

type ListInsightResultsResponse struct {
	Results []InsightResult `json:"results"`
}

type InsightResult struct {
	Query            string `json:"query"`
	ExecutedAt       int64  `json:"executedAt"`
	Result           int64  `json:"result"`
	LastDayValue     int64  `json:"last_day_value"`
	LastWeekValue    int64  `json:"last_week_value"`
	LastQuarterValue int64  `json:"last_quarter_value"`
	LastYearValue    int64  `json:"last_year_value"`
}
