package api

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

type InsightTag struct {
	ID    uint   `json:"id"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

type InsightLink struct {
	ID   uint   `json:"id"`
	Text string `json:"linkText"`
	URI  string `json:"linkURI"`
}

type Insight struct {
	ID          uint          `json:"id"`
	PeerGroupId *uint         `json:"peerGroupId"`
	Query       Query         `json:"query"`
	Category    string        `json:"category"`
	Connector   source.Type   `json:"connector"`
	ShortTitle  string        `json:"shortTitle"`
	LongTitle   string        `json:"longTitle"`
	Description string        `json:"description"`
	LogoURL     *string       `json:"logoURL"`
	Tags        []InsightTag  `json:"labels"`
	Links       []InsightLink `json:"links"`
	Enabled     bool          `json:"enabled"`
	Internal    bool          `json:"internal"`
}

type InsightPeerGroup struct {
	ID          uint          `json:"id"`
	Category    string        `json:"category"`
	Insights    []Insight     `json:"insights,omitempty"`
	ShortTitle  string        `json:"shortTitle"`
	LongTitle   string        `json:"longTitle"`
	Description string        `json:"description"`
	LogoURL     *string       `json:"logoURL"`
	Tags        []InsightTag  `json:"labels"`
	Links       []InsightLink `json:"links"`
}
