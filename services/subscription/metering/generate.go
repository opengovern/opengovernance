package metering

import (
	"github.com/kaytu-io/kaytu-engine/pkg/alerting/client"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	client4 "github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	client5 "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	client2 "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"github.com/kaytu-io/kaytu-engine/services/subscription/db/model"
	"strings"
	"time"
)

func (s *Service) generateMeter(workspaceId, dateHour string, meterType model.MeterType) error {
	var err error
	switch meterType {
	case model.MeterType_InventoryDiscoveryJobCount:
		err = s.generateInventoryDiscoveryJobCountMeter(workspaceId, dateHour)
	case model.MeterType_CostDiscoveryJobCount:
		err = s.generateCostDiscoveryJobCountMeter(workspaceId, dateHour)
	case model.MeterType_MetricEvaluationCount:
		err = s.generateMetricEvaluationCountMeter(workspaceId, dateHour)
	case model.MeterType_InsightEvaluationCount:
		err = s.generateInsightEvaluationCountMeter(workspaceId, dateHour)
	case model.MeterType_BenchmarkEvaluationCount:
		err = s.generateBenchmarkEvaluationCountMeter(workspaceId, dateHour)
	case model.MeterType_TotalFindings:
		err = s.generateTotalFindingsMeter(workspaceId, dateHour)
	case model.MeterType_TotalResource:
		err = s.generateTotalResourceMeter(workspaceId, dateHour)
	case model.MeterType_TotalUsers:
		err = s.generateTotalUsersMeter(workspaceId, dateHour)
	case model.MeterType_TotalApiKeys:
		err = s.generateTotalApiKeysMeter(workspaceId, dateHour)
	case model.MeterType_TotalRules:
		err = s.generateTotalRulesMeter(workspaceId, dateHour)
	case model.MeterType_AlertCount:
		err = s.generateAlertCountMeter(workspaceId, dateHour)
	}

	return err
}

func getStartEndByDateHour(dateHour string) (time.Time, time.Time) {
	tim, _ := time.Parse("2006-01-02-15", dateHour)
	startDate := time.Date(tim.Year(), tim.Month(), tim.Day(), tim.Hour(), 0, 0, 0, tim.Location())
	endDate := time.Date(tim.Year(), tim.Month(), tim.Day(), tim.Hour()+1, 0, 0, 0, tim.Location())
	endDate = endDate.Add(-1 * time.Millisecond)

	return startDate, endDate
}

func (s *Service) generateSchedulerMeter(workspaceId, dateHour string, jobType api2.JobType, meterType model.MeterType) error {
	ctx := &httpclient.Context{UserRole: api.InternalRole}
	schedulerClient := client5.NewSchedulerServiceClient(strings.ReplaceAll(s.cnf.Scheduler.BaseURL, "{NAMESPACE}", workspaceId))
	startDate, endDate := getStartEndByDateHour(dateHour)

	count, err := schedulerClient.CountJobsByDate(ctx, jobType, startDate, endDate)
	if err != nil {
		return err
	}
	return s.pdb.CreateMeter(&model.Meter{
		WorkspaceID: workspaceId,
		DateHour:    dateHour,
		MeterType:   meterType,
		CreatedAt:   time.Now(),
		Value:       count,
		Published:   false,
	})
}

func (s *Service) generateInventoryDiscoveryJobCountMeter(workspaceId, dateHour string) error {
	return s.generateSchedulerMeter(workspaceId, dateHour, api2.JobType_Discovery, model.MeterType_InventoryDiscoveryJobCount)
}

func (s *Service) generateCostDiscoveryJobCountMeter(workspaceId, dateHour string) error {
	return s.generateSchedulerMeter(workspaceId, dateHour, api2.JobType_Discovery, model.MeterType_CostDiscoveryJobCount)
}

func (s *Service) generateMetricEvaluationCountMeter(workspaceId, dateHour string) error {
	return s.generateSchedulerMeter(workspaceId, dateHour, api2.JobType_Analytics, model.MeterType_MetricEvaluationCount)
}

func (s *Service) generateInsightEvaluationCountMeter(workspaceId, dateHour string) error {
	return s.generateSchedulerMeter(workspaceId, dateHour, api2.JobType_Insight, model.MeterType_InsightEvaluationCount)
}

func (s *Service) generateBenchmarkEvaluationCountMeter(workspaceId, dateHour string) error {
	return s.generateSchedulerMeter(workspaceId, dateHour, api2.JobType_Compliance, model.MeterType_BenchmarkEvaluationCount)
}

func (s *Service) generateTotalFindingsMeter(workspaceId, dateHour string) error {
	ctx := &httpclient.Context{UserRole: api.InternalRole}
	complianceClient := client4.NewComplianceClient(strings.ReplaceAll(s.cnf.Compliance.BaseURL, "{NAMESPACE}", workspaceId))

	count, err := complianceClient.CountFindings(ctx)
	if err != nil {
		return err
	}
	return s.pdb.CreateMeter(&model.Meter{
		WorkspaceID: workspaceId,
		DateHour:    dateHour,
		MeterType:   model.MeterType_TotalFindings,
		CreatedAt:   time.Now(),
		Value:       count,
		Published:   false,
	})
}

func (s *Service) generateTotalResourceMeter(workspaceId, dateHour string) error {
	ctx := &httpclient.Context{UserRole: api.InternalRole}
	inventoryClient := client2.NewInventoryServiceClient(strings.ReplaceAll(s.cnf.Inventory.BaseURL, "{NAMESPACE}", workspaceId))

	count, err := inventoryClient.CountResources(ctx)
	if err != nil {
		return err
	}
	return s.pdb.CreateMeter(&model.Meter{
		WorkspaceID: workspaceId,
		DateHour:    dateHour,
		MeterType:   model.MeterType_TotalResource,
		CreatedAt:   time.Now(),
		Value:       count,
		Published:   false,
	})
}

func (s *Service) generateTotalUsersMeter(workspaceId, dateHour string) error {
	ctx := &httpclient.Context{UserRole: api.InternalRole, UserID: api.GodUserID}
	users, err := s.authClient.GetWorkspaceRoleBindings(ctx, workspaceId)
	if err != nil {
		return err
	}

	return s.pdb.CreateMeter(&model.Meter{
		WorkspaceID: workspaceId,
		DateHour:    dateHour,
		MeterType:   model.MeterType_TotalUsers,
		CreatedAt:   time.Now(),
		Value:       int64(len(users)),
		Published:   false,
	})
}

func (s *Service) generateTotalApiKeysMeter(workspaceId, dateHour string) error {
	ctx := &httpclient.Context{UserRole: api.InternalRole}
	apiKeys, err := s.authClient.ListAPIKeys(ctx, workspaceId)
	if err != nil {
		return err
	}

	return s.pdb.CreateMeter(&model.Meter{
		WorkspaceID: workspaceId,
		DateHour:    dateHour,
		MeterType:   model.MeterType_TotalApiKeys,
		CreatedAt:   time.Now(),
		Value:       int64(len(apiKeys)),
		Published:   false,
	})
}

func (s *Service) generateTotalRulesMeter(workspaceId, dateHour string) error {
	ctx := &httpclient.Context{UserRole: api.InternalRole}
	alertingClient := client.NewAlertingServiceClient(strings.ReplaceAll(s.cnf.Alerting.BaseURL, "{NAMESPACE}", workspaceId))
	rules, err := alertingClient.ListRules(ctx)
	if err != nil {
		return err
	}

	return s.pdb.CreateMeter(&model.Meter{
		WorkspaceID: workspaceId,
		DateHour:    dateHour,
		MeterType:   model.MeterType_TotalRules,
		CreatedAt:   time.Now(),
		Value:       int64(len(rules)),
		Published:   false,
	})
}

func (s *Service) generateAlertCountMeter(workspaceId, dateHour string) error {
	ctx := &httpclient.Context{UserRole: api.InternalRole}
	alertingClient := client.NewAlertingServiceClient(strings.ReplaceAll(s.cnf.Alerting.BaseURL, "{NAMESPACE}", workspaceId))
	startDate, endDate := getStartEndByDateHour(dateHour)
	count, err := alertingClient.CountTriggersByDate(ctx, startDate, endDate)
	if err != nil {
		return err
	}

	return s.pdb.CreateMeter(&model.Meter{
		WorkspaceID: workspaceId,
		DateHour:    dateHour,
		MeterType:   model.MeterType_AlertCount,
		CreatedAt:   time.Now(),
		Value:       count,
		Published:   false,
	})
}
