package service

import (
	"github.com/kaytu-io/kaytu-engine/pkg/alerting/client"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	client4 "github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	client5 "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	client2 "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"github.com/kaytu-io/kaytu-engine/services/subscription/api/entities"
	"github.com/kaytu-io/kaytu-engine/services/subscription/db/model"
	"strings"
	"time"
)

func (svc MeteringService) generateMeter(workspaceId string, usageDate time.Time, meterType entities.MeterType) error {
	var err error
	switch meterType {
	case entities.MeterType_InventoryDiscoveryJobCount:
		err = svc.generateInventoryDiscoveryJobCountMeter(workspaceId, usageDate)
	case entities.MeterType_CostDiscoveryJobCount:
		err = svc.generateCostDiscoveryJobCountMeter(workspaceId, usageDate)
	case entities.MeterType_MetricEvaluationCount:
		err = svc.generateMetricEvaluationCountMeter(workspaceId, usageDate)
	case entities.MeterType_InsightEvaluationCount:
		err = svc.generateInsightEvaluationCountMeter(workspaceId, usageDate)
	case entities.MeterType_BenchmarkEvaluationCount:
		err = svc.generateBenchmarkEvaluationCountMeter(workspaceId, usageDate)
	case entities.MeterType_TotalFindings:
		err = svc.generateTotalFindingsMeter(workspaceId, usageDate)
	case entities.MeterType_TotalResource:
		err = svc.generateTotalResourceMeter(workspaceId, usageDate)
	case entities.MeterType_TotalUsers:
		err = svc.generateTotalUsersMeter(workspaceId, usageDate)
	case entities.MeterType_TotalApiKeys:
		err = svc.generateTotalApiKeysMeter(workspaceId, usageDate)
	case entities.MeterType_TotalRules:
		err = svc.generateTotalRulesMeter(workspaceId, usageDate)
	case entities.MeterType_AlertCount:
		err = svc.generateAlertCountMeter(workspaceId, usageDate)
	}

	return err
}

func getStartEndByDateHour(tim time.Time) (time.Time, time.Time) {
	startDate := time.Date(tim.Year(), tim.Month(), tim.Day(), tim.Hour(), 0, 0, 0, tim.Location())
	endDate := time.Date(tim.Year(), tim.Month(), tim.Day(), tim.Hour()+1, 0, 0, 0, tim.Location())
	endDate = endDate.Add(-1 * time.Millisecond)

	return startDate, endDate
}

func (svc MeteringService) generateSchedulerMeter(workspaceId string, usageDate time.Time, jobType api2.JobType, includeCost *bool, meterType entities.MeterType) error {
	ctx := &httpclient.Context{UserRole: api.InternalRole}
	schedulerClient := client5.NewSchedulerServiceClient(strings.ReplaceAll(svc.cnf.Scheduler.BaseURL, "%NAMESPACE%", workspaceId))
	startDate, endDate := getStartEndByDateHour(usageDate)

	count, err := schedulerClient.CountJobsByDate(ctx, includeCost, jobType, startDate, endDate)
	if err != nil {
		return err
	}
	return svc.db.CreateMeter(&model.Meter{
		WorkspaceID: workspaceId,
		UsageDate:   usageDate,
		MeterType:   meterType,
		CreatedAt:   time.Now(),
		Value:       count,
		Published:   false,
	})
}

func (svc MeteringService) generateInventoryDiscoveryJobCountMeter(workspaceId string, usageDate time.Time) error {
	v := false
	return svc.generateSchedulerMeter(workspaceId, usageDate, api2.JobType_Discovery, &v, entities.MeterType_InventoryDiscoveryJobCount)
}

func (svc MeteringService) generateCostDiscoveryJobCountMeter(workspaceId string, usageDate time.Time) error {
	v := true
	return svc.generateSchedulerMeter(workspaceId, usageDate, api2.JobType_Discovery, &v, entities.MeterType_CostDiscoveryJobCount)
}

func (svc MeteringService) generateMetricEvaluationCountMeter(workspaceId string, usageDate time.Time) error {
	return svc.generateSchedulerMeter(workspaceId, usageDate, api2.JobType_Analytics, nil, entities.MeterType_MetricEvaluationCount)
}

func (svc MeteringService) generateInsightEvaluationCountMeter(workspaceId string, usageDate time.Time) error {
	return svc.generateSchedulerMeter(workspaceId, usageDate, api2.JobType_Insight, nil, entities.MeterType_InsightEvaluationCount)
}

func (svc MeteringService) generateBenchmarkEvaluationCountMeter(workspaceId string, usageDate time.Time) error {
	return svc.generateSchedulerMeter(workspaceId, usageDate, api2.JobType_Compliance, nil, entities.MeterType_BenchmarkEvaluationCount)
}

func (svc MeteringService) generateTotalFindingsMeter(workspaceId string, usageDate time.Time) error {
	ctx := &httpclient.Context{UserRole: api.InternalRole}
	complianceClient := client4.NewComplianceClient(strings.ReplaceAll(svc.cnf.Compliance.BaseURL, "%NAMESPACE%", workspaceId))

	count, err := complianceClient.CountFindings(ctx)
	if err != nil {
		return err
	}
	return svc.db.CreateMeter(&model.Meter{
		WorkspaceID: workspaceId,
		UsageDate:   usageDate,
		MeterType:   entities.MeterType_TotalFindings,
		CreatedAt:   time.Now(),
		Value:       count,
		Published:   false,
	})
}

func (svc MeteringService) generateTotalResourceMeter(workspaceId string, usageDate time.Time) error {
	ctx := &httpclient.Context{UserRole: api.InternalRole}
	inventoryClient := client2.NewInventoryServiceClient(strings.ReplaceAll(svc.cnf.Inventory.BaseURL, "%NAMESPACE%", workspaceId))

	count, err := inventoryClient.CountResources(ctx)
	if err != nil {
		return err
	}
	return svc.db.CreateMeter(&model.Meter{
		WorkspaceID: workspaceId,
		UsageDate:   usageDate,
		MeterType:   entities.MeterType_TotalResource,
		CreatedAt:   time.Now(),
		Value:       count,
		Published:   false,
	})
}

func (svc MeteringService) generateTotalUsersMeter(workspaceId string, usageDate time.Time) error {
	ctx := &httpclient.Context{UserRole: api.InternalRole, UserID: api.GodUserID}
	users, err := svc.authClient.GetWorkspaceRoleBindings(ctx, workspaceId)
	if err != nil {
		return err
	}

	return svc.db.CreateMeter(&model.Meter{
		WorkspaceID: workspaceId,
		UsageDate:   usageDate,
		MeterType:   entities.MeterType_TotalUsers,
		CreatedAt:   time.Now(),
		Value:       int64(len(users)),
		Published:   false,
	})
}

func (svc MeteringService) generateTotalApiKeysMeter(workspaceId string, usageDate time.Time) error {
	ctx := &httpclient.Context{UserRole: api.InternalRole, UserID: api.GodUserID}
	apiKeys, err := svc.authClient.ListAPIKeys(ctx, workspaceId)
	if err != nil {
		return err
	}

	return svc.db.CreateMeter(&model.Meter{
		WorkspaceID: workspaceId,
		UsageDate:   usageDate,
		MeterType:   entities.MeterType_TotalApiKeys,
		CreatedAt:   time.Now(),
		Value:       int64(len(apiKeys)),
		Published:   false,
	})
}

func (svc MeteringService) generateTotalRulesMeter(workspaceId string, usageDate time.Time) error {
	ctx := &httpclient.Context{UserRole: api.InternalRole}
	alertingClient := client.NewAlertingServiceClient(strings.ReplaceAll(svc.cnf.Alerting.BaseURL, "%NAMESPACE%", workspaceId))
	rules, err := alertingClient.ListRules(ctx)
	if err != nil {
		return err
	}

	return svc.db.CreateMeter(&model.Meter{
		WorkspaceID: workspaceId,
		UsageDate:   usageDate,
		MeterType:   entities.MeterType_TotalRules,
		CreatedAt:   time.Now(),
		Value:       int64(len(rules)),
		Published:   false,
	})
}

func (svc MeteringService) generateAlertCountMeter(workspaceId string, usageDate time.Time) error {
	ctx := &httpclient.Context{UserRole: api.InternalRole}
	alertingClient := client.NewAlertingServiceClient(strings.ReplaceAll(svc.cnf.Alerting.BaseURL, "%NAMESPACE%", workspaceId))
	startDate, endDate := getStartEndByDateHour(usageDate)
	count, err := alertingClient.CountTriggersByDate(ctx, startDate, endDate)
	if err != nil {
		return err
	}

	return svc.db.CreateMeter(&model.Meter{
		WorkspaceID: workspaceId,
		UsageDate:   usageDate,
		MeterType:   entities.MeterType_AlertCount,
		CreatedAt:   time.Now(),
		Value:       count,
		Published:   false,
	})
}
