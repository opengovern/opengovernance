package resourcebuilder

import (
	"fmt"
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/source"

	describe "github.com/kaytu-io/kaytu-engine/pkg/describe/es"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer/helpers"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer/query"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
)

type costSummaryBuilder struct {
	client                     kaytu.Client
	summarizerJobID            uint
	costsByService             map[string]es.ServiceCostSummary
	costsByServicePerConnector map[source.Type]map[int64]map[string]es.ServiceCostSummary
	costsByAccount             map[string]es.ConnectionCostSummary
	costsPerConnector          map[source.Type]map[int64]es.ConnectionCostSummary
}

type EBSCostDoc struct {
	Base es.ServiceCostSummary
	Desc helpers.EBSCostDescription
}

func NewCostSummaryBuilder(client kaytu.Client, summarizerJobID uint) *costSummaryBuilder {
	return &costSummaryBuilder{
		client:                     client,
		summarizerJobID:            summarizerJobID,
		costsByService:             make(map[string]es.ServiceCostSummary),
		costsByServicePerConnector: make(map[source.Type]map[int64]map[string]es.ServiceCostSummary),
		costsByAccount:             make(map[string]es.ConnectionCostSummary),
		costsPerConnector:          make(map[source.Type]map[int64]es.ConnectionCostSummary),
	}
}

func (b *costSummaryBuilder) Process(resource describe.LookupResource) {
	var fullResource *describe.Resource
	var err error
	costResourceType := es.GetCostResourceTypeFromString(resource.ResourceType)
	if costResourceType == es.CostResourceTypeNull {
		return
	}
	fullResource, err = query.GetResourceFromResourceLookup(b.client, resource)
	if err != nil {
		fmt.Printf("(costSummaryBuilder) - Error getting resource from lookup: %v", err)
		return
	}
	costSummary, key, err := costResourceType.GetCostSummaryAndKey(*fullResource, resource)
	if err != nil {
		fmt.Printf("(costSummaryBuilder) - Error getting service cost summary: %v", err)
		return
	}
	switch costSummary.(type) {
	case *es.ServiceCostSummary:
		serviceCostSummary := costSummary.(*es.ServiceCostSummary)
		serviceCostSummary.SummarizeJobID = b.summarizerJobID
		serviceCostSummary.SummarizeJobTime = time.Now().Unix()
		serviceCostSummary.Connector = resource.SourceType
		serviceCostSummary.SourceID = resource.SourceID
		serviceCostSummary.SourceJobID = resource.SourceJobID
		serviceCostSummary.ResourceType = resource.ResourceType
		costVal, _ := costResourceType.GetCostAndUnitFromResource(serviceCostSummary.Cost)
		serviceCostSummary.CostValue = costVal
		if _, ok := b.costsByService[key]; !ok {
			b.costsByService[key] = *serviceCostSummary
		}
		if serviceCostSummary.ReportType == es.CostServiceSummaryDaily {
			if _, ok := b.costsByServicePerConnector[resource.SourceType]; !ok {
				b.costsByServicePerConnector[resource.SourceType] = make(map[int64]map[string]es.ServiceCostSummary)
			}
			timeKey := (serviceCostSummary.PeriodEnd + serviceCostSummary.PeriodStart) / 2
			if _, ok := b.costsByServicePerConnector[resource.SourceType][timeKey]; !ok {
				b.costsByServicePerConnector[resource.SourceType][timeKey] = make(map[string]es.ServiceCostSummary)
			}
			if v, ok := b.costsByServicePerConnector[resource.SourceType][timeKey][key]; !ok {
				local := *serviceCostSummary
				local.SourceID = resource.SourceType.String()
				local.SourceJobID = 0
				local.Cost = nil
				local.ReportType = es.CostServiceConnectorSummaryDaily
				b.costsByServicePerConnector[resource.SourceType][timeKey][key] = local
			} else {
				v.CostValue += serviceCostSummary.CostValue
				b.costsByServicePerConnector[resource.SourceType][timeKey][key] = v
			}
		}

	case *es.ConnectionCostSummary:
		connectionCostSummary := costSummary.(*es.ConnectionCostSummary)
		connectionCostSummary.SummarizeJobID = b.summarizerJobID
		connectionCostSummary.SummarizeJobTime = time.Now().Unix()
		connectionCostSummary.SourceType = resource.SourceType
		connectionCostSummary.SourceID = resource.SourceID
		connectionCostSummary.SourceJobID = resource.SourceJobID
		connectionCostSummary.ResourceType = resource.ResourceType
		costVal, _ := costResourceType.GetCostAndUnitFromResource(connectionCostSummary.Cost)
		connectionCostSummary.CostValue = costVal
		if connectionCostSummary.ReportType == es.CostConnectionSummaryDaily {
			if _, ok := b.costsByAccount[key]; !ok {
				b.costsByAccount[key] = *connectionCostSummary
			}
			if _, ok := b.costsPerConnector[resource.SourceType]; !ok {
				b.costsPerConnector[resource.SourceType] = make(map[int64]es.ConnectionCostSummary)
			}
			timeKey := (connectionCostSummary.PeriodEnd + connectionCostSummary.PeriodStart) / 2
			if v, ok := b.costsPerConnector[resource.SourceType][timeKey]; !ok {
				local := *connectionCostSummary
				local.SourceID = resource.SourceType.String()
				local.AccountID = resource.SourceType.String()
				local.SourceJobID = 0
				local.Cost = nil
				local.ReportType = es.CostConnectorSummaryDaily
				b.costsPerConnector[resource.SourceType][timeKey] = local
			} else {
				v.CostValue += connectionCostSummary.CostValue
				b.costsPerConnector[resource.SourceType][timeKey] = v
			}
		}
	default:
		fmt.Printf("(costSummaryBuilder) - WARNING: Unknown cost summary type: %T:%v", costSummary, costSummary)
	}
}

func (b *costSummaryBuilder) Build() []kafka.Doc {
	var docs []kafka.Doc

	for _, v := range b.costsByAccount {
		docs = append(docs, v)
	}
	for _, v := range b.costsPerConnector {
		for _, v2 := range v {
			docs = append(docs, v2)
		}
	}

	for _, v := range b.costsByService {
		docs = append(docs, v)
	}
	for _, v := range b.costsByServicePerConnector {
		for _, v2 := range v {
			for _, v3 := range v2 {
				docs = append(docs, v3)
			}
		}
	}

	return docs
}

func (b *costSummaryBuilder) Cleanup(summarizeJobID uint) error {
	return nil
}
