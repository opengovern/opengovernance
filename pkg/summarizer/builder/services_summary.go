package builder

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"
	describe "gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
)

type serviceSummaryBuilder struct {
	summarizerJobID   uint
	connectionSummary map[string]es.ConnectionServiceSummary
	providerSummary   map[string]es.ProviderServiceSummary
}

func NewServiceSummaryBuilder(summarizerJobID uint) *serviceSummaryBuilder {
	return &serviceSummaryBuilder{
		summarizerJobID:   summarizerJobID,
		connectionSummary: make(map[string]es.ConnectionServiceSummary),
		providerSummary:   make(map[string]es.ProviderServiceSummary),
	}
}

func (b *serviceSummaryBuilder) Process(resource describe.LookupResource) {
	key := resource.SourceID + "-" + cloudservice.ServiceNameByResourceType(resource.ResourceType)
	if _, ok := b.connectionSummary[key]; !ok {
		b.connectionSummary[key] = es.ConnectionServiceSummary{
			ScheduleJobID:    resource.ScheduleJobID,
			SourceID:         resource.SourceID,
			SourceJobID:      resource.SourceJobID,
			ServiceName:      cloudservice.ServiceNameByResourceType(resource.ResourceType),
			ResourceType:     resource.ResourceType,
			SourceType:       resource.SourceType,
			DescribedAt:      resource.CreatedAt,
			ResourceCount:    0,
			LastDayCount:     nil,
			LastWeekCount:    nil,
			LastQuarterCount: nil,
			LastYearCount:    nil,
			ReportType:       es.ServiceSummary,
		}
	}

	v := b.connectionSummary[key]
	v.ResourceCount++
	b.connectionSummary[key] = v

	key = string(resource.SourceType) + "-" + cloudservice.ServiceNameByResourceType(resource.ResourceType)
	if _, ok := b.providerSummary[key]; !ok {
		b.providerSummary[key] = es.ProviderServiceSummary{
			ScheduleJobID:    resource.ScheduleJobID,
			ServiceName:      cloudservice.ServiceNameByResourceType(resource.ResourceType),
			ResourceType:     resource.ResourceType,
			SourceType:       resource.SourceType,
			DescribedAt:      resource.CreatedAt,
			ResourceCount:    0,
			LastDayCount:     nil,
			LastWeekCount:    nil,
			LastQuarterCount: nil,
			LastYearCount:    nil,
			ReportType:       es.ServiceProviderSummary,
		}
	}

	v2 := b.providerSummary[key]
	v2.ResourceCount++
	b.providerSummary[key] = v2
}

func (b *serviceSummaryBuilder) PopulateHistory() error {
	return nil
}

func (b *serviceSummaryBuilder) Build() []kafka.Doc {
	var docs []kafka.Doc
	for _, v := range b.connectionSummary {
		docs = append(docs, v)
	}
	for _, v := range b.providerSummary {
		docs = append(docs, v)
	}
	return docs
}
