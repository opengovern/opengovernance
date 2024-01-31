package api

import (
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/es"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"time"
)

type GetSingleResourceFindingResponse struct {
	Resource        es.Resource    `json:"resource"`
	FindingEvents   []FindingEvent `json:"findingEvents"`
	ControlFindings []Finding      `json:"controls"`
}

type FindingEvent struct {
	ID                string            `json:"id" example:"8e0f8e7a1b1c4e6fb7e49c6af9d2b1c8"`
	FindingID         string            `json:"findingID"`
	ComplianceJobID   uint              `json:"complianceJobID"`
	ConformanceStatus ConformanceStatus `json:"conformanceStatus"`
	StateActive       bool              `json:"stateActive"`
	EvaluatedAt       time.Time         `json:"evaluatedAt"`
	Reason            string            `json:"reason"`

	BenchmarkID               string                `json:"benchmarkID" example:"azure_cis_v140"`
	ControlID                 string                `json:"controlID" example:"azure_cis_v140_7_5"`
	ConnectionID              string                `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	Connector                 source.Type           `json:"connector" example:"Azure"`
	Severity                  types.FindingSeverity `json:"severity" example:"low"`
	KaytuResourceID           string                `json:"kaytuResourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceID                string                `json:"resourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceType              string                `json:"resourceType" example:"Microsoft.Compute/virtualMachines"`
	ParentBenchmarkReferences []string              `json:"parentBenchmarkReferences"`

	SortKey []any `json:"sortKey"`
}

func GetAPIFindingEventFromESFindingEvent(findingEvent types.FindingEvent) FindingEvent {
	f := FindingEvent{
		ID:                findingEvent.EsID,
		FindingID:         findingEvent.FindingEsID,
		ComplianceJobID:   findingEvent.ComplianceJobID,
		ConformanceStatus: "",
		StateActive:       findingEvent.StateActive,
		EvaluatedAt:       time.UnixMilli(findingEvent.EvaluatedAt),
		Reason:            findingEvent.Reason,

		BenchmarkID:               findingEvent.BenchmarkID,
		ControlID:                 findingEvent.ControlID,
		ConnectionID:              findingEvent.ConnectionID,
		Connector:                 findingEvent.Connector,
		Severity:                  findingEvent.Severity,
		KaytuResourceID:           findingEvent.KaytuResourceID,
		ResourceID:                findingEvent.ResourceID,
		ResourceType:              findingEvent.ResourceType,
		ParentBenchmarkReferences: findingEvent.ParentBenchmarkReferences,
	}
	if findingEvent.ConformanceStatus.IsPassed() {
		f.ConformanceStatus = ConformanceStatusPassed
	} else {
		f.ConformanceStatus = ConformanceStatusFailed
	}

	return f
}

type FindingEventFilters struct {
	Connector         []source.Type           `json:"connector" example:"Azure"`
	ResourceType      []string                `json:"resourceType" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines"`
	ConnectionID      []string                `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	NotConnectionID   []string                `json:"notConnectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	BenchmarkID       []string                `json:"benchmarkID" example:"azure_cis_v140"`
	ControlID         []string                `json:"controlID" example:"azure_cis_v140_7_5"`
	Severity          []types.FindingSeverity `json:"severity" example:"low"`
	ConformanceStatus []ConformanceStatus     `json:"conformanceStatus" example:"alarm"`
	StateActive       []bool                  `json:"stateActive" example:"true"`
	FindingID         []string                `json:"findingID" example:"8e0f8e7a1b1c4e6fb7e49c6af9d2b1c8"`
	KaytuResourceID   []string                `json:"kaytuResourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	EvaluatedAt       struct {
		From *time.Time `json:"from" example:"2020-05-13T00:00:00Z"`
		To   *time.Time `json:"to" example:"2020-05-13T00:00:00Z"`
	} `json:"evaluatedAt"`
}

type FindingEventFiltersWithMetadata struct {
	Connector          []FilterWithMetadata `json:"connector"`
	BenchmarkID        []FilterWithMetadata `json:"benchmarkID"`
	ControlID          []FilterWithMetadata `json:"controlID"`
	ResourceTypeID     []FilterWithMetadata `json:"resourceTypeID"`
	ConnectionID       []FilterWithMetadata `json:"connectionID"`
	ResourceCollection []FilterWithMetadata `json:"resourceCollection"`
	Severity           []FilterWithMetadata `json:"severity"`
	ConformanceStatus  []FilterWithMetadata `json:"conformanceStatus"`
	StateActive        []FilterWithMetadata `json:"stateActive"`
}

type FindingEventsSort struct {
	Connector         *SortDirection `json:"connector"`
	KaytuResourceID   *SortDirection `json:"kaytuResourceID"`
	ResourceType      *SortDirection `json:"resourceType"`
	ConnectionID      *SortDirection `json:"connectionID"`
	BenchmarkID       *SortDirection `json:"benchmarkID"`
	ControlID         *SortDirection `json:"controlID"`
	Severity          *SortDirection `json:"severity"`
	ConformanceStatus *SortDirection `json:"conformanceStatus"`
	StateActive       *SortDirection `json:"stateActive"`
}

type GetFindingEventsRequest struct {
	Filters      FindingEventFilters `json:"filters"`
	Sort         []FindingEventsSort `json:"sort"`
	Limit        int                 `json:"limit" example:"100"`
	AfterSortKey []any               `json:"afterSortKey"`
}

type GetFindingEventsResponse struct {
	FindingEvents []FindingEvent `json:"findingEvents"`
	TotalCount    int64          `json:"totalCount" example:"100"`
}
