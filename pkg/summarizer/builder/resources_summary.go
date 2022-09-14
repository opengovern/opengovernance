package builder

import (
	describe "gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
)

type resourceSummaryBuilder struct {
	summarizerJobID   uint
	connectionSummary map[string]es.ConnectionResourcesSummary
}

func NewResourceSummaryBuilder(summarizerJobID uint) *resourceSummaryBuilder {
	return &resourceSummaryBuilder{
		summarizerJobID:   summarizerJobID,
		connectionSummary: make(map[string]es.ConnectionResourcesSummary),
	}
}

func (b *resourceSummaryBuilder) Process(resource describe.LookupResource) {
	if _, ok := b.connectionSummary[resource.SourceID]; !ok {
		b.connectionSummary[resource.SourceID] = es.ConnectionResourcesSummary{
			ScheduleJobID: resource.ScheduleJobID,
			SourceID:      resource.SourceID,
			SourceType:    resource.SourceType,
			SourceJobID:   resource.SourceJobID,
			DescribedAt:   resource.CreatedAt,
			ResourceCount: 0,
			ReportType:    es.ResourceSummary,
		}
	}

	v := b.connectionSummary[resource.SourceID]
	v.ResourceCount++
	b.connectionSummary[resource.SourceID] = v
}

func (b *resourceSummaryBuilder) PopulateHistory() error {
	return nil
}

func (b *resourceSummaryBuilder) Build() []kafka.Doc {
	var docs []kafka.Doc
	for _, v := range b.connectionSummary {
		docs = append(docs, v)
	}
	return docs
}
