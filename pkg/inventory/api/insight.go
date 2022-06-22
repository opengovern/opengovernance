package api

type ListInsightResultsRequest struct {
	DescriptionFilter *string  `json:"descriptionFilter"`
	Labels            []string `json:"labels"`
}

type ListInsightResultsResponse struct {
	Results []InsightResult `json:"results"`
}

type InsightResult struct {
	Query      string `json:"query"`
	ExecutedAt int64  `json:"executedAt"`
	Result     int    `json:"result"`
}
