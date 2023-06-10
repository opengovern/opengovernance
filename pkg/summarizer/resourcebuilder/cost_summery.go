package resourcebuilder

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/kafka"

	ec2 "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	awsModel "github.com/kaytu-io/kaytu-aws-describer/aws/model"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	describe "gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/helpers"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/query"
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
		serviceCostSummary.SourceType = resource.SourceType
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

func (b *costSummaryBuilder) PopulateHistory(lastDayJobID, lastWeekJobID, lastQuarterJobID, lastYearJobID uint) error {
	return nil
}

func (b *costSummaryBuilder) Build() []kafka.Doc {
	var docs []kafka.Doc
	nowTime := time.Now().Truncate(24 * time.Hour).Unix()

	jsonCosts := helpers.GetEbsCosts()
	ebsCosts := make([]es.ServiceCostSummary, 0)
	ebsCostsRegionMap := make(map[string]EBSCostDoc)
	for _, v := range b.costsByAccount {
		docs = append(docs, v)
	}
	for _, v := range b.costsByService {
		switch v.ServiceName {
		case string(es.CostResourceTypeAWSEBSVolume):
			ebsCosts = append(ebsCosts, v)
		default:
			docs = append(docs, v)
		}
	}
	for _, v := range ebsCosts {
		costManifest := jsonCosts[strings.ToLower(*v.Region)]
		volDesc := v.Cost.(awsModel.EC2VolumeDescription)
		key := fmt.Sprintf("%s-%s", *v.Region, v.SourceID)
		if _, ok := ebsCostsRegionMap[key]; !ok {
			ebsCostsRegionMap[key] = EBSCostDoc{
				Base: v,
				Desc: helpers.EBSCostDescription{Region: *v.Region},
			}
		}
		ebsCost := ebsCostsRegionMap[key]
		size := 0
		if volDesc.Volume.Size != nil {
			size = int(*volDesc.Volume.Size)
		}
		Iops := 0
		if volDesc.Volume.Iops != nil {
			Iops = int(*volDesc.Volume.Iops)
		}
		throughput := 0
		if volDesc.Volume.Throughput != nil {
			throughput = int(*volDesc.Volume.Throughput)
		}
		switch volDesc.Volume.VolumeType {
		case ec2.VolumeTypeStandard:
			ebsCost.Desc.StandardSize += size
			ebsCost.Desc.StandardIOPS += Iops
		case ec2.VolumeTypeIo1:
			ebsCost.Desc.Io1Size += size
			ebsCost.Desc.Io1IOPS += Iops
		case ec2.VolumeTypeIo2:
			ebsCost.Desc.Io2Size += size
			ebsCost.Desc.Io2IOPS += Iops
		case ec2.VolumeTypeGp2:
			ebsCost.Desc.Gp2Size += size
		case ec2.VolumeTypeGp3:
			ebsCost.Desc.Gp3Size += size
			ebsCost.Desc.Gp3IOPS += int(math.Max(float64(Iops-costManifest.Gp3.FreeIOPSThreshold), 0))
			ebsCost.Desc.Gp3Throughput += int(math.Max(float64(throughput-costManifest.Gp3.FreeThroughputThreshold), 0))
		case ec2.VolumeTypeSc1:
			ebsCost.Desc.Sc1Size += size
		case ec2.VolumeTypeSt1:
			ebsCost.Desc.St1Size += size
		}
		ebsCost.Desc.CostValue = ebsCost.Desc.CalculateCostFromPriceJSON()
		ebsCostsRegionMap[key] = ebsCost
	}
	for _, v := range ebsCostsRegionMap {
		docs = append(docs, es.ServiceCostSummary{
			SummarizeJobTime: v.Base.SummarizeJobTime,
			SummarizeJobID:   v.Base.SummarizeJobID,
			ServiceName:      v.Base.ServiceName,
			SourceID:         v.Base.SourceID,
			SourceType:       v.Base.SourceType,
			SourceJobID:      v.Base.SourceJobID,
			ResourceType:     v.Base.ResourceType,
			ReportType:       v.Base.ReportType,
			Region:           v.Base.Region,

			Cost:        v.Desc,
			CostValue:   v.Desc.CostValue,
			PeriodStart: nowTime,
			PeriodEnd:   nowTime,
		})
	}
	return docs
}

func (b *costSummaryBuilder) Cleanup(summarizeJobID uint) error {
	return nil
}
