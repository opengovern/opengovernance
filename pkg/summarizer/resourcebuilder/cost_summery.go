package resourcebuilder

import (
	"fmt"
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/kafka"

	describe "github.com/kaytu-io/kaytu-engine/pkg/describe/es"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer/helpers"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer/query"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
)

type costSummaryBuilder struct {
	client          keibi.Client
	summarizerJobID uint
	costsByService  map[string]es.ServiceCostSummary
	costsByAccount  map[string]es.ConnectionCostSummary
}

type EBSCostDoc struct {
	Base es.ServiceCostSummary
	Desc helpers.EBSCostDescription
}

func NewCostSummaryBuilder(client keibi.Client, summarizerJobID uint) *costSummaryBuilder {
	return &costSummaryBuilder{
		client:          client,
		summarizerJobID: summarizerJobID,
		costsByService:  make(map[string]es.ServiceCostSummary),
		costsByAccount:  make(map[string]es.ConnectionCostSummary),
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
		if _, ok := b.costsByAccount[key]; !ok {
			b.costsByAccount[key] = *connectionCostSummary
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
	for _, v := range b.costsByService {
		docs = append(docs, v)
	}

	return docs
}

func (b *costSummaryBuilder) Cleanup(summarizeJobID uint) error {
	return nil
}
