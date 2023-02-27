package resourcebuilder

import (
	"fmt"
	"time"

	ec2 "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	awsModel "gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
	describe "gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
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
		serviceCostSummary.ScheduleJobID = resource.ScheduleJobID
		serviceCostSummary.SourceType = resource.SourceType
		serviceCostSummary.SourceID = resource.SourceID
		serviceCostSummary.SourceJobID = resource.SourceJobID
		serviceCostSummary.ResourceType = resource.ResourceType
		if _, ok := b.costsByService[key]; !ok {
			b.costsByService[key] = *serviceCostSummary
		}
	case *es.ConnectionCostSummary:
		connectionCostSummary := costSummary.(*es.ConnectionCostSummary)
		connectionCostSummary.SummarizeJobID = b.summarizerJobID
		connectionCostSummary.ScheduleJobID = resource.ScheduleJobID
		connectionCostSummary.SourceType = resource.SourceType
		connectionCostSummary.SourceID = resource.SourceID
		connectionCostSummary.SourceJobID = resource.SourceJobID
		connectionCostSummary.ResourceType = resource.ResourceType
		if _, ok := b.costsByAccount[key]; !ok {
			b.costsByAccount[key] = *connectionCostSummary
		}
	}
}

func (b *costSummaryBuilder) PopulateHistory(lastDayJobID, lastWeekJobID, lastQuarterJobID, lastYearJobID uint) error {
	return nil
}

func (b *costSummaryBuilder) Build() []kafka.Doc {
	var docs []kafka.Doc
	ebsCosts := make([]es.ServiceCostSummary, 0)
	ebsCostsRegionMap := make(map[string]EBSCostDoc)
	for _, v := range b.costsByService {
		switch v.ServiceName {
		case string(es.CostResourceTypeAWSEBSVolume):
			ebsCosts = append(ebsCosts, v)
		default:
			docs = append(docs, v)
		}
	}
	for _, v := range ebsCosts {
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
			ebsCost.Desc.Gp3IOPS += Iops
			ebsCost.Desc.Gp3Throughput += throughput
		case ec2.VolumeTypeSc1:
			ebsCost.Desc.Sc1Size += size
		case ec2.VolumeTypeSt1:
			ebsCost.Desc.St1Size += size
		}
		ebsCostsRegionMap[*v.Region] = ebsCost
	}

	nowTime := time.Now().Unix()

	for _, v := range ebsCostsRegionMap {
		docs = append(docs, es.ServiceCostSummary{
			SummarizeJobID: v.Base.SummarizeJobID,
			ServiceName:    v.Base.ServiceName,
			ScheduleJobID:  v.Base.ScheduleJobID,
			SourceID:       v.Base.SourceID,
			SourceType:     v.Base.SourceType,
			SourceJobID:    v.Base.SourceJobID,
			ResourceType:   v.Base.ResourceType,
			ReportType:     v.Base.ReportType,
			Region:         v.Base.Region,

			Cost:        v.Desc,
			PeriodStart: nowTime,
			PeriodEnd:   nowTime,
		})
	}
	return docs
}

func (b *costSummaryBuilder) Cleanup(summarizeJobID uint) error {
	return nil
}
