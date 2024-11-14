package analytics

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"

	authApi "github.com/opengovern/og-util/pkg/api"
	shared_entities "github.com/opengovern/og-util/pkg/api/shared-entities"
	esSinkClient "github.com/opengovern/og-util/pkg/es/ingest/client"
	"github.com/opengovern/og-util/pkg/httpclient"
	"github.com/opengovern/og-util/pkg/jq"
	"github.com/opengovern/opengovernance/pkg/utils"
	integrationApi "github.com/opengovern/opengovernance/services/integration/api/models"
	integrationClient "github.com/opengovern/opengovernance/services/integration/client"
	inventoryApi "github.com/opengovern/opengovernance/services/inventory/api"

	"github.com/opengovern/og-util/pkg/es"
	"github.com/opengovern/og-util/pkg/steampipe"
	"github.com/opengovern/opengovernance/pkg/analytics/api"
	"github.com/opengovern/opengovernance/pkg/analytics/config"
	"github.com/opengovern/opengovernance/pkg/analytics/db"
	"github.com/opengovern/opengovernance/pkg/analytics/es/resource"
	"github.com/opengovern/opengovernance/pkg/analytics/es/spend"
	describeApi "github.com/opengovern/opengovernance/pkg/describe/api"
	describeClient "github.com/opengovern/opengovernance/pkg/describe/client"
	inventoryClient "github.com/opengovern/opengovernance/services/inventory/client"
	"go.uber.org/zap"
)

type Job struct {
	JobID                 uint
	ResourceCollectionIDs []string
}

type JobResult struct {
	JobID  uint
	Status api.JobStatus
	Error  string
}

func (j *Job) Do(
	jq *jq.JobQueue,
	db db.Database,
	steampipeConn *steampipe.Database,
	integrationClient integrationClient.IntegrationServiceClient,
	schedulerClient describeClient.SchedulerServiceClient,
	inventoryClient inventoryClient.InventoryServiceClient,
	sinkClient esSinkClient.EsSinkServiceClient,
	logger *zap.Logger,
	config config.WorkerConfig,
	ctx context.Context,
) JobResult {
	result := JobResult{
		JobID:  j.JobID,
		Status: api.JobCompleted,
		Error:  "",
	}
	fail := func(err error) JobResult {
		result.Error = err.Error()
		result.Status = api.JobCompletedWithFailure
		return result
	}

	encodedResourceCollectionFilters := make(map[string]string)
	if len(j.ResourceCollectionIDs) > 0 {
		ctx2 := &httpclient.Context{UserRole: authApi.AdminRole}
		ctx2.Ctx = ctx
		rcs, err := inventoryClient.ListResourceCollectionsMetadata(ctx2,
			j.ResourceCollectionIDs)
		if err != nil {
			return fail(err)
		}
		for _, rc := range rcs {
			filtersJson, err := json.Marshal(rc.Filters)
			if err != nil {
				return fail(err)
			}
			encodedResourceCollectionFilters[rc.ID] = base64.StdEncoding.EncodeToString(filtersJson)
		}
	}

	err := steampipeConn.SetConfigTableValue(ctx, steampipe.OpenGovernanceConfigKeyAccountID, "all")
	if err != nil {
		logger.Error("failed to set steampipe context config for account id", zap.Error(err), zap.String("account_id", "all"))
		return fail(err)
	}
	defer steampipeConn.UnsetConfigTableValue(ctx, steampipe.OpenGovernanceConfigKeyAccountID)

	err = steampipeConn.SetConfigTableValue(ctx, steampipe.OpenGovernanceConfigKeyClientType, "analytics")
	if err != nil {
		logger.Error("failed to set steampipe context config for client type", zap.Error(err), zap.String("client_type", "analytics"))
		return fail(err)
	}
	defer steampipeConn.UnsetConfigTableValue(ctx, steampipe.OpenGovernanceConfigKeyClientType)

	if err := j.Run(ctx, jq, db, encodedResourceCollectionFilters, steampipeConn, schedulerClient, integrationClient, sinkClient, inventoryClient, logger, config); err != nil {
		fail(err)
	}

	if config.DoTelemetry {
		// send telemetry
		j.SendTelemetry(ctx, logger, config, integrationClient, inventoryClient)
	}

	return result
}

func (j *Job) SendTelemetry(ctx context.Context, logger *zap.Logger, workerConfig config.WorkerConfig, integrationClient integrationClient.IntegrationServiceClient, inventoryClient inventoryClient.InventoryServiceClient) {
	now := time.Now()

	httpCtx := httpclient.Context{Ctx: ctx, UserRole: authApi.AdminRole}

	req := shared_entities.CspmUsageRequest{
		GatherTimestamp:      now,
		Hostname:             workerConfig.TelemetryHostname,
		IntegrationTypeCount: make(map[string]int),
		ApproximateSpend:     0,
	}

	integrations, err := integrationClient.ListIntegrations(&httpCtx, nil)
	if err != nil {
		logger.Error("failed to list sources", zap.Error(err))
		return
	}
	for _, integration := range integrations.Integrations {
		if _, ok := req.IntegrationTypeCount[integration.IntegrationType.String()]; !ok {
			req.IntegrationTypeCount[integration.IntegrationType.String()] = 0
		}
		req.IntegrationTypeCount[integration.IntegrationType.String()] += 1
	}

	connData, err := inventoryClient.ListIntegrationsData(&httpCtx, nil, nil,
		utils.GetPointer(now.AddDate(0, -1, 0)), &now, nil, true, false)
	if err != nil {
		logger.Error("failed to list connections data", zap.Error(err))
		return
	}
	totalSpend := float64(0)
	for _, conn := range connData {
		if conn.TotalCost != nil {
			totalSpend += *conn.TotalCost
		}
	}

	req.ApproximateSpend = int(math.Floor(totalSpend/5000000)) * 5000000

	url := fmt.Sprintf("%s/api/v1/information/usage", workerConfig.TelemetryBaseURL)
	reqBytes, err := json.Marshal(req)
	if err != nil {
		logger.Error("failed to marshal telemetry request", zap.Error(err))
		return
	}
	var resp any
	if statusCode, err := httpclient.DoRequest(httpCtx.Ctx, http.MethodPost, url, httpCtx.ToHeaders(), reqBytes, &resp); err != nil {
		logger.Error("failed to send telemetry", zap.Error(err), zap.Int("status_code", statusCode), zap.String("url", url), zap.Any("req", req), zap.Any("resp", resp))
		return
	}

	logger.Info("sent telemetry", zap.String("url", url))
}

func (j *Job) Run(ctx context.Context, jq *jq.JobQueue, dbc db.Database, encodedResourceCollectionFilters map[string]string, steampipeDB *steampipe.Database, schedulerClient describeClient.SchedulerServiceClient, integrationClient integrationClient.IntegrationServiceClient, sinkClient esSinkClient.EsSinkServiceClient, inventoryClient inventoryClient.InventoryServiceClient, logger *zap.Logger, config config.WorkerConfig) error {
	startTime := time.Now()
	metrics, err := dbc.ListMetrics([]db.AnalyticMetricStatus{db.AnalyticMetricStatusActive, db.AnalyticMetricStatusInvisible})
	if err != nil {
		return err
	}

	integrationCache := map[string]integrationApi.Integration{}

	for _, metric := range metrics {
		switch metric.Type {
		case db.MetricTypeAssets:
			s := map[string]describeApi.DescribeStatus{}
			for _, resourceType := range metric.Tables {
				status, err := schedulerClient.GetDescribeStatus(&httpclient.Context{UserRole: authApi.AdminRole}, resourceType)
				if err != nil {
					return err
				}

				for _, st := range status {
					if v, ok := s[st.ConnectionID]; ok {
						if st.Status != describeApi.DescribeResourceJobSucceeded {
							v.Status = st.Status
							s[st.ConnectionID] = v
						}
					} else {
						s[st.ConnectionID] = st
					}
				}
			}

			var status []describeApi.DescribeStatus
			for _, v := range s {
				status = append(status, v)
			}

			err = j.DoAssetMetric(
				ctx,
				jq,
				steampipeDB,
				encodedResourceCollectionFilters,
				integrationClient,
				sinkClient,
				inventoryClient,
				logger,
				metric,
				integrationCache,
				startTime,
				status,
				config,
			)
			if err != nil {
				return err
			}
		case db.MetricTypeSpend:
			// We do not support spend metrics for resource collections
			if len(encodedResourceCollectionFilters) > 0 {
				continue
			}
			awsStatus, err := schedulerClient.GetDescribeStatus(&httpclient.Context{UserRole: authApi.AdminRole}, "AWS::CostExplorer::ByServiceDaily")
			if err != nil {
				return err
			}

			azureStatus, err := schedulerClient.GetDescribeStatus(&httpclient.Context{UserRole: authApi.AdminRole}, "Microsoft.CostManagement/CostByResourceType")
			if err != nil {
				return err
			}

			s := map[string]describeApi.DescribeStatus{}
			for _, st := range append(awsStatus, azureStatus...) {
				if v, ok := s[st.ConnectionID]; ok {
					if st.Status != describeApi.DescribeResourceJobSucceeded {
						v.Status = st.Status
						s[st.ConnectionID] = v
					}
				} else {
					s[st.ConnectionID] = st
				}
			}

			var status []describeApi.DescribeStatus
			for _, v := range s {
				status = append(status, v)
			}

			err = j.DoSpendMetric(
				ctx,
				jq,
				steampipeDB,
				integrationClient,
				sinkClient,
				inventoryClient,
				logger,
				metric,
				integrationCache,
				status,
				config,
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (j *Job) DoSingleAssetMetric(ctx context.Context, logger *zap.Logger, steampipeDB *steampipe.Database, metric db.AnalyticMetric,
	integrationCache map[string]integrationApi.Integration,
	status []describeApi.DescribeStatus,
	integrationClient integrationClient.IntegrationServiceClient,
	inventoryClient inventoryClient.InventoryServiceClient) (
	*resource.IntegrationMetricTrendSummaryResult,
	*resource.IntegrationTypeMetricTrendSummaryResult,
	error,
) {
	var res *steampipe.Result
	var err error

	logger.Info("assets ==== ", zap.String("query", metric.Query))
	if metric.Engine == db.QueryEngine_cloudqlRego {
		ctx2 := &httpclient.Context{UserRole: authApi.AdminRole}
		ctx2.Ctx = ctx
		var engine inventoryApi.QueryEngine
		engine = inventoryApi.QueryEngine_cloudqlRego
		results, err := inventoryClient.RunQuery(ctx2, inventoryApi.RunQueryRequest{
			Page: inventoryApi.Page{
				No:   1,
				Size: 1000,
			},
			Engine: &engine,
			Query:  &metric.Query,
			Sorts:  nil,
		})
		if err != nil {
			return nil, nil, err
		}
		res = &steampipe.Result{
			Headers: results.Headers,
			Data:    results.Result,
		}
	} else {
		res, err = steampipeDB.QueryAll(ctx, metric.Query)
		if err != nil {
			return nil, nil, err
		}
	}
	logger.Info("assets ==== ", zap.Int("count", len(res.Data)))

	totalCount := 0
	perConnection := make(map[string]resource.PerIntegrationMetricTrendSummary)
	perConnector := make(map[string]resource.PerIntegrationTypeMetricTrendSummary)

	connectorCount := map[string]int64{}
	connectorSuccessCount := map[string]int64{}
	for _, st := range status {
		connectorCount[st.Connector]++
		if st.Status == describeApi.DescribeResourceJobSucceeded {
			connectorSuccessCount[st.Connector]++
		}
	}

	for _, record := range res.Data {
		if len(record) != 3 {
			return nil, nil, fmt.Errorf("invalid query: %s", metric.Query)
		}

		integrationID, ok := record[0].(string)
		if !ok {
			return nil, nil, fmt.Errorf("invalid format for connectionId: [%s] %v", reflect.TypeOf(record[0]), record[0])
		}

		count, ok := record[2].(int64)
		if !ok {
			return nil, nil, fmt.Errorf("invalid format for count: [%s] %v", reflect.TypeOf(record[2]), record[2])
		}

		var integration *integrationApi.Integration
		if cached, ok := integrationCache[integrationID]; ok {
			integration = &cached
		} else {
			ctx2 := &httpclient.Context{UserRole: authApi.AdminRole}
			ctx2.Ctx = ctx
			integration, err = integrationClient.GetIntegration(ctx2, integrationID)
			if err != nil {
				if strings.Contains(err.Error(), "source not found") {
					continue
				}
				return nil, nil, fmt.Errorf("GetIntegration id=%s err=%v", integrationID, err)
			}
			if integration == nil {
				return nil, nil, fmt.Errorf("integration not found: %s", integrationID)
			}

			integrationCache[integrationID] = *integration
		}

		isJobSuccessful := true
		for _, st := range status {
			if st.ConnectionID == integration.IntegrationID {
				if st.Status == describeApi.DescribeResourceJobFailed || st.Status == describeApi.DescribeResourceJobTimeout {
					isJobSuccessful = false
				}
			}
		}

		if v, ok := perConnection[integration.IntegrationID]; ok {
			v.ResourceCount += int(count)
			perConnection[integration.IntegrationID] = v
		} else {
			vn := resource.PerIntegrationMetricTrendSummary{
				IntegrationID:   integration.IntegrationID,
				IntegrationName: integration.Name,
				IntegrationType: integration.IntegrationType,
				ResourceCount:   int(count),
				IsJobSuccessful: isJobSuccessful,
			}
			perConnection[integration.IntegrationID] = vn
		}

		if v, ok := perConnector[integration.IntegrationType.String()]; ok {
			v.ResourceCount += int(count)
			perConnector[integration.IntegrationType.String()] = v
		} else {
			vn := resource.PerIntegrationTypeMetricTrendSummary{
				IntegrationType:                 integration.IntegrationType,
				ResourceCount:                   int(count),
				TotalIntegrationTypes:           connectorCount[integration.IntegrationType.String()],
				TotalSuccessfulIntegrationTypes: connectorSuccessCount[integration.IntegrationType.String()],
			}
			perConnector[integration.IntegrationType.String()] = vn
		}
		totalCount += int(count)
	}
	perConnectionArray := make([]resource.PerIntegrationMetricTrendSummary, 0, len(perConnection))
	for _, v := range perConnection {
		perConnectionArray = append(perConnectionArray, v)
	}
	perConnectorArray := make([]resource.PerIntegrationTypeMetricTrendSummary, 0, len(perConnector))
	for _, v := range perConnector {
		perConnectorArray = append(perConnectorArray, v)
	}
	logger.Info("assets ==== ", zap.String("metric_id", metric.ID), zap.Int("totalCount", totalCount))

	return &resource.IntegrationMetricTrendSummaryResult{
			TotalResourceCount: totalCount,
			Integrations:       perConnectionArray,
		}, &resource.IntegrationTypeMetricTrendSummaryResult{
			TotalResourceCount: totalCount,
			IntegrationTypes:   perConnectorArray,
		}, nil
}

func (j *Job) DoAssetMetric(ctx context.Context, jq *jq.JobQueue, steampipeDB *steampipe.Database, encodedResourceCollectionFilters map[string]string, integrationClient integrationClient.IntegrationServiceClient, sinkClient esSinkClient.EsSinkServiceClient, inventoryClient inventoryClient.InventoryServiceClient, logger *zap.Logger, metric db.AnalyticMetric, integrationCache map[string]integrationApi.Integration, startTime time.Time, status []describeApi.DescribeStatus, conf config.WorkerConfig) error {
	connectionMetricTrendSummary := resource.IntegrationMetricTrendSummary{
		EvaluatedAt:         startTime.UnixMilli(),
		Date:                startTime.Format("2006-01-02"),
		Month:               startTime.Format("2006-01"),
		Year:                startTime.Format("2006"),
		MetricID:            metric.ID,
		MetricName:          metric.Name,
		Integrations:        nil,
		ResourceCollections: nil,
	}
	connectorMetricTrendSummary := resource.IntegrationTypeMetricTrendSummary{
		EvaluatedAt:         startTime.UnixMilli(),
		Date:                startTime.Format("2006-01-02"),
		Month:               startTime.Format("2006-01"),
		Year:                startTime.Format("2006"),
		MetricID:            metric.ID,
		MetricName:          metric.Name,
		IntegrationTypes:    nil,
		ResourceCollections: nil,
	}
	if len(encodedResourceCollectionFilters) > 0 {
		connectionMetricTrendSummary.ResourceCollections = make(map[string]resource.IntegrationMetricTrendSummaryResult)
		connectorMetricTrendSummary.ResourceCollections = make(map[string]resource.IntegrationTypeMetricTrendSummaryResult)

		for rcId, encodedFilter := range encodedResourceCollectionFilters {
			err := steampipeDB.SetConfigTableValue(ctx, steampipe.OpenGovernanceConfigKeyResourceCollectionFilters, encodedFilter)
			if err != nil {
				logger.Error("failed to set steampipe context config for resource collection filters", zap.Error(err),
					zap.String("resource_collection", rcId))
				return err
			}
			perConnection, perConnector, err := j.DoSingleAssetMetric(ctx, logger, steampipeDB, metric, integrationCache, status, integrationClient, inventoryClient)
			if err != nil {
				logger.Error("failed to do single asset metric for rc", zap.Error(err), zap.String("metric", metric.ID), zap.String("resource_collection_filters", encodedFilter))
				return err
			}
			connectionMetricTrendSummary.ResourceCollections[rcId] = *perConnection
			connectorMetricTrendSummary.ResourceCollections[rcId] = *perConnector
		}
	} else {
		err := steampipeDB.UnsetConfigTableValue(ctx, steampipe.OpenGovernanceConfigKeyResourceCollectionFilters)
		if err != nil {
			logger.Error("failed to unset steampipe context config for resource collection filters", zap.Error(err))
			return err
		}
		perConnection, perConnector, err := j.DoSingleAssetMetric(ctx, logger, steampipeDB, metric, integrationCache, status, integrationClient, inventoryClient)
		if err != nil {
			logger.Error("failed to do single asset metric", zap.Error(err), zap.String("metric", metric.ID))
			return err
		}
		connectionMetricTrendSummary.Integrations = perConnection
		connectorMetricTrendSummary.IntegrationTypes = perConnector
	}

	keys, idx := connectionMetricTrendSummary.KeysAndIndex()
	connectionMetricTrendSummary.EsID = es.HashOf(keys...)
	connectionMetricTrendSummary.EsIndex = idx

	keys, idx = connectorMetricTrendSummary.KeysAndIndex()
	connectorMetricTrendSummary.EsID = es.HashOf(keys...)
	connectorMetricTrendSummary.EsIndex = idx

	msgs := []es.Doc{
		connectionMetricTrendSummary,
		connectorMetricTrendSummary,
	}

	if _, err := sinkClient.Ingest(&httpclient.Context{UserRole: authApi.AdminRole}, msgs); err != nil {
		logger.Error("failed to send to ingest", zap.Error(err))
		return err
	}
	logger.Info("done sending result to elastic", zap.String("metric", metric.ID), zap.Bool("isOpenSearch", conf.ElasticSearch.IsOpenSearch))

	return nil
}

func (j *Job) DoSpendMetric(ctx context.Context, jq *jq.JobQueue, steampipeDB *steampipe.Database, integrationClient integrationClient.IntegrationServiceClient, sinkClient esSinkClient.EsSinkServiceClient, inventoryClient inventoryClient.InventoryServiceClient, logger *zap.Logger, metric db.AnalyticMetric, connectionCache map[string]integrationApi.Integration, status []describeApi.DescribeStatus, conf config.WorkerConfig) error {
	connectionResultMap := map[string]spend.IntegrationMetricTrendSummary{}
	connectorResultMap := map[string]spend.ConnectorMetricTrendSummary{}

	query := metric.Query

	logger.Info("spend ==== ", zap.String("query", query))

	var res *steampipe.Result
	var err error

	if metric.Engine == db.QueryEngine_cloudqlRego {
		ctx2 := &httpclient.Context{UserRole: authApi.AdminRole}
		ctx2.Ctx = ctx
		var engine inventoryApi.QueryEngine
		engine = inventoryApi.QueryEngine_cloudqlRego
		results, err := inventoryClient.RunQuery(ctx2, inventoryApi.RunQueryRequest{
			Page: inventoryApi.Page{
				No:   1,
				Size: 1000,
			},
			Engine: &engine,
			Query:  &metric.Query,
			Sorts:  nil,
		})
		if err != nil {
			return err
		}
		res = &steampipe.Result{
			Headers: results.Headers,
			Data:    results.Result,
		}
	} else {
		res, err = steampipeDB.QueryAll(ctx, metric.Query)
		if err != nil {
			return err
		}
	}

	connectorCount := map[string]int64{}
	connectorSuccessCount := map[string]int64{}
	for _, st := range status {
		connectorCount[st.Connector]++
		if st.Status == describeApi.DescribeResourceJobSucceeded {
			connectorSuccessCount[st.Connector]++
		}
	}

	for _, record := range res.Data {
		if len(record) != 3 {
			return fmt.Errorf("invalid query: %s", query)
		}

		integrationID, ok := record[0].(string)
		if !ok {
			return fmt.Errorf("invalid format for integrationID: [%s] %v", reflect.TypeOf(record[0]), record[0])
		}
		date, ok := record[1].(string)
		if !ok {
			return fmt.Errorf("invalid format for date: [%s] %v", reflect.TypeOf(record[1]), record[1])
		}
		sum, ok := record[2].(float64)
		if !ok {
			return fmt.Errorf("invalid format for sum: [%s] %v", reflect.TypeOf(record[2]), record[2])
		}

		var integration *integrationApi.Integration
		if cached, ok := connectionCache[integrationID]; ok {
			integration = &cached
		} else {
			integration, err = integrationClient.GetIntegration(&httpclient.Context{UserRole: authApi.AdminRole}, integrationID)
			if err != nil {
				if strings.Contains(err.Error(), "source not found") {
					logger.Warn("data fro connection found but got source not found", zap.String("integrationID", integrationID))
					continue
				}
				return fmt.Errorf("GetIntegration id=%s err=%v", integrationID, err)
			}
			if integration == nil {
				return fmt.Errorf("integration not found: %s", integrationID)
			}

			connectionCache[integrationID] = *integration
		}

		isJobSuccessful := true
		for _, st := range status {
			if st.ConnectionID == integration.IntegrationID {
				if st.Status == describeApi.DescribeResourceJobFailed || st.Status == describeApi.DescribeResourceJobTimeout {
					isJobSuccessful = false
				}
			}
		}

		if r, err := regexp.Compile("^\\d+$"); err == nil && r.MatchString(date) {
			date = date[:4] + "-" + date[4:6] + "-" + date[6:]
		}

		dateTimestamp, err := time.Parse("2006-01-02", date)
		if err != nil {
			return fmt.Errorf("failed to parse date %s due to %v", date, err)
		}

		y, m, d := dateTimestamp.Date()
		startTime := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
		endTime := time.Date(y, m, d, 23, 59, 59, 0, time.UTC)

		if v, ok := connectionResultMap[date]; ok {
			v.TotalCostValue += sum
			if v2, ok2 := v.IntegrationsMap[integration.IntegrationID]; ok2 {
				v2.CostValue += sum
				v2.IsJobSuccessful = isJobSuccessful
				v.IntegrationsMap[integration.IntegrationID] = v2
			} else {
				v.IntegrationsMap[integration.IntegrationID] = spend.PerIntegrationMetricTrendSummary{
					DateEpoch:       dateTimestamp.UnixMilli(),
					IntegrationID:   integration.IntegrationID,
					IntegrationName: integration.Name,
					IntegrationType: integration.IntegrationType,
					CostValue:       sum,
					IsJobSuccessful: isJobSuccessful,
				}
			}
			connectionResultMap[date] = v
		} else {
			vn := spend.IntegrationMetricTrendSummary{
				Date:           dateTimestamp.Format("2006-01-02"),
				DateEpoch:      dateTimestamp.UnixMilli(),
				Month:          dateTimestamp.Format("2006-01"),
				Year:           dateTimestamp.Format("2006"),
				MetricID:       metric.ID,
				MetricName:     metric.Name,
				PeriodStart:    startTime.UnixMilli(),
				PeriodEnd:      endTime.UnixMilli(),
				EvaluatedAt:    time.Now().UnixMilli(),
				TotalCostValue: sum,
				IntegrationsMap: map[string]spend.PerIntegrationMetricTrendSummary{
					integration.IntegrationID: {
						DateEpoch:       dateTimestamp.UnixMilli(),
						IntegrationID:   integration.IntegrationID,
						IntegrationName: integration.Name,
						IntegrationType: integration.IntegrationType,
						CostValue:       sum,
						IsJobSuccessful: isJobSuccessful,
					},
				},
			}
			connectionResultMap[date] = vn
		}

		if v, ok := connectorResultMap[date]; ok {
			v.TotalCostValue += sum
			if v2, ok2 := v.ConnectorsMap[integration.IntegrationType.String()]; ok2 {
				v2.CostValue += sum
				v2.TotalConnections = connectorCount[integration.IntegrationType.String()]
				v2.TotalSuccessfulConnections = connectorSuccessCount[integration.IntegrationType.String()]
				v.ConnectorsMap[integration.IntegrationType.String()] = v2
			} else {
				v.ConnectorsMap[integration.IntegrationType.String()] = spend.PerConnectorMetricTrendSummary{
					DateEpoch:                  dateTimestamp.UnixMilli(),
					Connector:                  integration.IntegrationType,
					CostValue:                  sum,
					TotalConnections:           connectorCount[integration.IntegrationType.String()],
					TotalSuccessfulConnections: connectorSuccessCount[integration.IntegrationType.String()],
				}
			}
			connectorResultMap[date] = v
		} else {
			vn := spend.ConnectorMetricTrendSummary{
				Date:        dateTimestamp.Format("2006-01-02"),
				DateEpoch:   dateTimestamp.UnixMilli(),
				Month:       dateTimestamp.Format("2006-01"),
				Year:        dateTimestamp.Format("2006"),
				MetricID:    metric.ID,
				MetricName:  metric.Name,
				PeriodStart: startTime.UnixMilli(),
				PeriodEnd:   endTime.UnixMilli(),
				EvaluatedAt: time.Now().UnixMilli(),

				TotalCostValue: sum,
				ConnectorsMap: map[string]spend.PerConnectorMetricTrendSummary{
					integration.IntegrationType.String(): {
						DateEpoch:                  dateTimestamp.UnixMilli(),
						Connector:                  integration.IntegrationType,
						CostValue:                  sum,
						TotalConnections:           connectorCount[integration.IntegrationType.String()],
						TotalSuccessfulConnections: connectorSuccessCount[integration.IntegrationType.String()],
					},
				},
			}
			connectorResultMap[date] = vn
		}
	}
	var msgs []es.Doc
	for _, item := range connectionResultMap {
		for _, v := range item.IntegrationsMap {
			item.Integrations = append(item.Integrations, v)
		}
		keys, idx := item.KeysAndIndex()
		item.EsID = es.HashOf(keys...)
		item.EsIndex = idx

		msgs = append(msgs, item)
	}
	for _, item := range connectorResultMap {
		for _, v := range item.ConnectorsMap {
			item.IntegrationTypes = append(item.IntegrationTypes, v)
		}
		keys, idx := item.KeysAndIndex()
		item.EsID = es.HashOf(keys...)
		item.EsIndex = idx

		msgs = append(msgs, item)
	}

	if _, err := sinkClient.Ingest(&httpclient.Context{UserRole: authApi.AdminRole}, msgs); err != nil {
		logger.Error("failed to send to ingest", zap.Error(err))
		return err
	}
	logger.Info("done with spend metric",
		zap.String("metric", metric.ID),
		zap.Int("connector_count", len(connectorResultMap)),
		zap.Int("integration_count", len(connectionResultMap)))
	return nil
}
