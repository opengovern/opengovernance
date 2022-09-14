package builder

import (
	describe "gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
)

type trendSummaryBuilder struct {
	summarizerJobID   uint
	connectionSummary map[string]es.ConnectionTrendSummary
	providerSummary   map[source.Type]es.ProviderTrendSummary
}

func NewTrendSummaryBuilder(summarizerJobID uint) *trendSummaryBuilder {
	return &trendSummaryBuilder{
		summarizerJobID:   summarizerJobID,
		connectionSummary: make(map[string]es.ConnectionTrendSummary),
		providerSummary:   make(map[source.Type]es.ProviderTrendSummary),
	}
}

func (b *trendSummaryBuilder) Process(resource describe.LookupResource) {
	if _, ok := b.connectionSummary[resource.SourceID]; !ok {
		b.connectionSummary[resource.SourceID] = es.ConnectionTrendSummary{
			ScheduleJobID: resource.ScheduleJobID,
			SourceID:      resource.SourceID,
			SourceType:    resource.SourceType,
			SourceJobID:   resource.SourceJobID,
			DescribedAt:   resource.CreatedAt,
			ReportType:    es.TrendConnectionSummary,
			ResourceCount: 0,
		}
	}
	v := b.connectionSummary[resource.SourceID]
	v.ResourceCount++
	b.connectionSummary[resource.SourceID] = v

	if _, ok := b.providerSummary[resource.SourceType]; !ok {
		b.providerSummary[resource.SourceType] = es.ProviderTrendSummary{
			ScheduleJobID: resource.ScheduleJobID,
			SourceType:    resource.SourceType,
			DescribedAt:   resource.CreatedAt,
			ReportType:    es.TrendProviderSummary,
			ResourceCount: 0,
		}
	}
	v2 := b.providerSummary[resource.SourceType]
	v2.ResourceCount++
	b.providerSummary[resource.SourceType] = v2
}

func (b *trendSummaryBuilder) PopulateHistory() error {
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
	return docs
}
