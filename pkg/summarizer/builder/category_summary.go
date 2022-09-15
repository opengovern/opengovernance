package builder

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"
	describe "gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
)

type categorySummaryBuilder struct {
	summarizerJobID   uint
	connectionSummary map[string]es.ConnectionCategorySummary
	providerSummary   map[string]es.ProviderCategorySummary
}

func NewCategorySummaryBuilder(summarizerJobID uint) *categorySummaryBuilder {
	return &categorySummaryBuilder{
		summarizerJobID:   summarizerJobID,
		connectionSummary: make(map[string]es.ConnectionCategorySummary),
		providerSummary:   make(map[string]es.ProviderCategorySummary),
	}
}

func (b *categorySummaryBuilder) Process(resource describe.LookupResource) {
	key := resource.SourceID + "-" + cloudservice.CategoryByResourceType(resource.ResourceType)
	if _, ok := b.connectionSummary[key]; !ok {
		b.connectionSummary[key] = es.ConnectionCategorySummary{
			ScheduleJobID:    resource.ScheduleJobID,
			SourceID:         resource.SourceID,
			SourceJobID:      resource.SourceJobID,
			CategoryName:     cloudservice.CategoryByResourceType(resource.ResourceType),
			ResourceType:     resource.ResourceType,
			SourceType:       resource.SourceType,
			DescribedAt:      resource.CreatedAt,
			ResourceCount:    0,
			LastDayCount:     nil,
			LastWeekCount:    nil,
			LastQuarterCount: nil,
			LastYearCount:    nil,
			ReportType:       es.CategorySummary,
		}
	}

	v := b.connectionSummary[key]
	v.ResourceCount++
	b.connectionSummary[key] = v

	key = string(resource.SourceType) + "-" + cloudservice.CategoryByResourceType(resource.ResourceType)
	if _, ok := b.providerSummary[key]; !ok {
		b.providerSummary[key] = es.ProviderCategorySummary{
			ScheduleJobID:    resource.ScheduleJobID,
			CategoryName:     cloudservice.CategoryByResourceType(resource.ResourceType),
			ResourceType:     resource.ResourceType,
			SourceType:       resource.SourceType,
			DescribedAt:      resource.CreatedAt,
			ResourceCount:    0,
			LastDayCount:     nil,
			LastWeekCount:    nil,
			LastQuarterCount: nil,
			LastYearCount:    nil,
			ReportType:       es.CategoryProviderSummary,
		}
	}

	v2 := b.providerSummary[key]
	v2.ResourceCount++
	b.providerSummary[key] = v2
}

func (b *categorySummaryBuilder) PopulateHistory() error {
	return nil
}

func (b *categorySummaryBuilder) Build() []kafka.Doc {
	var docs []kafka.Doc
	for _, v := range b.connectionSummary {
		docs = append(docs, v)
		h := v
		h.ReportType = h.ReportType + "History"
		docs = append(docs, h)
	}
	for _, v := range b.providerSummary {
		docs = append(docs, v)
		h := v
		h.ReportType = h.ReportType + "History"
		docs = append(docs, h)
	}
	return docs
}
