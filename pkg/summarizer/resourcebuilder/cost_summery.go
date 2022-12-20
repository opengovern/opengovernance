package resourcebuilder

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
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
		desc := model.CostExplorerByServiceMonthlyDescription{}
		err = json.Unmarshal(jsonDesc, &desc)
		if err != nil {
			return
		}
		fullResource.Description = desc

		key := fmt.Sprintf("%s|%s|%s", resource.SourceID, *desc.PeriodStart, *desc.PeriodEnd)
		if _, ok := b.costsByService[key]; !ok {
			b.costsByService[key] = es.ServiceCostSummary{
				ServiceName:   *desc.Dimension1,
				ScheduleJobID: resource.ScheduleJobID,
				SourceID:      resource.SourceID,
				SourceType:    resource.SourceType,
				SourceJobID:   resource.SourceJobID,
				ResourceType:  resource.ResourceType,
				Cost:          desc,
				PeriodStart:   getTimeFromTimestring(*desc.PeriodStart).Unix(),
				PeriodEnd:     getTimeFromTimestring(*desc.PeriodEnd).Unix(),
				ReportType:    es.CostProviderSummary,
			}
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
		desc := model.CostExplorerByAccountMonthlyDescription{}
		err = json.Unmarshal(jsonDesc, &desc)
		if err != nil {
			return
		}
		fullResource.Description = desc
		key := fmt.Sprintf("%s|%s|%s", resource.SourceID, *desc.PeriodStart, *desc.PeriodEnd)
		if _, ok := b.costsByAccount[key]; !ok {
			b.costsByAccount[key] = es.ConnectionCostSummary{
				AccountID:     *desc.Dimension1,
				ScheduleJobID: resource.ScheduleJobID,
				SourceID:      resource.SourceID,
				SourceType:    resource.SourceType,
				SourceJobID:   resource.SourceJobID,
				ResourceType:  resource.ResourceType,
				Cost:          desc,
				PeriodStart:   getTimeFromTimestring(*desc.PeriodStart).Unix(),
				PeriodEnd:     getTimeFromTimestring(*desc.PeriodEnd).Unix(),
				ReportType:    es.CostConnectionSummary,
			}
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

func (b *costSummaryBuilder) Cleanup(scheduleJobID uint) error {
	return nil
}
