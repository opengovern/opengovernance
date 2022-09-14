package builder

import (
	describe "gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
)

type resourceTypeSummaryBuilder struct {
	summarizerJobID   uint
	connectionSummary map[string]es.ConnectionResourceTypeSummary
	providerSummary   map[string]es.ProviderResourceTypeSummary
}

func NewResourceTypeSummaryBuilder(summarizerJobID uint) *resourceTypeSummaryBuilder {
	return &resourceTypeSummaryBuilder{
		summarizerJobID:   summarizerJobID,
		connectionSummary: make(map[string]es.ConnectionResourceTypeSummary),
		providerSummary:   make(map[string]es.ProviderResourceTypeSummary),
	}
}

func (b *resourceTypeSummaryBuilder) Process(resource describe.LookupResource) {
	key := string(resource.SourceID) + "-" + resource.ResourceType
	if _, ok := b.connectionSummary[key]; !ok {
		b.connectionSummary[key] = es.ConnectionResourceTypeSummary{
			ScheduleJobID:    resource.ScheduleJobID,
			SourceID:         resource.SourceID,
			SourceJobID:      resource.SourceJobID,
			ResourceType:     resource.ResourceType,
			SourceType:       resource.SourceType,
			ResourceCount:    0,
			LastDayCount:     nil,
			LastWeekCount:    nil,
			LastQuarterCount: nil,
			LastYearCount:    nil,
			ReportType:       es.ResourceTypeSummary,
		}
	}

	v := b.connectionSummary[key]
	v.ResourceCount++
	b.connectionSummary[key] = v

	key = string(resource.SourceType) + "-" + resource.ResourceType
	if _, ok := b.providerSummary[key]; !ok {
		b.providerSummary[key] = es.ProviderResourceTypeSummary{
			ScheduleJobID:    resource.ScheduleJobID,
			ResourceType:     resource.ResourceType,
			SourceType:       resource.SourceType,
			ResourceCount:    0,
			LastDayCount:     nil,
			LastWeekCount:    nil,
			LastQuarterCount: nil,
			LastYearCount:    nil,
			ReportType:       es.ResourceTypeProviderSummary,
		}
	}

	v2 := b.providerSummary[key]
	v2.ResourceCount++
	b.providerSummary[key] = v2
}

func (b *resourceTypeSummaryBuilder) PopulateHistory() error {
	return nil
}

func (b *resourceTypeSummaryBuilder) Build() []kafka.Doc {
	var docs []kafka.Doc
	for _, v := range b.connectionSummary {
		docs = append(docs, v)
	}
	for _, v := range b.providerSummary {
		docs = append(docs, v)
	}
	return docs
}
