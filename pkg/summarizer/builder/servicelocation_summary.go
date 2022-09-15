package builder

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"
	describe "gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
)

type serviceLocationSummaryBuilder struct {
	client            keibi.Client
	summarizerJobID   uint
	connectionSummary map[string]es.ConnectionServiceLocationSummary
}

func NewServiceLocationSummaryBuilder(client keibi.Client, summarizerJobID uint) *serviceLocationSummaryBuilder {
	return &serviceLocationSummaryBuilder{
		client:            client,
		summarizerJobID:   summarizerJobID,
		connectionSummary: make(map[string]es.ConnectionServiceLocationSummary),
	}
}

func (b *serviceLocationSummaryBuilder) Process(resource describe.LookupResource) {
	key := resource.SourceID + "-" + cloudservice.ServiceNameByResourceType(resource.ResourceType)
	if _, ok := b.connectionSummary[key]; !ok {
		b.connectionSummary[key] = es.ConnectionServiceLocationSummary{
			ScheduleJobID:        resource.ScheduleJobID,
			SourceID:             resource.SourceID,
			SourceType:           resource.SourceType,
			SourceJobID:          resource.SourceJobID,
			ServiceName:          cloudservice.ServiceNameByResourceType(resource.ResourceType),
			LocationDistribution: map[string]int{},
			ReportType:           es.ServiceLocationConnectionSummary,
		}
	}

	v := b.connectionSummary[key]
	v.LocationDistribution[resource.Location]++
	b.connectionSummary[key] = v
}

func (b *serviceLocationSummaryBuilder) PopulateHistory(lastDayJobID, lastWeekJobID, lastQuarterJobID, lastYearJobID uint) error {
	return nil
}

func (b *serviceLocationSummaryBuilder) Build() []kafka.Doc {
	var docs []kafka.Doc
	for _, v := range b.connectionSummary {
		docs = append(docs, v)
	}
	return docs
}
