package api

type Insight struct {
	ID          uint     `json:"id"`
	Description string   `json:"description"`
	Query       string   `json:"query"`
	Labels      []string `json:"labels"`
}

type ListInsightsRequest struct {
	DescriptionFilter string   `json:"descriptionFilter"`
	Labels            []string `json:"labels"`
}

type CreateInsightRequest struct {
	Description string   `json:"description"`
	Query       string   `json:"query"`
	Labels      []string `json:"labels"`
}
