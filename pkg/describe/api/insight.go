package api

import "gitlab.com/keibiengine/keibi-engine/pkg/source"

type Insight struct {
	ID           uint        `json:"id"`
	Description  string      `json:"description"`
	Query        string      `json:"query"`
	Provider     source.Type `json:"provider"`
	Category     string      `json:"category"`
	SmartQueryID uint        `json:"smartQueryID"`
}

type ListInsightsRequest struct {
	DescriptionFilter string `json:"descriptionFilter"`
}

type CreateInsightRequest struct {
	Description  string      `json:"description"`
	Query        string      `json:"query"`
	Provider     source.Type `json:"provider"`
	Category     string      `json:"category"`
	SmartQueryID uint        `json:"smartQueryID"`
	Internal     bool        `json:"internal"`
}
