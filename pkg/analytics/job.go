package analytics

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	inventoryApi "github.com/kaytu-io/kaytu-engine/pkg/inventory/api"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	authApi "github.com/kaytu-io/kaytu-util/pkg/api"
	shared_entities "github.com/kaytu-io/kaytu-util/pkg/api/shared-entities"
	esSinkClient "github.com/kaytu-io/kaytu-util/pkg/es/ingest/client"
	"github.com/kaytu-io/kaytu-util/pkg/httpclient"
	"github.com/kaytu-io/kaytu-util/pkg/jq"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"math"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/analytics/api"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics/config"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics/db"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics/es/resource"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics/es/spend"
	describeApi "github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	describeClient "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	inventoryClient "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	onboardApi "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-util/pkg/es"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
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
	onboardClient onboardClient.OnboardServiceClient,
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
		ctx2 := &httpclient.Context{UserRole: authApi.InternalRole}
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

	err := steampipeConn.SetConfigTableValue(ctx, steampipe.KaytuConfigKeyAccountID, "all")
	if err != nil {
		logger.Error("failed to set steampipe context config for account id", zap.Error(err), zap.String("account_id", "all"))
		return fail(err)
	}
	defer steampipeConn.UnsetConfigTableValue(ctx, steampipe.KaytuConfigKeyAccountID)

	err = steampipeConn.SetConfigTableValue(ctx, steampipe.KaytuConfigKeyClientType, "analytics")
	if err != nil {
		logger.Error("failed to set steampipe context config for client type", zap.Error(err), zap.String("client_type", "analytics"))
		return fail(err)
	}
	defer steampipeConn.UnsetConfigTableValue(ctx, steampipe.KaytuConfigKeyClientType)

	if err := j.Run(ctx, jq, db, encodedResourceCollectionFilters, steampipeConn, schedulerClient, onboardClient, sinkClient, inventoryClient, logger, config); err != nil {
		fail(err)
	}

	if config.DoTelemetry {
		// send telemetry
		j.SendTelemetry(ctx, logger, config, onboardClient, inventoryClient)
	}

	return result
}

func (j *Job) SendTelemetry(ctx context.Context, logger *zap.Logger, workerConfig config.WorkerConfig, onboardClient onboardClient.OnboardServiceClient, inventoryClient inventoryClient.InventoryServiceClient) {
	now := time.Now()

	httpCtx := httpclient.Context{Ctx: ctx, UserRole: authApi.InternalRole}

	req := shared_entities.CspmUsageRequest{
		WorkspaceId:            workerConfig.TelemetryWorkspaceID,
		GatherTimestamp:        now,
		Hostname:               workerConfig.TelemetryHostname,
		AwsAccountCount:        0,
		AzureSubscriptionCount: 0,
		ApproximateSpend:       0,
	}

	connections, err := onboardClient.ListSources(&httpCtx, nil)
	if err != nil {
		logger.Error("failed to list sources", zap.Error(err))
		return
	}
	for _, conn := range connections {
		switch conn.Connector {
		case source.CloudAWS:
			req.AwsAccountCount++
		case source.CloudAzure:
			req.AzureSubscriptionCount++
		}
	}

	connData, err := inventoryClient.ListConnectionsData(&httpCtx, nil, nil,
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

func (j *Job) Run(ctx context.Context, jq *jq.JobQueue, dbc db.Database, encodedResourceCollectionFilters map[string]string, steampipeDB *steampipe.Database, schedulerClient describeClient.SchedulerServiceClient, onboardClient onboardClient.OnboardServiceClient, sinkClient esSinkClient.EsSinkServiceClient, inventoryClient inventoryClient.InventoryServiceClient, logger *zap.Logger, config config.WorkerConfig) error {
	startTime := time.Now()
	metrics, err := dbc.ListMetrics([]db.AnalyticMetricStatus{db.AnalyticMetricStatusActive, db.AnalyticMetricStatusInvisible})
	if err != nil {
		return err
	}

	connectionCache := map[string]onboardApi.Connection{}

	for _, metric := range metrics {
		switch metric.Type {
		case db.MetricTypeAssets:
			s := map[string]describeApi.DescribeStatus{}
			for _, resourceType := range metric.Tables {
				status, err := schedulerClient.GetDescribeStatus(&httpclient.Context{UserRole: authApi.InternalRole}, resourceType)
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
				onboardClient,
				sinkClient,
				inventoryClient,
				logger,
				metric,
				connectionCache,
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
			awsStatus, err := schedulerClient.GetDescribeStatus(&httpclient.Context{UserRole: authApi.InternalRole}, "AWS::CostExplorer::ByServiceDaily")
			if err != nil {
				return err
			}

			azureStatus, err := schedulerClient.GetDescribeStatus(&httpclient.Context{UserRole: authApi.InternalRole}, "Microsoft.CostManagement/CostByResourceType")
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
				onboardClient,
				sinkClient,
				inventoryClient,
				logger,
				metric,
				connectionCache,
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
	connectionCache map[string]onboardApi.Connection,
	status []describeApi.DescribeStatus,
	onboardClient onboardClient.OnboardServiceClient,
	inventoryClient inventoryClient.InventoryServiceClient) (
	*resource.ConnectionMetricTrendSummaryResult,
	*resource.ConnectorMetricTrendSummaryResult,
	error,
) {
	var res *steampipe.Result
	var err error

	logger.Info("assets ==== ", zap.String("query", metric.Query))
	if metric.Engine == db.QueryEngine_OdysseusRego {
		ctx2 := &httpclient.Context{UserRole: authApi.InternalRole}
		ctx2.Ctx = ctx
		var engine inventoryApi.QueryEngine
		engine = inventoryApi.QueryEngine_OdysseusRego
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
	perConnection := make(map[string]resource.PerConnectionMetricTrendSummary)
	perConnector := make(map[string]resource.PerConnectorMetricTrendSummary)

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

		connectionId, ok := record[0].(string)
		if !ok {
			return nil, nil, fmt.Errorf("invalid format for connectionId: [%s] %v", reflect.TypeOf(record[0]), record[0])
		}

		count, ok := record[2].(int64)
		if !ok {
			return nil, nil, fmt.Errorf("invalid format for count: [%s] %v", reflect.TypeOf(record[2]), record[2])
		}

		var conn *onboardApi.Connection
		if cached, ok := connectionCache[connectionId]; ok {
			conn = &cached
		} else {
			ctx2 := &httpclient.Context{UserRole: authApi.AdminRole}
			ctx2.Ctx = ctx
			conn, err = onboardClient.GetSource(ctx2, connectionId)
			if err != nil {
				if strings.Contains(err.Error(), "source not found") {
					continue
				}
				return nil, nil, fmt.Errorf("GetSource id=%s err=%v", connectionId, err)
			}
			if conn == nil {
				return nil, nil, fmt.Errorf("connection not found: %s", connectionId)
			}

			connectionCache[connectionId] = *conn
		}

		isJobSuccessful := true
		for _, st := range status {
			if st.ConnectionID == conn.ID.String() {
				if st.Status == describeApi.DescribeResourceJobFailed || st.Status == describeApi.DescribeResourceJobTimeout {
					isJobSuccessful = false
				}
			}
		}

		if v, ok := perConnection[conn.ID.String()]; ok {
			v.ResourceCount += int(count)
			perConnection[conn.ID.String()] = v
		} else {
			vn := resource.PerConnectionMetricTrendSummary{
				ConnectionID:    conn.ID.String(),
				ConnectionName:  conn.ConnectionName,
				Connector:       conn.Connector,
				ResourceCount:   int(count),
				IsJobSuccessful: isJobSuccessful,
			}
			perConnection[conn.ID.String()] = vn
		}

		if v, ok := perConnector[conn.Connector.String()]; ok {
			v.ResourceCount += int(count)
			perConnector[conn.Connector.String()] = v
		} else {
			vn := resource.PerConnectorMetricTrendSummary{
				Connector:                  conn.Connector,
				ResourceCount:              int(count),
				TotalConnections:           connectorCount[string(conn.Connector)],
				TotalSuccessfulConnections: connectorSuccessCount[string(conn.Connector)],
			}
			perConnector[conn.Connector.String()] = vn
		}
		totalCount += int(count)
	}
	perConnectionArray := make([]resource.PerConnectionMetricTrendSummary, 0, len(perConnection))
	for _, v := range perConnection {
		perConnectionArray = append(perConnectionArray, v)
	}
	perConnectorArray := make([]resource.PerConnectorMetricTrendSummary, 0, len(perConnector))
	for _, v := range perConnector {
		perConnectorArray = append(perConnectorArray, v)
	}
	logger.Info("assets ==== ", zap.String("metric_id", metric.ID), zap.Int("totalCount", totalCount))

	return &resource.ConnectionMetricTrendSummaryResult{
			TotalResourceCount: totalCount,
			Connections:        perConnectionArray,
		}, &resource.ConnectorMetricTrendSummaryResult{
			TotalResourceCount: totalCount,
			Connectors:         perConnectorArray,
		}, nil
}

func (j *Job) DoAssetMetric(ctx context.Context, jq *jq.JobQueue, steampipeDB *steampipe.Database, encodedResourceCollectionFilters map[string]string, onboardClient onboardClient.OnboardServiceClient, sinkClient esSinkClient.EsSinkServiceClient, inventoryClient inventoryClient.InventoryServiceClient, logger *zap.Logger, metric db.AnalyticMetric, connectionCache map[string]onboardApi.Connection, startTime time.Time, status []describeApi.DescribeStatus, conf config.WorkerConfig) error {
	connectionMetricTrendSummary := resource.ConnectionMetricTrendSummary{
		EvaluatedAt:         startTime.UnixMilli(),
		Date:                startTime.Format("2006-01-02"),
		Month:               startTime.Format("2006-01"),
		Year:                startTime.Format("2006"),
		MetricID:            metric.ID,
		MetricName:          metric.Name,
		Connections:         nil,
		ResourceCollections: nil,
	}
	connectorMetricTrendSummary := resource.ConnectorMetricTrendSummary{
		EvaluatedAt:         startTime.UnixMilli(),
		Date:                startTime.Format("2006-01-02"),
		Month:               startTime.Format("2006-01"),
		Year:                startTime.Format("2006"),
		MetricID:            metric.ID,
		MetricName:          metric.Name,
		Connectors:          nil,
		ResourceCollections: nil,
	}
	if len(encodedResourceCollectionFilters) > 0 {
		connectionMetricTrendSummary.ResourceCollections = make(map[string]resource.ConnectionMetricTrendSummaryResult)
		connectorMetricTrendSummary.ResourceCollections = make(map[string]resource.ConnectorMetricTrendSummaryResult)

		for rcId, encodedFilter := range encodedResourceCollectionFilters {
			err := steampipeDB.SetConfigTableValue(ctx, steampipe.KaytuConfigKeyResourceCollectionFilters, encodedFilter)
			if err != nil {
				logger.Error("failed to set steampipe context config for resource collection filters", zap.Error(err),
					zap.String("resource_collection", rcId))
				return err
			}
			perConnection, perConnector, err := j.DoSingleAssetMetric(ctx, logger, steampipeDB, metric, connectionCache, status, onboardClient, inventoryClient)
			if err != nil {
				logger.Error("failed to do single asset metric for rc", zap.Error(err), zap.String("metric", metric.ID), zap.String("resource_collection_filters", encodedFilter))
				return err
			}
			connectionMetricTrendSummary.ResourceCollections[rcId] = *perConnection
			connectorMetricTrendSummary.ResourceCollections[rcId] = *perConnector
		}
	} else {
		err := steampipeDB.UnsetConfigTableValue(ctx, steampipe.KaytuConfigKeyResourceCollectionFilters)
		if err != nil {
			logger.Error("failed to unset steampipe context config for resource collection filters", zap.Error(err))
			return err
		}
		perConnection, perConnector, err := j.DoSingleAssetMetric(ctx, logger, steampipeDB, metric, connectionCache, status, onboardClient, inventoryClient)
		if err != nil {
			logger.Error("failed to do single asset metric", zap.Error(err), zap.String("metric", metric.ID))
			return err
		}
		connectionMetricTrendSummary.Connections = perConnection
		connectorMetricTrendSummary.Connectors = perConnector
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

	if _, err := sinkClient.Ingest(&httpclient.Context{UserRole: authApi.InternalRole}, msgs); err != nil {
		logger.Error("failed to send to ingest", zap.Error(err))
		return err
	}
	logger.Info("done sending result to elastic", zap.String("metric", metric.ID), zap.Bool("isOpenSearch", conf.ElasticSearch.IsOpenSearch))

	return nil
}

func (j *Job) DoSpendMetric(ctx context.Context, jq *jq.JobQueue, steampipeDB *steampipe.Database, onboardClient onboardClient.OnboardServiceClient, sinkClient esSinkClient.EsSinkServiceClient, inventoryClient inventoryClient.InventoryServiceClient, logger *zap.Logger, metric db.AnalyticMetric, connectionCache map[string]onboardApi.Connection, status []describeApi.DescribeStatus, conf config.WorkerConfig) error {
	connectionResultMap := map[string]spend.ConnectionMetricTrendSummary{}
	connectorResultMap := map[string]spend.ConnectorMetricTrendSummary{}

	query := metric.Query

	logger.Info("spend ==== ", zap.String("query", query))

	var res *steampipe.Result
	var err error

	if metric.Engine == db.QueryEngine_OdysseusRego {
		ctx2 := &httpclient.Context{UserRole: authApi.InternalRole}
		ctx2.Ctx = ctx
		var engine inventoryApi.QueryEngine
		engine = inventoryApi.QueryEngine_OdysseusRego
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

		connectionID, ok := record[0].(string)
		if !ok {
			return fmt.Errorf("invalid format for connectionID: [%s] %v", reflect.TypeOf(record[0]), record[0])
		}
		date, ok := record[1].(string)
		if !ok {
			return fmt.Errorf("invalid format for date: [%s] %v", reflect.TypeOf(record[1]), record[1])
		}
		sum, ok := record[2].(float64)
		if !ok {
			return fmt.Errorf("invalid format for sum: [%s] %v", reflect.TypeOf(record[2]), record[2])
		}

		var conn *onboardApi.Connection
		if cached, ok := connectionCache[connectionID]; ok {
			conn = &cached
		} else {
			conn, err = onboardClient.GetSource(&httpclient.Context{UserRole: authApi.AdminRole}, connectionID)
			if err != nil {
				if strings.Contains(err.Error(), "source not found") {
					logger.Warn("data fro connection found but got source not found", zap.String("connectionID", connectionID))
					continue
				}
				return fmt.Errorf("GetSource id=%s err=%v", connectionID, err)
			}
			if conn == nil {
				return fmt.Errorf("connection not found: %s", connectionID)
			}

			connectionCache[connectionID] = *conn
		}

		isJobSuccessful := true
		for _, st := range status {
			if st.ConnectionID == conn.ID.String() {
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
			if v2, ok2 := v.ConnectionsMap[conn.ID.String()]; ok2 {
				v2.CostValue += sum
				v2.IsJobSuccessful = isJobSuccessful
				v.ConnectionsMap[conn.ID.String()] = v2
			} else {
				v.ConnectionsMap[conn.ID.String()] = spend.PerConnectionMetricTrendSummary{
					DateEpoch:       dateTimestamp.UnixMilli(),
					ConnectionID:    conn.ID.String(),
					ConnectionName:  conn.ConnectionName,
					Connector:       conn.Connector,
					CostValue:       sum,
					IsJobSuccessful: isJobSuccessful,
				}
			}
			connectionResultMap[date] = v
		} else {
			vn := spend.ConnectionMetricTrendSummary{
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
				ConnectionsMap: map[string]spend.PerConnectionMetricTrendSummary{
					conn.ID.String(): {
						DateEpoch:       dateTimestamp.UnixMilli(),
						ConnectionID:    conn.ID.String(),
						ConnectionName:  conn.ConnectionName,
						Connector:       conn.Connector,
						CostValue:       sum,
						IsJobSuccessful: isJobSuccessful,
					},
				},
			}
			connectionResultMap[date] = vn
		}

		if v, ok := connectorResultMap[date]; ok {
			v.TotalCostValue += sum
			if v2, ok2 := v.ConnectorsMap[conn.Connector.String()]; ok2 {
				v2.CostValue += sum
				v2.TotalConnections = connectorCount[string(conn.Connector)]
				v2.TotalSuccessfulConnections = connectorSuccessCount[string(conn.Connector)]
				v.ConnectorsMap[conn.Connector.String()] = v2
			} else {
				v.ConnectorsMap[conn.Connector.String()] = spend.PerConnectorMetricTrendSummary{
					DateEpoch:                  dateTimestamp.UnixMilli(),
					Connector:                  conn.Connector,
					CostValue:                  sum,
					TotalConnections:           connectorCount[string(conn.Connector)],
					TotalSuccessfulConnections: connectorSuccessCount[string(conn.Connector)],
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
					conn.Connector.String(): {
						DateEpoch:                  dateTimestamp.UnixMilli(),
						Connector:                  conn.Connector,
						CostValue:                  sum,
						TotalConnections:           connectorCount[string(conn.Connector)],
						TotalSuccessfulConnections: connectorSuccessCount[string(conn.Connector)],
					},
				},
			}
			connectorResultMap[date] = vn
		}
	}
	var msgs []es.Doc
	for _, item := range connectionResultMap {
		for _, v := range item.ConnectionsMap {
			item.Connections = append(item.Connections, v)
		}
		keys, idx := item.KeysAndIndex()
		item.EsID = es.HashOf(keys...)
		item.EsIndex = idx

		msgs = append(msgs, item)
	}
	for _, item := range connectorResultMap {
		for _, v := range item.ConnectorsMap {
			item.Connectors = append(item.Connectors, v)
		}
		keys, idx := item.KeysAndIndex()
		item.EsID = es.HashOf(keys...)
		item.EsIndex = idx

		msgs = append(msgs, item)
	}

	if _, err := sinkClient.Ingest(&httpclient.Context{UserRole: authApi.InternalRole}, msgs); err != nil {
		logger.Error("failed to send to ingest", zap.Error(err))
		return err
	}
	logger.Info("done with spend metric",
		zap.String("metric", metric.ID),
		zap.Int("connector_count", len(connectorResultMap)),
		zap.Int("connection_count", len(connectionResultMap)))
	return nil
}
