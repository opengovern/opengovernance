package builder

import (
	describe "gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
)

type locationSummaryBuilder struct {
	summarizerJobID   uint
	connectionSummary map[string]es.ConnectionLocationSummary
	providerSummary   map[source.Type]es.ProviderLocationSummary
}

func NewLocationSummaryBuilder(summarizerJobID uint) *locationSummaryBuilder {
	return &locationSummaryBuilder{
		summarizerJobID:   summarizerJobID,
		connectionSummary: make(map[string]es.ConnectionLocationSummary),
		providerSummary:   make(map[source.Type]es.ProviderLocationSummary),
	}
}

func (b *locationSummaryBuilder) Process(resource describe.LookupResource) {
	if _, ok := b.connectionSummary[resource.SourceID]; !ok {
		b.connectionSummary[resource.SourceID] = es.ConnectionLocationSummary{
			ScheduleJobID:        resource.ScheduleJobID,
			SourceID:             resource.SourceID,
			SourceType:           resource.SourceType,
			SourceJobID:          resource.SourceJobID,
			LocationDistribution: map[string]int{},
			ReportType:           es.LocationConnectionSummary,
		}
	}

	v := b.connectionSummary[resource.SourceID]
	v.LocationDistribution[resource.Location]++
	b.connectionSummary[resource.SourceID] = v

	if _, ok := b.providerSummary[resource.SourceType]; !ok {
		b.providerSummary[resource.SourceType] = es.ProviderLocationSummary{
			ScheduleJobID:        resource.ScheduleJobID,
			SourceType:           resource.SourceType,
			LocationDistribution: map[string]int{},
			ReportType:           es.LocationProviderSummary,
		}
	}

	v2 := b.providerSummary[resource.SourceType]
	v2.LocationDistribution[resource.Location]++
	b.providerSummary[resource.SourceType] = v2
}

func (b *locationSummaryBuilder) PopulateHistory() error {
	return nil
}

func (b *locationSummaryBuilder) Build() []kafka.Doc {
	var docs []kafka.Doc
	for _, v := range b.connectionSummary {
		docs = append(docs, v)
	}
	for _, v := range b.providerSummary {
		docs = append(docs, v)
	}
	return docs
}
