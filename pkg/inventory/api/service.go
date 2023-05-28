package api

import "github.com/kaytu-io/kaytu-util/pkg/source"

type Service struct {
	Connector    source.Type         `json:"connector"`
	ServiceName  string              `json:"service_name"`
	ServiceLabel string              `json:"service_label"`
	Tags         map[string][]string `json:"tags,omitempty"`
	LogoURI      *string             `json:"logo_uri,omitempty"`
	Cost         *float64            `json:"count,omitempty"`
}

type ListServiceMetricsResponse struct {
	TotalCost     float64   `json:"total_cost"`
	TotalServices int       `json:"total_services"`
	Services      []Service `json:"services"`
}
