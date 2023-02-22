package resourcebuilder

import (
	describe "gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
)

type trendSummaryBuilder struct {
	client                        keibi.Client
	summarizerJobID               uint
	connectionSummary             map[string]es.ConnectionTrendSummary
	providerSummary               map[source.Type]es.ProviderTrendSummary
	connectionResourceTypeSummary map[string]es.ConnectionResourceTypeTrendSummary
	providerResourceTypeSummary   map[string]es.ProviderResourceTypeTrendSummary
}

func NewTrendSummaryBuilder(client keibi.Client, summarizerJobID uint) *trendSummaryBuilder {
	return &trendSummaryBuilder{
		client:                        client,
		summarizerJobID:               summarizerJobID,
		connectionSummary:             make(map[string]es.ConnectionTrendSummary),
		providerSummary:               make(map[source.Type]es.ProviderTrendSummary),
		connectionResourceTypeSummary: make(map[string]es.ConnectionResourceTypeTrendSummary),
		providerResourceTypeSummary:   make(map[string]es.ProviderResourceTypeTrendSummary),
	}
}

func (b *trendSummaryBuilder) Process(resource describe.LookupResource) {
	if _, ok := b.connectionSummary[resource.SourceID]; !ok {
		b.connectionSummary[resource.SourceID] = es.ConnectionTrendSummary{
			SummarizeJobID: b.summarizerJobID,
			ScheduleJobID:  resource.ScheduleJobID,
			SourceID:       resource.SourceID,
			SourceType:     resource.SourceType,
			SourceJobID:    resource.SourceJobID,
			DescribedAt:    resource.CreatedAt,
			ReportType:     es.TrendConnectionSummary,
			ResourceCount:  0,
		}
	}
	v := b.connectionSummary[resource.SourceID]
	v.ResourceCount++
	b.connectionSummary[resource.SourceID] = v

	if _, ok := b.providerSummary[resource.SourceType]; !ok {
		b.providerSummary[resource.SourceType] = es.ProviderTrendSummary{
			SummarizeJobID: b.summarizerJobID,
			ScheduleJobID:  resource.ScheduleJobID,
			SourceType:     resource.SourceType,
			DescribedAt:    resource.CreatedAt,
			ReportType:     es.TrendProviderSummary,
			ResourceCount:  0,
		}
	}
	v2 := b.providerSummary[resource.SourceType]
	v2.ResourceCount++
	b.providerSummary[resource.SourceType] = v2

	key := resource.SourceID + "_" + resource.ResourceType
	if _, ok := b.connectionResourceTypeSummary[key]; !ok {
		b.connectionResourceTypeSummary[key] = es.ConnectionResourceTypeTrendSummary{
			SummarizeJobID: b.summarizerJobID,
			ScheduleJobID:  resource.ScheduleJobID,
			SourceID:       resource.SourceID,
			SourceType:     resource.SourceType,
			SourceJobID:    resource.SourceJobID,
			DescribedAt:    resource.CreatedAt,
			ResourceType:   resource.ResourceType,
			ResourceCount:  0,
			ReportType:     es.ResourceTypeTrendConnectionSummary,
		}
	}
	v3 := b.connectionResourceTypeSummary[key]
	v3.ResourceCount++
	b.connectionResourceTypeSummary[key] = v3

	key = resource.SourceType.String() + "_" + resource.ResourceType
	if _, ok := b.providerResourceTypeSummary[key]; !ok {
		b.providerResourceTypeSummary[key] = es.ProviderResourceTypeTrendSummary{
			SummarizeJobID: b.summarizerJobID,
			ScheduleJobID:  resource.ScheduleJobID,
			SourceType:     resource.SourceType,
			DescribedAt:    resource.CreatedAt,
			ResourceType:   resource.ResourceType,
			ResourceCount:  0,
			ReportType:     es.ResourceTypeTrendProviderSummary,
		}
	}
	v4 := b.providerResourceTypeSummary[key]
	v4.ResourceCount++
	b.providerResourceTypeSummary[key] = v4
}

func (b *trendSummaryBuilder) PopulateHistory(lastDayJobID, lastWeekJobID, lastQuarterJobID, lastYearJobID uint) error {
	return nil
}

func (b *trendSummaryBuilder) Build() []kafka.Doc {
	var docs []kafka.Doc
	for _, v := range b.connectionSummary {
		docs = append(docs, v)
	}
	for _, v := range b.providerSummary {
		docs = append(docs, v)
	}
	for _, v := range b.connectionResourceTypeSummary {
		docs = append(docs, v)
	}
	for _, v := range b.providerResourceTypeSummary {
		docs = append(docs, v)
	}
	return docs
}

func (b *trendSummaryBuilder) Cleanup(summarizeJobID uint) error {
	return nil
}
