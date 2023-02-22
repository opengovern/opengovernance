package resourcebuilder

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	awsModel "gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
	azureModel "gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	describe "gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/query"
)

type costSummaryBuilder struct {
	client          keibi.Client
	summarizerJobID uint
	costsByService  map[string]es.ServiceCostSummary
	costsByAccount  map[string]es.ConnectionCostSummary
}

func NewCostSummaryBuilder(client keibi.Client, summarizerJobID uint) *costSummaryBuilder {
	return &costSummaryBuilder{
		client:          client,
		summarizerJobID: summarizerJobID,
		costsByService:  make(map[string]es.ServiceCostSummary),
		costsByAccount:  make(map[string]es.ConnectionCostSummary),
	}
}

func getTimeFromTimestring(timestring string) time.Time {
	t, _ := time.Parse("2006-01-02", timestring)
	return t
}

func getTimeFromTimeInt(timeint int) time.Time {
	timestring := fmt.Sprintf("%d", timeint)
	t, _ := time.Parse("20060102", timestring)
	return t
}

func (b *costSummaryBuilder) Process(resource describe.LookupResource) {
	var fullResource *describe.Resource
	var err error
	switch strings.ToLower(resource.ResourceType) {
	case "aws::costexplorer::byservicemonthly":
		fullResource, err = query.GetResourceFromResourceLookup(b.client, resource)
		if err != nil {
			fmt.Printf("(costSummaryBuilder) - Error getting resource from lookup: %v", err)
			return
		}
		jsonDesc, err := json.Marshal(fullResource.Description)
		if err != nil {
			return
		}
		desc := awsModel.CostExplorerByServiceMonthlyDescription{}
		err = json.Unmarshal(jsonDesc, &desc)
		if err != nil {
			return
		}
		fullResource.Description = desc

		key := fmt.Sprintf("%s|%s|%s|%s", resource.SourceID, *desc.Dimension1, *desc.PeriodStart, *desc.PeriodEnd)
		if _, ok := b.costsByService[key]; !ok {
			v := es.ServiceCostSummary{
				SummarizeJobID: b.summarizerJobID,
				ServiceName:    *desc.Dimension1,
				ScheduleJobID:  resource.ScheduleJobID,
				SourceID:       resource.SourceID,
				SourceType:     resource.SourceType,
				SourceJobID:    resource.SourceJobID,
				ResourceType:   resource.ResourceType,
				Cost:           desc,
				PeriodStart:    getTimeFromTimestring(*desc.PeriodStart).Unix(),
				PeriodEnd:      getTimeFromTimestring(*desc.PeriodEnd).Unix(),
			}
			v.ReportType = es.CostProviderSummaryMonthly
			b.costsByService[key] = v
		}
	case "aws::costexplorer::byservicedaily":
		fullResource, err = query.GetResourceFromResourceLookup(b.client, resource)
		if err != nil {
			fmt.Printf("(costSummaryBuilder) - Error getting resource from lookup: %v", err)
			return
		}
		jsonDesc, err := json.Marshal(fullResource.Description)
		if err != nil {
			return
		}
		desc := awsModel.CostExplorerByServiceDailyDescription{}
		err = json.Unmarshal(jsonDesc, &desc)
		if err != nil {
			return
		}
		fullResource.Description = desc

		key := fmt.Sprintf("%s|%s|%s|%s", resource.SourceID, *desc.Dimension1, *desc.PeriodStart, *desc.PeriodEnd)
		if _, ok := b.costsByService[key]; !ok {
			v := es.ServiceCostSummary{
				SummarizeJobID: b.summarizerJobID,
				ServiceName:    *desc.Dimension1,
				ScheduleJobID:  resource.ScheduleJobID,
				SourceID:       resource.SourceID,
				SourceType:     resource.SourceType,
				SourceJobID:    resource.SourceJobID,
				ResourceType:   resource.ResourceType,
				Cost:           desc,
				PeriodStart:    getTimeFromTimestring(*desc.PeriodStart).Unix(),
				PeriodEnd:      getTimeFromTimestring(*desc.PeriodEnd).Unix(),
			}
			v.ReportType = es.CostProviderSummaryDaily
			b.costsByService[key] = v
		}
	case "aws::costexplorer::byaccountmonthly":
		fullResource, err = query.GetResourceFromResourceLookup(b.client, resource)
		if err != nil {
			fmt.Printf("(costSummaryBuilder) - Error getting resource from lookup: %v", err)
			return
		}
		jsonDesc, err := json.Marshal(fullResource.Description)
		if err != nil {
			return
		}
		desc := awsModel.CostExplorerByAccountMonthlyDescription{}
		err = json.Unmarshal(jsonDesc, &desc)
		if err != nil {
			return
		}
		fullResource.Description = desc
		key := fmt.Sprintf("%s|%s|%s", resource.SourceID, *desc.PeriodStart, *desc.PeriodEnd)
		if _, ok := b.costsByAccount[key]; !ok {
			v := es.ConnectionCostSummary{
				SummarizeJobID: b.summarizerJobID,
				AccountID:      *desc.Dimension1,
				ScheduleJobID:  resource.ScheduleJobID,
				SourceID:       resource.SourceID,
				SourceType:     resource.SourceType,
				SourceJobID:    resource.SourceJobID,
				ResourceType:   resource.ResourceType,
				Cost:           desc,
				PeriodStart:    getTimeFromTimestring(*desc.PeriodStart).Unix(),
				PeriodEnd:      getTimeFromTimestring(*desc.PeriodEnd).Unix(),
			}
			v.ReportType = es.CostConnectionSummaryMonthly
			b.costsByAccount[key] = v
		}
	case "aws::costexplorer::byaccountdaily":
		fullResource, err = query.GetResourceFromResourceLookup(b.client, resource)
		if err != nil {
			fmt.Printf("(costSummaryBuilder) - Error getting resource from lookup: %v", err)
			return
		}
		jsonDesc, err := json.Marshal(fullResource.Description)
		if err != nil {
			return
		}
		desc := awsModel.CostExplorerByAccountDailyDescription{}
		err = json.Unmarshal(jsonDesc, &desc)
		if err != nil {
			return
		}
		fullResource.Description = desc
		key := fmt.Sprintf("%s|%s|%s", resource.SourceID, *desc.PeriodStart, *desc.PeriodEnd)
		if _, ok := b.costsByAccount[key]; !ok {
			v := es.ConnectionCostSummary{
				SummarizeJobID: b.summarizerJobID,
				AccountID:      *desc.Dimension1,
				ScheduleJobID:  resource.ScheduleJobID,
				SourceID:       resource.SourceID,
				SourceType:     resource.SourceType,
				SourceJobID:    resource.SourceJobID,
				ResourceType:   resource.ResourceType,
				Cost:           desc,
				PeriodStart:    getTimeFromTimestring(*desc.PeriodStart).Unix(),
				PeriodEnd:      getTimeFromTimestring(*desc.PeriodEnd).Unix(),
			}
			v.ReportType = es.CostConnectionSummaryDaily
			b.costsByAccount[key] = v
		}
	case "microsoft.costmanagement/costbyresourcetype":
		fullResource, err = query.GetResourceFromResourceLookup(b.client, resource)
		if err != nil {
			fmt.Printf("(costSummaryBuilder) - Error getting resource from lookup: %v", err)
			return
		}
		jsonDesc, err := json.Marshal(fullResource.Description)
		if err != nil {
			return
		}
		desc := azureModel.CostManagementCostByResourceTypeDescription{}
		err = json.Unmarshal(jsonDesc, &desc)
		if err != nil {
			return
		}
		fullResource.Description = desc

		key := fmt.Sprintf("%s|%s|%s|%d", resource.SourceID, *desc.CostManagementCostByResourceType.ResourceType, desc.CostManagementCostByResourceType.Currency, desc.CostManagementCostByResourceType.UsageDate)
		if _, ok := b.costsByService[key]; !ok {
			v := es.ServiceCostSummary{
				SummarizeJobID: b.summarizerJobID,
				ServiceName:    *desc.CostManagementCostByResourceType.ResourceType,
				ScheduleJobID:  resource.ScheduleJobID,
				SourceID:       resource.SourceID,
				SourceType:     resource.SourceType,
				SourceJobID:    resource.SourceJobID,
				ResourceType:   resource.ResourceType,
				Cost:           desc.CostManagementCostByResourceType,
				PeriodStart:    getTimeFromTimeInt(desc.CostManagementCostByResourceType.UsageDate).Unix(),
				PeriodEnd:      getTimeFromTimeInt(desc.CostManagementCostByResourceType.UsageDate).Unix(),
			}
			v.ReportType = es.CostProviderSummaryDaily
			b.costsByService[key] = v
		}
	case "microsoft.costmanagement/costbysubscription":
		fullResource, err = query.GetResourceFromResourceLookup(b.client, resource)
		if err != nil {
			fmt.Printf("(costSummaryBuilder) - Error getting resource from lookup: %v", err)
			return
		}
		jsonDesc, err := json.Marshal(fullResource.Description)
		if err != nil {
			return
		}
		desc := azureModel.CostManagementCostBySubscriptionDescription{}
		err = json.Unmarshal(jsonDesc, &desc)
		if err != nil {
			return
		}
		fullResource.Description = desc
		key := fmt.Sprintf("%s|%s|%d", resource.SourceID, desc.CostManagementCostBySubscription.Currency, desc.CostManagementCostBySubscription.UsageDate)
		if _, ok := b.costsByAccount[key]; !ok {
			v := es.ConnectionCostSummary{
				SummarizeJobID: b.summarizerJobID,
				AccountID:      resource.SourceID,
				ScheduleJobID:  resource.ScheduleJobID,
				SourceID:       resource.SourceID,
				SourceType:     resource.SourceType,
				SourceJobID:    resource.SourceJobID,
				ResourceType:   resource.ResourceType,
				Cost:           desc.CostManagementCostBySubscription,
				PeriodStart:    getTimeFromTimeInt(desc.CostManagementCostBySubscription.UsageDate).Unix(),
				PeriodEnd:      getTimeFromTimeInt(desc.CostManagementCostBySubscription.UsageDate).Unix(),
			}
			v.ReportType = es.CostConnectionSummaryDaily
			b.costsByAccount[key] = v
		}
	default:
		return
	}
}

func (b *costSummaryBuilder) PopulateHistory(lastDayJobID, lastWeekJobID, lastQuarterJobID, lastYearJobID uint) error {
	return nil
}

func (b *costSummaryBuilder) Build() []kafka.Doc {
	var docs []kafka.Doc
	for _, v := range b.costsByService {
		docs = append(docs, v)
	}
	for _, v := range b.costsByAccount {
		docs = append(docs, v)
	}
	return docs
}

func (b *costSummaryBuilder) Cleanup(summarizeJobID uint) error {
	return nil
}
