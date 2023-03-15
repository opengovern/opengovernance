package api

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

type InsightLabel struct {
	ID    uint   `json:"id"`
	Label string `json:"label"`
}

type InsightLink struct {
	ID   uint   `json:"id"`
	Text string `json:"linkText"`
	URI  string `json:"linkURI"`
}

type Insight struct {
	ID          uint           `json:"id"`
	PeerGroupId *uint          `json:"peerGroupId"`
	Query       string         `json:"query"`
	Category    string         `json:"category"`
	Provider    source.Type    `json:"provider"`
	ShortTitle  string         `json:"shortTitle"`
	LongTitle   string         `json:"longTitle"`
	Description string         `json:"description"`
	LogoURL     *string        `json:"logoURL"`
	Labels      []InsightLabel `json:"labels"`
	Links       []InsightLink  `json:"links"`
	Enabled     bool           `json:"enabled"`
}

type InsightPeerGroup struct {
	ID          uint           `json:"id"`
	Category    string         `json:"category"`
	Insights    []Insight      `json:"insights,omitempty"`
	ShortTitle  string         `json:"shortTitle"`
	LongTitle   string         `json:"longTitle"`
	Description string         `json:"description"`
	LogoURL     *string        `json:"logoURL"`
	Labels      []InsightLabel `json:"labels"`
	Links       []InsightLink  `json:"links"`
}

type ListInsightsRequest struct {
	DescriptionFilter string `json:"descriptionFilter"`
}

type CreateInsightRequest struct {
	Description string      `json:"description"`
	Query       string      `json:"query"`
	Provider    source.Type `json:"provider"`
	Category    string      `json:"category"`
	Internal    bool        `json:"internal"`
}
