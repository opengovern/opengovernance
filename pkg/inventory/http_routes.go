package inventory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/demo"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/analytics/es/spend"

	"github.com/google/uuid"
	awsSteampipe "github.com/kaytu-io/kaytu-aws-describer/pkg/steampipe"
	azureSteampipe "github.com/kaytu-io/kaytu-azure-describer/pkg/steampipe"
	analyticsDB "github.com/kaytu-io/kaytu-engine/pkg/analytics/db"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics/es/resource"
	authApi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	insight "github.com/kaytu-io/kaytu-engine/pkg/insight/es"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	inventoryApi "github.com/kaytu-io/kaytu-engine/pkg/inventory/api"
	"github.com/kaytu-io/kaytu-engine/pkg/inventory/es"
	"github.com/kaytu-io/kaytu-engine/pkg/inventory/internal"
	es3 "github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	EsFetchPageSize = 10000
	MaxConns        = 100
	KafkaPageSize   = 5000
)

const (
	ConnectionIdParam    = "connectionId"
	ConnectionGroupParam = "connectionGroup"
)

func (h *HttpHandler) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	queryV1 := v1.Group("/query")
	queryV1.GET("", httpserver.AuthorizeHandler(h.ListQueries, authApi.ViewerRole))
	queryV1.POST("/run", httpserver.AuthorizeHandler(h.RunQuery, authApi.ViewerRole))
	queryV1.GET("/run/history", httpserver.AuthorizeHandler(h.GetRecentRanQueries, authApi.ViewerRole))

	v2 := e.Group("/api/v2")

	resourcesV2 := v2.Group("/resources")
	resourcesV2.GET("/count", httpserver.AuthorizeHandler(h.CountResources, authApi.ViewerRole))
	resourcesV2.GET("/metric/:resourceType", httpserver.AuthorizeHandler(h.GetResourceTypeMetricsHandler, authApi.ViewerRole))

	analyticsV2 := v2.Group("/analytics")
	analyticsV2.GET("/metrics/list", httpserver.AuthorizeHandler(h.ListMetrics, authApi.ViewerRole))
	analyticsV2.GET("/metrics/:metric_id", httpserver.AuthorizeHandler(h.GetMetric, authApi.ViewerRole))

	analyticsV2.GET("/metric", httpserver.AuthorizeHandler(h.ListAnalyticsMetricsHandler, authApi.ViewerRole))
	analyticsV2.GET("/tag", httpserver.AuthorizeHandler(h.ListAnalyticsTags, authApi.ViewerRole))
	analyticsV2.GET("/trend", httpserver.AuthorizeHandler(h.ListAnalyticsMetricTrend, authApi.ViewerRole))
	analyticsV2.GET("/composition/:key", httpserver.AuthorizeHandler(h.ListAnalyticsComposition, authApi.ViewerRole))
	analyticsV2.GET("/categories", httpserver.AuthorizeHandler(h.ListAnalyticsCategories, authApi.ViewerRole))
	analyticsV2.GET("/table", httpserver.AuthorizeHandler(h.GetAssetsTable, authApi.ViewerRole))

	analyticsSpend := analyticsV2.Group("/spend")
	analyticsSpend.GET("/metric", httpserver.AuthorizeHandler(h.ListAnalyticsSpendMetricsHandler, authApi.ViewerRole))
	analyticsSpend.GET("/composition", httpserver.AuthorizeHandler(h.ListAnalyticsSpendComposition, authApi.ViewerRole))
	analyticsSpend.GET("/trend", httpserver.AuthorizeHandler(h.GetAnalyticsSpendTrend, authApi.ViewerRole))
	analyticsSpend.GET("/metrics/trend", httpserver.AuthorizeHandler(h.GetAnalyticsSpendMetricsTrend, authApi.ViewerRole))
	analyticsSpend.GET("/table", httpserver.AuthorizeHandler(h.GetSpendTable, authApi.ViewerRole))

	connectionsV2 := v2.Group("/connections")
	connectionsV2.GET("/data", httpserver.AuthorizeHandler(h.ListConnectionsData, authApi.ViewerRole))
	connectionsV2.GET("/data/:connectionId", httpserver.AuthorizeHandler(h.GetConnectionData, authApi.ViewerRole))

	insightsV2 := v2.Group("/insights")
	insightsV2.GET("", httpserver.AuthorizeHandler(h.ListInsightResults, authApi.ViewerRole))
	insightsV2.GET("/:insightId/trend", httpserver.AuthorizeHandler(h.GetInsightTrendResults, authApi.ViewerRole))
	insightsV2.GET("/:insightId", httpserver.AuthorizeHandler(h.GetInsightResult, authApi.ViewerRole))

	metadata := v2.Group("/metadata")
	metadata.GET("/resourcetype", httpserver.AuthorizeHandler(h.ListResourceTypeMetadata, authApi.ViewerRole))

	//v1.GET("/migrate-analytics", httpserver.AuthorizeHandler(h.MigrateAnalytics, authApi.AdminRole))
	v1.GET("/migrate-spend", httpserver.AuthorizeHandler(h.MigrateSpend, authApi.AdminRole))
}

var tracer = otel.Tracer("new_inventory")

func (h *HttpHandler) getConnectionIdFilterFromParams(ctx echo.Context) ([]string, error) {
	connectionIds := httpserver.QueryArrayParam(ctx, ConnectionIdParam)
	connectionGroup := httpserver.QueryArrayParam(ctx, ConnectionGroupParam)
	if len(connectionIds) == 0 && len(connectionGroup) == 0 {
		return nil, nil
	}

	if len(connectionIds) > 0 && len(connectionGroup) > 0 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "connectionId and connectionGroup cannot be used together")
	}

	if len(connectionIds) > 0 {
		return connectionIds, nil
	}

	connectionMap := map[string]bool{}
	for _, connectionGroupID := range connectionGroup {
		connectionGroupObj, err := h.onboardClient.GetConnectionGroup(&httpclient.Context{UserRole: authApi.KaytuAdminRole}, connectionGroupID)
		if err != nil {
			return nil, err
		}
		for _, connectionID := range connectionGroupObj.ConnectionIds {
			connectionMap[connectionID] = true
		}
	}
	connectionIds = make([]string, 0, len(connectionMap))
	for connectionID := range connectionMap {
		connectionIds = append(connectionIds, connectionID)
	}
	if len(connectionIds) == 0 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "connectionGroup(s) do not have any connections")
	}

	return connectionIds, nil
}

func (h *HttpHandler) MigrateAnalytics(ctx echo.Context) error {
	for i := 0; i < 1000; i++ {
		err := h.MigrateAnalyticsPart(i)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *HttpHandler) MigrateAnalyticsPart(summarizerJobID int) error {
	aDB := analyticsDB.NewDatabase(h.db.orm)

	connectionMap := map[string]resource.ConnectionMetricTrendSummary{}
	connectorMap := map[string]resource.ConnectorMetricTrendSummary{}

	resourceTypeMetricIDCache := map[string]analyticsDB.AnalyticMetric{}

	cctx := context.Background()

	pagination, err := es.NewConnectionResourceTypePaginator(
		h.client,
		[]kaytu.BoolFilter{
			kaytu.NewTermFilter("report_type", string(es3.ResourceTypeTrendConnectionSummary)),
			kaytu.NewTermFilter("summarize_job_id", fmt.Sprintf("%d", summarizerJobID)),
		},
		nil,
	)
	if err != nil {
		return err
	}

	var docs []kafka.Doc
	for {
		if !pagination.HasNext() {
			fmt.Println("MigrateAnalytics = page done", summarizerJobID)
			break
		}

		fmt.Println("MigrateAnalytics = ask page", summarizerJobID)
		page, err := pagination.NextPage(cctx)
		if err != nil {
			return err
		}
		fmt.Println("MigrateAnalytics = next page", summarizerJobID)

		for _, hit := range page {
			connectionID, err := uuid.Parse(hit.SourceID)
			if err != nil {
				return err
			}

			conn, err := h.onboardClient.GetSource(&httpclient.Context{UserRole: authApi.AdminRole}, connectionID.String())
			if err != nil {
				return err
			}

			if conn == nil {
				fmt.Println("failed to find source", connectionID)
				continue
			}

			var metricItem analyticsDB.AnalyticMetric

			if v, ok := resourceTypeMetricIDCache[hit.ResourceType]; ok {
				metricItem = v
			} else {
				// tracer :
				_, span1 := tracer.Start(cctx, "new_GetMetric", trace.WithSpanKind(trace.SpanKindServer))
				span1.SetName("new_GetMetric")

				metric, err := aDB.GetMetric(analyticsDB.MetricTypeAssets, hit.ResourceType)
				if err != nil {
					span1.RecordError(err)
					span1.SetStatus(codes.Error, err.Error())

					return err
				}
				span1.AddEvent("information", trace.WithAttributes(
					attribute.String("metricID", metric.ID),
				))
				span1.End()
				if metric == nil {
					return fmt.Errorf("resource type %s not found", hit.ResourceType)
				}

				resourceTypeMetricIDCache[hit.ResourceType] = *metric
				metricItem = *metric
			}

			if metricItem.ID == "" {
				continue
			}

			connection := resource.ConnectionMetricTrendSummary{
				ConnectionID:   connectionID,
				ConnectionName: conn.ConnectionName,
				Connector:      hit.SourceType,
				EvaluatedAt:    hit.DescribedAt,
				Date:           time.UnixMilli(hit.DescribedAt).Format("2006-01-02"),
				Month:          time.UnixMilli(hit.DescribedAt).Format("2006-01"),
				Year:           time.UnixMilli(hit.DescribedAt).Format("2006"),
				MetricID:       metricItem.ID,
				MetricName:     metricItem.Name,
				ResourceCount:  hit.ResourceCount,
			}
			key := fmt.Sprintf("%s-%s-%d", connectionID.String(), metricItem.ID, hit.SummarizeJobID)
			if v, ok := connectionMap[key]; ok {
				v.ResourceCount += connection.ResourceCount
				connectionMap[key] = v
			} else {
				connectionMap[key] = connection
			}

			connector := resource.ConnectorMetricTrendSummary{
				Connector:     hit.SourceType,
				EvaluatedAt:   hit.DescribedAt,
				Date:          time.UnixMilli(hit.DescribedAt).Format("2006-01-02"),
				Month:         time.UnixMilli(hit.DescribedAt).Format("2006-01"),
				Year:          time.UnixMilli(hit.DescribedAt).Format("2006"),
				MetricID:      metricItem.ID,
				MetricName:    metricItem.Name,
				ResourceCount: hit.ResourceCount,
			}
			key = fmt.Sprintf("%s-%s-%d", connector.Connector, metricItem.ID, hit.SummarizeJobID)
			if v, ok := connectorMap[key]; ok {
				v.ResourceCount += connector.ResourceCount
				connectorMap[key] = v
			} else {
				connectorMap[key] = connector
			}
		}
	}

	for _, c := range connectionMap {
		docs = append(docs, c)
	}

	for _, c := range connectorMap {
		docs = append(docs, c)
	}

	for startPageIdx := 0; startPageIdx < len(docs); startPageIdx += KafkaPageSize {
		docsToSend := docs[startPageIdx:min(startPageIdx+KafkaPageSize, len(docs))]
		err = kafka.DoSend(h.kafkaProducer, "cloud-resources", -1, docsToSend, h.logger, nil)
		if err != nil {
			h.logger.Warn("failed to send to kafka", zap.Error(err), zap.Int("len", h.kafkaProducer.Len()))
			continue
		}
	}
	return nil
}

func (h *HttpHandler) MigrateSpend(ctx echo.Context) error {
	connectorMap := map[string]spend.ConnectorMetricTrendSummary{}

	startJobId := 0
	if jobIdStr := ctx.QueryParam("startJobId"); jobIdStr != "" {
		jobId, err := strconv.ParseInt(jobIdStr, 10, 64)
		if err != nil {
			return err
		}
		startJobId = int(jobId)
	}

	maxJobID := startJobId + 1000
	for i := startJobId; i < maxJobID; i++ {
		cm, err := h.MigrateSpendPart(i, true)
		if err != nil {
			return err
		}
		if len(cm) > 0 {
			maxJobID = i + 1000
		}

		for key, newValue := range cm {
			if v, ok := connectorMap[key]; ok {
				v.CostValue += newValue.CostValue
				connectorMap[key] = v
			} else {
				connectorMap[key] = newValue
			}
		}

		cm, err = h.MigrateSpendPart(i, false)
		if err != nil {
			return err
		}
		if len(cm) > 0 {
			maxJobID = i + 1000
		}

		for key, newValue := range cm {
			if v, ok := connectorMap[key]; ok {
				v.CostValue += newValue.CostValue
				connectorMap[key] = v
			} else {
				connectorMap[key] = newValue
			}
		}
	}

	var docs []kafka.Doc
	for _, c := range connectorMap {
		docs = append(docs, c)
	}

	for startPageIdx := 0; startPageIdx < len(docs); startPageIdx += KafkaPageSize {
		docsToSend := docs[startPageIdx:min(startPageIdx+KafkaPageSize, len(docs))]
		err := kafka.DoSend(h.kafkaProducer, "cloud-resources", -1, docsToSend, h.logger, nil)
		if err != nil {
			h.logger.Warn("failed to send to kafka", zap.Error(err), zap.Int("len", h.kafkaProducer.Len()))
			continue
		}
	}
	return nil
}

type ExistFilter struct {
	field string
}

func NewExistFilter(field string) kaytu.BoolFilter {
	return ExistFilter{
		field: field,
	}
}
func (t ExistFilter) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"exists": map[string]string{
			"field": t.field,
		},
	})
}
func (t ExistFilter) IsBoolFilter() {}

func (h *HttpHandler) MigrateSpendPart(summarizerJobID int, isAWS bool) (map[string]spend.ConnectorMetricTrendSummary, error) {
	aDB := analyticsDB.NewDatabase(h.db.orm)
	connectionMap := map[string]spend.ConnectionMetricTrendSummary{}
	connectorMap := map[string]spend.ConnectorMetricTrendSummary{}

	cctx := context.Background()
	var boolFilters []kaytu.BoolFilter
	if isAWS {
		boolFilters = []kaytu.BoolFilter{
			kaytu.NewTermFilter("report_type", string(es3.CostServiceSummaryDaily)),
			kaytu.NewTermFilter("summarize_job_id", fmt.Sprintf("%d", summarizerJobID)),
			kaytu.NewTermFilter("source_type", "AWS"),
			NewExistFilter("cost.Dimension1"),
		}
	} else {
		boolFilters = []kaytu.BoolFilter{
			kaytu.NewTermFilter("report_type", string(es3.CostServiceSummaryDaily)),
			kaytu.NewTermFilter("summarize_job_id", fmt.Sprintf("%d", summarizerJobID)),
			kaytu.NewTermFilter("source_type", "Azure"),
			NewExistFilter("cost.ServiceName"),
		}
	}

	pagination, err := es.NewConnectionCostPaginator(
		h.client,
		boolFilters,
		nil,
	)
	if err != nil {
		return nil, err
	}
	serviceNameMetricCache := map[string]analyticsDB.AnalyticMetric{}

	var docs []kafka.Doc
	for {
		if !pagination.HasNext() {
			fmt.Println("MigrateAnalytics = page done", summarizerJobID)
			break
		}

		fmt.Println("MigrateAnalytics = ask page", summarizerJobID)
		page, err := pagination.NextPage(cctx)
		if err != nil {
			return nil, err
		}
		fmt.Println("MigrateAnalytics = next page", summarizerJobID, len(page))

		for _, hit := range page {
			connectionID, err := uuid.Parse(hit.SourceID)
			if err != nil {
				return nil, err
			}

			var metricID, metricName string
			if v, ok := serviceNameMetricCache[hit.ServiceName]; ok {
				metricID = v.ID
				metricName = v.Name
			} else {
				isMarketplace := false

				costDescriptionJson, _ := json.Marshal(hit.Cost)
				var costDescription map[string]any
				_ = json.Unmarshal(costDescriptionJson, &costDescription)
				if costDescription["Dimension2"] != nil {
					switch costDescription["Dimension2"].(type) {
					case string:
						if costDescription["Dimension2"].(string) == "AWS Marketplace" {
							isMarketplace = true
						}
					}
				}
				if costDescription["PublisherType"] != nil {
					switch costDescription["PublisherType"].(type) {
					case string:
						if costDescription["PublisherType"].(string) == "Marketplace" {
							isMarketplace = true
						}
					}
				}

				if isMarketplace {
					table := "AWS Marketplace"
					if !isAWS {
						table = "Azure Marketplace"
					}
					// tracer :
					_, span1 := tracer.Start(cctx, "new_GetMetric", trace.WithSpanKind(trace.SpanKindServer))
					span1.SetName("new_GetMetric")
					metric, err := aDB.GetMetric(analyticsDB.MetricTypeSpend, table)
					if err != nil {
						span1.RecordError(err)
						span1.SetStatus(codes.Error, err.Error())
						return nil, err
					}
					span1.AddEvent("information", trace.WithAttributes(
						attribute.String("metricID", metric.ID),
					))
					span1.End()
					if metric != nil {
						serviceNameMetricCache[hit.ServiceName] = *metric
						metricID = metric.ID
						metricName = metric.Name
					}
				} else {
					// tracer :
					_, span2 := tracer.Start(cctx, "new_GetMetric", trace.WithSpanKind(trace.SpanKindServer))
					span2.SetName("new_GetMetric")
					metric, err := aDB.GetMetric(analyticsDB.MetricTypeSpend, hit.ServiceName)
					if err != nil {
						span2.RecordError(err)
						span2.SetStatus(codes.Error, err.Error())
						return nil, err
					}
					span2.AddEvent("information", trace.WithAttributes(
						attribute.String("metricID", metric.ID),
					))
					span2.End()
					if metric == nil {
						return nil, fmt.Errorf("GetMetric, table %s not found", hit.ServiceName)
					}
					serviceNameMetricCache[hit.ServiceName] = *metric
					metricID = metric.ID
					metricName = metric.Name
				}
			}

			if metricID == "" {
				fmt.Println(hit.ServiceName, "doesnt have metricID")
				continue
			}

			conn, err := h.onboardClient.GetSource(&httpclient.Context{UserRole: authApi.AdminRole}, hit.SourceID)
			if err != nil {
				fmt.Println(err)
				continue
				//return err
			}

			dateTimestamp := (hit.PeriodStart + hit.PeriodEnd) / 2
			dateStr := time.Unix(dateTimestamp, 0).Format("2006-01-02")
			monthStr := time.Unix(dateTimestamp, 0).Format("2006-01")
			yearStr := time.Unix(dateTimestamp, 0).Format("2006")
			connection := spend.ConnectionMetricTrendSummary{
				ConnectionID:    connectionID,
				ConnectionName:  conn.ConnectionName,
				Connector:       hit.Connector,
				Date:            dateStr,
				DateEpoch:       dateTimestamp * 1000,
				Month:           monthStr,
				Year:            yearStr,
				MetricID:        metricID,
				MetricName:      metricName,
				CostValue:       hit.CostValue,
				PeriodStart:     hit.PeriodStart * 1000,
				PeriodEnd:       hit.PeriodEnd * 1000,
				IsJobSuccessful: true,
			}
			key := fmt.Sprintf("%s-%s-%s", connectionID.String(), metricID, dateStr)
			if v, ok := connectionMap[key]; ok {
				v.CostValue += connection.CostValue
				connectionMap[key] = v
			} else {
				connectionMap[key] = connection
			}

			connector := spend.ConnectorMetricTrendSummary{
				Connector:                  hit.Connector,
				Date:                       dateStr,
				DateEpoch:                  dateTimestamp * 1000,
				Month:                      monthStr,
				Year:                       yearStr,
				MetricID:                   metricID,
				MetricName:                 metricName,
				CostValue:                  hit.CostValue,
				PeriodStart:                hit.PeriodStart * 1000,
				PeriodEnd:                  hit.PeriodEnd * 1000,
				TotalConnections:           0, //TODO
				TotalSuccessfulConnections: 0, //TODO
			}
			key = fmt.Sprintf("%s-%s-%s", connector.Connector, metricID, dateStr)
			if dateStr == "2023-07-05" && metricID == "spend_amazon_elastic_compute_cloud___compute" {
				fmt.Println(key, connector.CostValue)
			}

			if v, ok := connectorMap[key]; ok {
				v.CostValue += connector.CostValue
				connectorMap[key] = v
			} else {
				connectorMap[key] = connector
			}
		}
	}

	for _, c := range connectionMap {
		docs = append(docs, c)
	}

	for startPageIdx := 0; startPageIdx < len(docs); startPageIdx += KafkaPageSize {
		docsToSend := docs[startPageIdx:min(startPageIdx+KafkaPageSize, len(docs))]
		err = kafka.DoSend(h.kafkaProducer, "cloud-resources", -1, docsToSend, h.logger, nil)
		if err != nil {
			h.logger.Warn("failed to send to kafka", zap.Error(err), zap.Int("len", h.kafkaProducer.Len()))
			continue
		}
	}

	return connectorMap, nil
}

func bindValidate(ctx echo.Context, i interface{}) error {
	if err := ctx.Bind(i); err != nil {
		return err
	}

	if err := ctx.Validate(i); err != nil {
		return err
	}

	return nil
}

func (h *HttpHandler) getConnectorTypesFromConnectionIDs(ctx echo.Context, connectorTypes []source.Type, connectionIDs []string) ([]source.Type, error) {
	if len(connectionIDs) == 0 {
		return connectorTypes, nil
	}
	if len(connectorTypes) != 0 {
		return connectorTypes, nil
	}
	connections, err := h.onboardClient.GetSources(httpclient.FromEchoContext(ctx), connectionIDs)
	if err != nil {
		return nil, err
	}

	enabledConnectors := make(map[source.Type]bool)
	for _, connection := range connections {
		enabledConnectors[connection.Connector] = true
	}
	connectorTypes = make([]source.Type, 0, len(enabledConnectors))
	for connectorType := range enabledConnectors {
		connectorTypes = append(connectorTypes, connectorType)
	}

	return connectorTypes, nil
}

func (h *HttpHandler) ListAnalyticsMetrics(ctx context.Context, metricIDs []string, metricType analyticsDB.MetricType, tagMap map[string][]string, connectorTypes []source.Type, connectionIDs []string, minCount int, timeAt time.Time) (int, []inventoryApi.Metric, error) {
	aDB := analyticsDB.NewDatabase(h.db.orm)
	// tracer :
	_, span := tracer.Start(ctx, "new_ListFilteredMetrics", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_ListFilteredMetrics")
	mts, err := aDB.ListFilteredMetrics(tagMap, metricType, metricIDs, connectorTypes, false)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, nil, err
	}
	span.End()

	filteredMetricIDs := make([]string, 0, len(mts))
	for _, metric := range mts {
		filteredMetricIDs = append(filteredMetricIDs, metric.ID)
	}

	var metricIndexed map[string]int
	if len(connectionIDs) > 0 {
		metricIndexed, err = es.FetchConnectionAnalyticMetricCountAtTime(h.client, connectorTypes, connectionIDs, timeAt, filteredMetricIDs, EsFetchPageSize)
	} else {
		metricIndexed, err = es.FetchConnectorAnalyticMetricCountAtTime(h.client, connectorTypes, timeAt, filteredMetricIDs, EsFetchPageSize)
	}
	if err != nil {
		return 0, nil, err
	}

	apiMetrics := make([]inventoryApi.Metric, 0, len(mts))
	totalCount := 0
	for _, metric := range mts {
		apiMetric := inventoryApi.MetricToAPI(metric)
		if count, ok := metricIndexed[metric.ID]; ok && count >= minCount {
			apiMetric.Count = &count
			totalCount += count
		}
		if (minCount == 0) || (apiMetric.Count != nil && *apiMetric.Count >= minCount) {
			apiMetrics = append(apiMetrics, apiMetric)
		}
	}

	return totalCount, apiMetrics, nil
}

// ListAnalyticsMetricsHandler godoc
//
//	@Summary		List analytics metrics
//	@Description	Retrieving list of analytics with metrics of each type based on the given input filters.
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			tag				query		[]string		false	"Key-Value tags in key=value format to filter by"
//	@Param			metricType		query		string			false	"Metric type, default: assets"	Enums(assets, spend)
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by - mutually exclusive with connectionGroup"
//	@Param			connectionGroup	query		[]string		false	"Connection group to filter by - mutually exclusive with connectionId"
//	@Param			metricIDs		query		[]string		false	"Metric IDs"
//	@Param			endTime			query		int64			false	"timestamp for resource count in epoch seconds"
//	@Param			startTime		query		int64			false	"timestamp for resource count change comparison in epoch seconds"
//	@Param			minCount		query		int				false	"Minimum number of resources with this tag value, default 1"
//	@Param			sortBy			query		string			false	"Sort by field - default is count"	Enums(name,count,growth,growth_rate)
//	@Param			pageSize		query		int				false	"page size - default is 20"
//	@Param			pageNumber		query		int				false	"page number - default is 1"
//	@Success		200				{object}	inventoryApi.ListMetricsResponse
//	@Router			/inventory/api/v2/analytics/metric [get]
func (h *HttpHandler) ListAnalyticsMetricsHandler(ctx echo.Context) error {
	var err error
	tagMap := model.TagStringsToTagMap(httpserver.QueryArrayParam(ctx, "tag"))
	metricType := analyticsDB.MetricType(ctx.QueryParam("metricType"))
	if metricType == "" {
		metricType = analyticsDB.MetricTypeAssets
	}
	connectorTypes := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
	if len(connectionIDs) > MaxConns {
		return ctx.JSON(http.StatusBadRequest, "too many connections")
	}
	metricIDs := httpserver.QueryArrayParam(ctx, "metricIDs")

	connectorTypes, err = h.getConnectorTypesFromConnectionIDs(ctx, connectorTypes, connectionIDs)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	endTime := time.Now()
	if endTimeStr := ctx.QueryParam("endTime"); endTimeStr != "" {
		endTimeVal, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "invalid endTime value")
		}
		endTime = time.Unix(endTimeVal, 0)
	}
	startTime := endTime.AddDate(0, 0, -7)
	if startTimeStr := ctx.QueryParam("startTime"); startTimeStr != "" {
		startTimeVal, err := strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "invalid startTime value")
		}
		startTime = time.Unix(startTimeVal, 0)
	}
	minCount := 1
	if minCountStr := ctx.QueryParam("minCount"); minCountStr != "" {
		minCountVal, err := strconv.ParseInt(minCountStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "minCount must be a number")
		}
		minCount = int(minCountVal)
	}
	pageNumber, pageSize, err := utils.PageConfigFromStrings(ctx.QueryParam("pageNumber"), ctx.QueryParam("pageSize"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	sortBy := strings.ToLower(ctx.QueryParam("sortBy"))
	if sortBy == "" {
		sortBy = "count"
	}
	if sortBy != "name" && sortBy != "count" &&
		sortBy != "growth" && sortBy != "growth_rate" {
		return ctx.JSON(http.StatusBadRequest, "invalid sortBy value")
	}

	totalCount, apiMetrics, err := h.ListAnalyticsMetrics(ctx.Request().Context(), metricIDs, metricType, tagMap, connectorTypes, connectionIDs, minCount, endTime)
	if err != nil {
		return err
	}

	apiMetricsMap := make(map[string]inventoryApi.Metric, len(apiMetrics))
	for _, apiMetric := range apiMetrics {
		apiMetricsMap[apiMetric.ID] = apiMetric
	}

	totalOldCount, oldApiMetrics, err := h.ListAnalyticsMetrics(ctx.Request().Context(), metricIDs, metricType, tagMap, connectorTypes, connectionIDs, 0, startTime)
	if err != nil {
		return err
	}
	for _, oldApiMetric := range oldApiMetrics {
		if apiMetric, ok := apiMetricsMap[oldApiMetric.ID]; ok {
			apiMetric.OldCount = oldApiMetric.Count
			apiMetricsMap[oldApiMetric.ID] = apiMetric
		}
	}

	apiMetrics = make([]inventoryApi.Metric, 0, len(apiMetricsMap))
	for _, apiMetric := range apiMetricsMap {
		apiMetrics = append(apiMetrics, apiMetric)
	}

	sort.Slice(apiMetrics, func(i, j int) bool {
		switch sortBy {
		case "name":
			return apiMetrics[i].Name < apiMetrics[j].Name
		case "count":
			if apiMetrics[i].Count == nil && apiMetrics[j].Count == nil {
				break
			}
			if apiMetrics[i].Count == nil {
				return false
			}
			if apiMetrics[j].Count == nil {
				return true
			}
			if *apiMetrics[i].Count != *apiMetrics[j].Count {
				return *apiMetrics[i].Count > *apiMetrics[j].Count
			}
		case "growth":
			diffi := utils.PSub(apiMetrics[i].Count, apiMetrics[i].OldCount)
			diffj := utils.PSub(apiMetrics[j].Count, apiMetrics[j].OldCount)
			if diffi == nil && diffj == nil {
				break
			}
			if diffi == nil {
				return false
			}
			if diffj == nil {
				return true
			}
			if *diffi != *diffj {
				return *diffi > *diffj
			}
		case "growth_rate":
			diffi := utils.PSub(apiMetrics[i].Count, apiMetrics[i].OldCount)
			diffj := utils.PSub(apiMetrics[j].Count, apiMetrics[j].OldCount)
			if diffi == nil && diffj == nil {
				break
			}
			if diffi == nil {
				return false
			}
			if diffj == nil {
				return true
			}
			if apiMetrics[i].OldCount == nil && apiMetrics[j].OldCount == nil {
				break
			}
			if apiMetrics[i].OldCount == nil {
				return true
			}
			if apiMetrics[j].OldCount == nil {
				return false
			}
			if *apiMetrics[i].OldCount == 0 && *apiMetrics[j].OldCount == 0 {
				break
			}
			if *apiMetrics[i].OldCount == 0 {
				return false
			}
			if *apiMetrics[j].OldCount == 0 {
				return true
			}
			if float64(*diffi)/float64(*apiMetrics[i].OldCount) != float64(*diffj)/float64(*apiMetrics[j].OldCount) {
				return float64(*diffi)/float64(*apiMetrics[i].OldCount) > float64(*diffj)/float64(*apiMetrics[j].OldCount)
			}
		}
		return apiMetrics[i].Name < apiMetrics[j].Name
	})

	result := inventoryApi.ListMetricsResponse{
		TotalCount:    totalCount,
		TotalOldCount: totalOldCount,
		TotalMetrics:  len(apiMetrics),
		Metrics:       utils.Paginate(pageNumber, pageSize, apiMetrics),
	}

	return ctx.JSON(http.StatusOK, result)
}

// ListAnalyticsTags godoc
//
//	@Summary		List analytics tags
//	@Description	Retrieving a list of tag keys with their possible values for all analytic metrics.
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			connector		query		[]string	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string	false	"Connection IDs to filter by - mutually exclusive with connectionGroup"
//	@Param			connectionGroup	query		[]string	false	"Connection group to filter by - mutually exclusive with connectionId"
//	@Param			minCount		query		int			false	"Minimum number of resources/spend with this tag value, default 1"
//	@Param			startTime		query		int64		false	"Start time in unix timestamp format, default now - 1 month"
//	@Param			endTime			query		int64		false	"End time in unix timestamp format, default now"
//	@Param			metricType		query		string		false	"Metric type, default: assets"	Enums(assets, spend)
//	@Success		200				{object}	map[string][]string
//	@Router			/inventory/api/v2/analytics/tag [get]
func (h *HttpHandler) ListAnalyticsTags(ctx echo.Context) error {
	connectorTypes := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if len(connectionIDs) > MaxConns {
		return ctx.JSON(http.StatusBadRequest, "too many connections")
	}
	connectorTypes, err = h.getConnectorTypesFromConnectionIDs(ctx, connectorTypes, connectionIDs)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	minCount := 1
	if minCountStr := ctx.QueryParam("minCount"); minCountStr != "" {
		minCountVal, err := strconv.ParseInt(minCountStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "minCount must be a number")
		}
		minCount = int(minCountVal)
	}
	minAmount := float64(minCount)
	endTime, err := utils.TimeFromQueryParam(ctx, "endTime", time.Now())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "endTime must be a number")
	}
	startTime, err := utils.TimeFromQueryParam(ctx, "startTime", endTime.AddDate(0, -1, 0))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "startTime must be a number")
	}

	metricType := analyticsDB.MetricType(ctx.QueryParam("metricType"))
	if metricType == "" {
		metricType = analyticsDB.MetricTypeAssets
	}

	aDB := analyticsDB.NewDatabase(h.db.orm)
	fmt.Println("connectorTypes", connectorTypes)
	// trace :
	outputS1, span1 := tracer.Start(ctx.Request().Context(), "new_ListMetricTagsKeysWithPossibleValues", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_ListMetricTagsKeysWithPossibleValues")

	tags, err := aDB.ListMetricTagsKeysWithPossibleValues(connectorTypes)
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.End()
	tags = model.TrimPrivateTags(tags)

	var metricCount map[string]int
	var spend map[string]es.SpendMetricResp

	if metricType == analyticsDB.MetricTypeAssets {
		if len(connectionIDs) > 0 {
			metricCount, err = es.FetchConnectionAnalyticMetricCountAtTime(h.client, connectorTypes, connectionIDs, endTime, nil, EsFetchPageSize)
		} else {
			metricCount, err = es.FetchConnectorAnalyticMetricCountAtTime(h.client, connectorTypes, endTime, nil, EsFetchPageSize)
		}
		if err != nil {
			return err
		}
	} else {
		spend, err = es.FetchSpendByMetric(h.client, connectionIDs, connectorTypes, nil, startTime, endTime, EsFetchPageSize)
		if err != nil {
			return err
		}
	}

	fmt.Println("metricCount", metricCount)
	fmt.Println("spend", spend)
	fmt.Println("tags", tags)

	filteredTags := map[string][]string{}
	// tracer:
	outputS2, span2 := tracer.Start(outputS1, "new_ListFilteredMetrics(loop)", trace.WithSpanKind(trace.SpanKindServer))
	span2.SetName("new_ListFilteredMetrics(loop)")

	for key, values := range tags {
		for _, tagValue := range values {
			_, span3 := tracer.Start(outputS2, "new_ListFilteredMetrics", trace.WithSpanKind(trace.SpanKindServer))
			span3.SetName("new_ListFilteredMetrics")

			metrics, err := aDB.ListFilteredMetrics(map[string][]string{
				key: {tagValue},
			}, metricType, nil, connectorTypes, false)
			if err != nil {
				span3.RecordError(err)
				span3.SetStatus(codes.Error, err.Error())
				return err
			}
			span3.End()

			fmt.Println("metrics", key, tagValue, metrics)
			for _, metric := range metrics {
				if (metric.Type == analyticsDB.MetricTypeAssets && metricCount[metric.ID] >= minCount) ||
					(metric.Type == analyticsDB.MetricTypeSpend && spend[metric.ID].CostValue >= minAmount) {
					filteredTags[key] = append(filteredTags[key], tagValue)
					break
				}
			}
		}
	}
	tags = filteredTags
	fmt.Println("filteredTags", filteredTags)

	return ctx.JSON(http.StatusOK, tags)
}

// ListAnalyticsMetricTrend godoc
//
//	@Summary		Get metric trend
//
//	@Description	Retrieving a list of resource counts over the course of the specified time frame based on the given input filters
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			tag				query		[]string		false	"Key-Value tags in key=value format to filter by"
//	@Param			metricType		query		string			false	"Metric type, default: assets"	Enums(assets, spend)
//	@Param			ids				query		[]string		false	"Metric IDs to filter by"
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by - mutually exclusive with connectionGroup"
//	@Param			connectionGroup	query		[]string		false	"Connection group to filter by - mutually exclusive with connectionId"
//	@Param			startTime		query		int64			false	"timestamp for start in epoch seconds"
//	@Param			endTime			query		int64			false	"timestamp for end in epoch seconds"
//	@Param			datapointCount	query		string			false	"maximum number of datapoints to return, default is 30"
//	@Success		200				{object}	[]inventoryApi.ResourceTypeTrendDatapoint
//	@Router			/inventory/api/v2/analytics/trend [get]
func (h *HttpHandler) ListAnalyticsMetricTrend(ctx echo.Context) error {
	aDB := analyticsDB.NewDatabase(h.db.orm)
	tagMap := model.TagStringsToTagMap(httpserver.QueryArrayParam(ctx, "tag"))
	metricType := analyticsDB.MetricType(ctx.QueryParam("metricType"))
	if metricType == "" {
		metricType = analyticsDB.MetricTypeAssets
	}
	ids := httpserver.QueryArrayParam(ctx, "ids")
	connectorTypes := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
	if len(connectionIDs) > MaxConns {
		return echo.NewHTTPError(http.StatusBadRequest, "too many connections")
	}

	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now()
	if endTimeStr != "" {
		endTimeVal, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
		endTime = time.Unix(endTimeVal, 0)
	}
	startTimeStr := ctx.QueryParam("startTime")
	startTime := endTime.AddDate(0, -1, 0)
	if startTimeStr != "" {
		startTimeVal, err := strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
		startTime = time.Unix(startTimeVal, 0)
	}

	datapointCountStr := ctx.QueryParam("datapointCount")
	datapointCount := int64(30)
	if datapointCountStr != "" {
		datapointCount, err = strconv.ParseInt(datapointCountStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid datapointCount")
		}
	}
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_ListFilteredMetrics", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_ListFilteredMetrics")

	metrics, err := aDB.ListFilteredMetrics(tagMap, metricType, ids, connectorTypes, false)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.End()

	metricIDs := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		metricIDs = append(metricIDs, metric.ID)
	}

	timeToCountMap := make(map[int]es.DatapointWithFailures)
	if endTime.Round(24 * time.Hour).Before(endTime) {
		endTime = endTime.Round(24 * time.Hour).Add(24 * time.Hour)
	} else {
		endTime = endTime.Round(24 * time.Hour)
	}
	if startTime.Round(24 * time.Hour).After(startTime) {
		startTime = startTime.Round(24 * time.Hour).Add(-24 * time.Hour)
	} else {
		startTime = startTime.Round(24 * time.Hour)
	}

	esDatapointCount := int(math.Floor(endTime.Sub(startTime).Hours() / 24))
	if esDatapointCount == 0 {
		esDatapointCount = 1
	}
	if len(connectionIDs) != 0 {
		timeToCountMap, err = es.FetchConnectionMetricTrendSummaryPage(h.client, connectionIDs, metricIDs, startTime, endTime, esDatapointCount, EsFetchPageSize)
		if err != nil {
			return err
		}
	} else {
		timeToCountMap, err = es.FetchConnectorMetricTrendSummaryPage(h.client, connectorTypes, metricIDs, startTime, endTime, esDatapointCount, EsFetchPageSize)
		if err != nil {
			return err
		}
	}

	apiDatapoints := make([]inventoryApi.ResourceTypeTrendDatapoint, 0, len(timeToCountMap))
	for timeAt, val := range timeToCountMap {
		apiDatapoints = append(apiDatapoints, inventoryApi.ResourceTypeTrendDatapoint{
			Count:                                   val.Count,
			TotalDescribedConnectionCount:           val.TotalConnections,
			TotalSuccessfulDescribedConnectionCount: val.TotalSuccessfulConnections,
			Date:                                    time.UnixMilli(int64(timeAt)),
		})
	}
	sort.Slice(apiDatapoints, func(i, j int) bool {
		return apiDatapoints[i].Date.Before(apiDatapoints[j].Date)
	})
	apiDatapoints = internal.DownSampleResourceTypeTrendDatapoints(apiDatapoints, int(datapointCount))

	return ctx.JSON(http.StatusOK, apiDatapoints)
}

// ListAnalyticsComposition godoc
//
//	@Summary		List analytics composition
//	@Description	Retrieving tag values with the most resources for the given key.
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			key				path		string			true	"Tag key"
//	@Param			metricType		query		string			false	"Metric type, default: assets"	Enums(assets, spend)
//	@Param			top				query		int				true	"How many top values to return default is 5"
//	@Param			connector		query		[]source.Type	false	"Connector types to filter by"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by - mutually exclusive with connectionGroup"
//	@Param			connectionGroup	query		[]string		false	"Connection group to filter by - mutually exclusive with connectionId"
//	@Param			endTime			query		int64			false	"timestamp for resource count in epoch seconds"
//	@Param			startTime		query		int64			false	"timestamp for resource count change comparison in epoch seconds"
//	@Success		200				{object}	inventoryApi.ListResourceTypeCompositionResponse
//	@Router			/inventory/api/v2/analytics/composition/{key} [get]
func (h *HttpHandler) ListAnalyticsComposition(ctx echo.Context) error {
	aDB := analyticsDB.NewDatabase(h.db.orm)

	var err error
	tagKey := ctx.Param("key")
	if tagKey == "" || strings.HasPrefix(tagKey, model.KaytuPrivateTagPrefix) {
		return echo.NewHTTPError(http.StatusBadRequest, "tag key is invalid")
	}
	metricType := analyticsDB.MetricType(ctx.QueryParam("metricType"))
	if metricType == "" {
		metricType = analyticsDB.MetricTypeAssets
	}

	topStr := ctx.QueryParam("top")
	top := int64(5)
	if topStr != "" {
		top, err = strconv.ParseInt(topStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid top value")
		}

	}
	connectorTypes := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
	if len(connectionIDs) > MaxConns {
		return ctx.JSON(http.StatusBadRequest, "too many connections")
	}

	endTime := time.Now()
	if endTimeStr := ctx.QueryParam("endTime"); endTimeStr != "" {
		endTimeVal, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "invalid endTime value")
		}
		endTime = time.Unix(endTimeVal, 0)
	}
	startTime := endTime.AddDate(0, 0, -7)
	if startTimeStr := ctx.QueryParam("startTime"); startTimeStr != "" {
		startTimeVal, err := strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "invalid startTime value")
		}
		startTime = time.Unix(startTimeVal, 0)
	}
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_ListFilteredMetrics", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_ListFilteredMetrics")

	filteredMetrics, err := aDB.ListFilteredMetrics(map[string][]string{tagKey: nil}, metricType, nil, connectorTypes, false)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.End()

	var metrics []analyticsDB.AnalyticMetric
	for _, metric := range filteredMetrics {
		metrics = append(metrics, metric)
	}
	metricsIDs := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		metricsIDs = append(metricsIDs, metric.ID)
	}

	var metricIndexed map[string]int
	if len(connectionIDs) > 0 {
		metricIndexed, err = es.FetchConnectionAnalyticMetricCountAtTime(h.client, connectorTypes, connectionIDs, endTime, metricsIDs, EsFetchPageSize)
	} else {
		metricIndexed, err = es.FetchConnectorAnalyticMetricCountAtTime(h.client, connectorTypes, endTime, metricsIDs, EsFetchPageSize)
	}
	if err != nil {
		return err
	}

	var oldMetricIndexed map[string]int
	if len(connectionIDs) > 0 {
		oldMetricIndexed, err = es.FetchConnectionAnalyticMetricCountAtTime(h.client, connectorTypes, connectionIDs, startTime, metricsIDs, EsFetchPageSize)
	} else {
		oldMetricIndexed, err = es.FetchConnectorAnalyticMetricCountAtTime(h.client, connectorTypes, startTime, metricsIDs, EsFetchPageSize)
	}
	if err != nil {
		return err
	}

	type currentAndOldCount struct {
		current int
		old     int
	}

	valueCountMap := make(map[string]currentAndOldCount)
	totalCount := 0
	totalOldCount := 0
	for _, metric := range metrics {
		for _, tagValue := range metric.GetTagsMap()[tagKey] {
			if _, ok := valueCountMap[tagValue]; !ok {
				valueCountMap[tagValue] = currentAndOldCount{}
			}
			v := valueCountMap[tagValue]
			v.current += metricIndexed[metric.ID]
			v.old += oldMetricIndexed[metric.ID]
			totalCount += metricIndexed[metric.ID]
			totalOldCount += oldMetricIndexed[metric.ID]
			valueCountMap[tagValue] = v
			break
		}
	}

	type strIntPair struct {
		str    string
		counts currentAndOldCount
	}
	valueCountPairs := make([]strIntPair, 0, len(valueCountMap))
	for value, count := range valueCountMap {
		valueCountPairs = append(valueCountPairs, strIntPair{str: value, counts: count})
	}
	sort.Slice(valueCountPairs, func(i, j int) bool {
		return valueCountPairs[i].counts.current > valueCountPairs[j].counts.current
	})

	apiResult := inventoryApi.ListResourceTypeCompositionResponse{
		TotalCount:      totalCount,
		TotalValueCount: len(valueCountMap),
		TopValues:       make(map[string]inventoryApi.CountPair),
		Others:          inventoryApi.CountPair{},
	}

	for i, pair := range valueCountPairs {
		if i < int(top) {
			apiResult.TopValues[pair.str] = inventoryApi.CountPair{
				Count:    pair.counts.current,
				OldCount: pair.counts.old,
			}
		} else {
			apiResult.Others.Count += pair.counts.current
			apiResult.Others.OldCount += pair.counts.old
		}
	}

	return ctx.JSON(http.StatusOK, apiResult)
}

// ListAnalyticsCategories godoc
//
//	@Summary		List Analytics categories
//	@Description	Retrieving list of categories for analytics
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			metricType	query		string	false	"Metric type, default: assets"	Enums(assets, spend)
//	@Success		200			{object}	inventoryApi.AnalyticsCategoriesResponse
//	@Router			/inventory/api/v2/analytics/categories [get]
func (h *HttpHandler) ListAnalyticsCategories(ctx echo.Context) error {
	aDB := analyticsDB.NewDatabase(h.db.orm)
	metricType := analyticsDB.MetricType(ctx.QueryParam("metricType"))
	if metricType == "" {
		metricType = analyticsDB.MetricTypeAssets
	}
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_ListMetrics", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_ListMetrics")

	metrics, err := aDB.ListMetrics(false)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.End()

	categoryResourceTypeMap := map[string][]string{}
	for _, metric := range metrics {
		if metric.Type != metricType {
			continue
		}

		for _, tag := range metric.Tags {
			if tag.Key == "category" {
				for _, category := range tag.GetValue() {
					categoryResourceTypeMap[category] = append(
						categoryResourceTypeMap[category],
						metric.Tables...,
					)
				}
			}
		}
	}

	return ctx.JSON(http.StatusOK, inventoryApi.AnalyticsCategoriesResponse{
		CategoryResourceType: categoryResourceTypeMap,
	})
}

// GetAssetsTable godoc
//
//	@Summary		Get Assets Table
//	@Description	Returns asset table with respect to the dimension and granularity
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			startTime	query		int64	false	"timestamp for start in epoch seconds"
//	@Param			endTime		query		int64	false	"timestamp for end in epoch seconds"
//	@Param			granularity	query		string	false	"Granularity of the table, default is daily"	Enums(monthly, daily, yearly)
//	@Param			dimension	query		string	false	"Dimension of the table, default is metric"		Enums(connection, metric)
//
//	@Success		200			{object}	[]inventoryApi.AssetTableRow
//	@Router			/inventory/api/v2/analytics/table [get]
func (h *HttpHandler) GetAssetsTable(ctx echo.Context) error {
	var err error
	endTime, err := utils.TimeFromQueryParam(ctx, "endTime", time.Now())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	startTime, err := utils.TimeFromQueryParam(ctx, "startTime", endTime.AddDate(0, -1, 0))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	granularity := inventoryApi.SpendTableGranularity(ctx.QueryParam("granularity"))
	if granularity == "" {
		granularity = inventoryApi.SpendTableGranularityDaily
	}
	if granularity != inventoryApi.SpendTableGranularityDaily &&
		granularity != inventoryApi.SpendTableGranularityMonthly &&
		granularity != inventoryApi.SpendTableGranularityYearly {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid granularity")
	}
	dimension := inventoryApi.SpendDimension(ctx.QueryParam("dimension"))
	if dimension == "" {
		dimension = inventoryApi.SpendDimensionMetric
	}
	if dimension != inventoryApi.SpendDimensionMetric &&
		dimension != inventoryApi.SpendDimensionConnection {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid dimension")
	}

	aDB := analyticsDB.NewDatabase(h.db.orm)
	ms, err := aDB.ListFilteredMetrics(nil, analyticsDB.MetricTypeAssets, nil, nil, false)
	if err != nil {
		return err
	}
	var metricIds []string
	for _, m := range ms {
		metricIds = append(metricIds, m.ID)
	}
	mt, err := es.FetchAssetTableByDimension(h.client, metricIds, granularity, dimension, startTime, endTime)
	if err != nil {
		return err
	}

	var table []inventoryApi.AssetTableRow
	for _, m := range mt {
		resourceCount := map[string]float64{}
		for dateKey, costItem := range m.Trend {
			resourceCount[dateKey] = costItem
		}
		table = append(table, inventoryApi.AssetTableRow{
			DimensionID:   m.DimensionID,
			DimensionName: m.DimensionName,
			ResourceCount: resourceCount,
		})
	}
	return ctx.JSON(http.StatusOK, table)
}

// ListAnalyticsSpendMetricsHandler godoc
//
//	@Summary		List spend metrics
//	@Description	Retrieving cost metrics with respect to specified filters. The API returns information such as the total cost and costs per each service based on the specified filters.
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			filter			query		string			false	"Filter costs"
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by - mutually exclusive with connectionGroup"
//	@Param			connectionGroup	query		[]string		false	"Connection group to filter by - mutually exclusive with connectionId"
//	@Param			startTime		query		int64			false	"timestamp for start in epoch seconds"
//	@Param			endTime			query		int64			false	"timestamp for end in epoch seconds"
//	@Param			sortBy			query		string			false	"Sort by field - default is cost"	Enums(dimension,cost,growth,growth_rate)
//	@Param			pageSize		query		int				false	"page size - default is 20"
//	@Param			pageNumber		query		int				false	"page number - default is 1"
//	@Param			metricIDs		query		[]string		false	"Metric IDs"
//	@Success		200				{object}	inventoryApi.ListCostMetricsResponse
//	@Router			/inventory/api/v2/analytics/spend/metric [get]
func (h *HttpHandler) ListAnalyticsSpendMetricsHandler(ctx echo.Context) error {
	var err error
	connectorTypes := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
	endTime, err := utils.TimeFromQueryParam(ctx, "endTime", time.Now())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	startTime, err := utils.TimeFromQueryParam(ctx, "startTime", endTime.AddDate(0, -1, 0))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	pageNumber, pageSize, err := utils.PageConfigFromStrings(ctx.QueryParam("pageNumber"), ctx.QueryParam("pageSize"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	sortBy := strings.ToLower(ctx.QueryParam("sortBy"))
	if sortBy == "" {
		sortBy = "cost"
	}
	if sortBy != "dimension" && sortBy != "cost" &&
		sortBy != "growth" && sortBy != "growth_rate" {
		return ctx.JSON(http.StatusBadRequest, "invalid sortBy value")
	}

	aDB := analyticsDB.NewDatabase(h.db.orm)
	metricIds := httpserver.QueryArrayParam(ctx, "metricIDs")
	metrics, err := aDB.ListFilteredMetrics(nil, analyticsDB.MetricTypeSpend, metricIds, connectorTypes, false)
	if err != nil {
		return err
	}
	metricIds = []string{}
	for _, m := range metrics {
		metricIds = append(metricIds, m.ID)
	}

	filterStr := ctx.QueryParam("filter")
	if filterStr != "" {
		var filter map[string]interface{}
		err = json.Unmarshal([]byte(filterStr), &filter)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "could not parse filter")
		}
		connectionIDs, err = h.connectionsFilter(filter)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, fmt.Sprintf("invalid filter: %s", err.Error()))
		}
		h.logger.Warn(fmt.Sprintf("===Filtered Connections: %v", connectionIDs))
	}

	costMetricMap := make(map[string]inventoryApi.CostMetric)
	if filterStr != "" && len(connectionIDs) == 0 {
		return ctx.JSON(http.StatusOK, inventoryApi.ListCostMetricsResponse{
			TotalCount: 0,
			TotalCost:  0,
			Metrics:    []inventoryApi.CostMetric{},
		})
	} else if len(connectionIDs) > 0 {
		hits, err := es.FetchConnectionDailySpendHistoryByMetric(h.client, connectionIDs, connectorTypes, metricIds, startTime, endTime, EsFetchPageSize)
		if err != nil {
			return err
		}
		for _, hit := range hits {
			localHit := hit
			if v, ok := costMetricMap[localHit.MetricID]; ok {
				exists := false
				for _, cnt := range v.Connector {
					if cnt.String() == localHit.Connector.String() {
						exists = true
						break
					}
				}
				if !exists {
					v.Connector = append(v.Connector, localHit.Connector)
				}
				v.TotalCost = utils.PAdd(v.TotalCost, &localHit.TotalCost)
				v.DailyCostAtStartTime = utils.PAdd(v.DailyCostAtStartTime, &localHit.StartDateCost)
				v.DailyCostAtEndTime = utils.PAdd(v.DailyCostAtEndTime, &localHit.EndDateCost)
				costMetricMap[localHit.MetricID] = v
			} else {
				costMetricMap[localHit.MetricID] = inventoryApi.CostMetric{
					Connector:            []source.Type{localHit.Connector},
					CostDimensionName:    localHit.MetricName,
					CostDimensionID:      localHit.MetricID,
					TotalCost:            &localHit.TotalCost,
					DailyCostAtStartTime: &localHit.StartDateCost,
					DailyCostAtEndTime:   &localHit.EndDateCost,
				}
			}
		}
	} else {
		hits, err := es.FetchConnectorDailySpendHistoryByMetric(h.client, connectorTypes, metricIds, startTime, endTime, EsFetchPageSize)
		if err != nil {
			return err
		}
		for _, hit := range hits {
			localHit := hit
			connector, _ := source.ParseType(localHit.Connector)
			if v, ok := costMetricMap[localHit.MetricID]; ok {
				exists := false
				for _, cnt := range v.Connector {
					if cnt.String() == connector.String() {
						exists = true
						break
					}
				}
				if !exists {
					v.Connector = append(v.Connector, connector)
				}
				v.TotalCost = utils.PAdd(v.TotalCost, &localHit.TotalCost)
				v.DailyCostAtStartTime = utils.PAdd(v.DailyCostAtStartTime, &localHit.StartDateCost)
				v.DailyCostAtEndTime = utils.PAdd(v.DailyCostAtEndTime, &localHit.EndDateCost)
				costMetricMap[localHit.MetricID] = v
			} else {
				costMetricMap[localHit.MetricID] = inventoryApi.CostMetric{
					Connector:            []source.Type{connector},
					CostDimensionName:    localHit.MetricName,
					CostDimensionID:      localHit.MetricID,
					TotalCost:            &localHit.TotalCost,
					DailyCostAtStartTime: &localHit.StartDateCost,
					DailyCostAtEndTime:   &localHit.EndDateCost,
				}
			}
		}
	}

	var costMetrics []inventoryApi.CostMetric
	totalCost := float64(0)
	for _, costMetric := range costMetricMap {
		costMetrics = append(costMetrics, costMetric)
		if costMetric.TotalCost != nil {
			totalCost += *costMetric.TotalCost
		}
	}

	sort.Slice(costMetrics, func(i, j int) bool {
		switch sortBy {
		case "dimension":
			return costMetrics[i].CostDimensionName < costMetrics[j].CostDimensionName
		case "cost":
			if costMetrics[i].TotalCost == nil && costMetrics[j].TotalCost == nil {
				break
			}
			if costMetrics[i].TotalCost == nil {
				return false
			}
			if costMetrics[j].TotalCost == nil {
				return true
			}
			if *costMetrics[i].TotalCost != *costMetrics[j].TotalCost {
				return *costMetrics[i].TotalCost > *costMetrics[j].TotalCost
			}
		case "growth":
			diffi := utils.PSub(costMetrics[i].DailyCostAtEndTime, costMetrics[i].DailyCostAtStartTime)
			diffj := utils.PSub(costMetrics[j].DailyCostAtEndTime, costMetrics[j].DailyCostAtStartTime)
			if diffi == nil && diffj == nil {
				break
			}
			if diffi == nil {
				return false
			}
			if diffj == nil {
				return true
			}
			if *diffi != *diffj {
				return *diffi > *diffj
			}
		case "growth_rate":
			diffi := utils.PSub(costMetrics[i].DailyCostAtEndTime, costMetrics[i].DailyCostAtStartTime)
			diffj := utils.PSub(costMetrics[j].DailyCostAtEndTime, costMetrics[j].DailyCostAtStartTime)
			if diffi == nil && diffj == nil {
				break
			}
			if diffi == nil {
				return false
			}
			if diffj == nil {
				return true
			}
			if costMetrics[i].DailyCostAtStartTime == nil && costMetrics[j].DailyCostAtStartTime == nil {
				break
			}
			if costMetrics[i].DailyCostAtStartTime == nil {
				return true
			}
			if costMetrics[j].DailyCostAtStartTime == nil {
				return false
			}
			if *costMetrics[i].DailyCostAtStartTime == 0 && *costMetrics[j].DailyCostAtStartTime == 0 {
				break
			}
			if *costMetrics[i].DailyCostAtStartTime == 0 {
				return false
			}
			if *costMetrics[j].DailyCostAtStartTime == 0 {
				return true
			}
			if *diffi/(*costMetrics[i].DailyCostAtStartTime) != *diffj/(*costMetrics[j].DailyCostAtStartTime) {
				return *diffi/(*costMetrics[i].DailyCostAtStartTime) > *diffj/(*costMetrics[j].DailyCostAtStartTime)
			}
		}
		return costMetrics[i].CostDimensionName < costMetrics[j].CostDimensionName
	})

	return ctx.JSON(http.StatusOK, inventoryApi.ListCostMetricsResponse{
		TotalCount: len(costMetrics),
		TotalCost:  totalCost,
		Metrics:    utils.Paginate(pageNumber, pageSize, costMetrics),
	})
}

// ListMetrics godoc
//
//	@Summary		List metrics
//	@Description	Returns list of metrics
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			connector	query		[]source.Type	false	"Connector type to filter by"
//	@Param			metricType	query		string			false	"Metric type, default: assets"	Enums(assets, spend)
//
//	@Success		200			{object}	[]inventoryApi.AnalyticsMetric
//	@Router			/inventory/api/v2/analytics/metrics/list [get]
func (h *HttpHandler) ListMetrics(ctx echo.Context) error {
	aDB := analyticsDB.NewDatabase(h.db.orm)
	var err error
	connectorTypes := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	metricType := analyticsDB.MetricType(ctx.QueryParam("metricType"))
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_ListFilteredMetrics", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_ListFilteredMetrics")

	metrics, err := aDB.ListFilteredMetrics(nil, metricType, nil, connectorTypes, false)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.End()

	var apiMetrics []inventoryApi.AnalyticsMetric
	for _, metric := range metrics {
		apiMetric := inventoryApi.AnalyticsMetric{
			ID:          metric.ID,
			Connectors:  source.ParseTypes(metric.Connectors),
			Type:        metric.Type,
			Name:        metric.Name,
			Query:       metric.Query,
			Tables:      metric.Tables,
			FinderQuery: metric.FinderQuery,
			Tags:        metric.GetTagsMap(),
		}

		apiMetrics = append(apiMetrics, apiMetric)
	}
	return ctx.JSON(http.StatusOK, apiMetrics)
}

// GetMetric godoc
//
//	@Summary		List metrics
//	@Description	Returns list of metrics
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			metric_id	path		string	true	"MetricID"
//
//	@Success		200			{object}	inventoryApi.AnalyticsMetric
//	@Router			/inventory/api/v2/analytics/metrics/{metric_id} [get]
func (h *HttpHandler) GetMetric(ctx echo.Context) error {
	aDB := analyticsDB.NewDatabase(h.db.orm)
	var err error

	metricID := ctx.Param("metric_id")
	_, span := tracer.Start(ctx.Request().Context(), "new_GetMetric", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetMetric")

	metric, err := aDB.GetMetricByID(metricID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if metric == nil {
		return echo.NewHTTPError(http.StatusNotFound, "metric not found")
	}

	span.End()

	apiMetric := inventoryApi.AnalyticsMetric{
		ID:          metric.ID,
		Connectors:  source.ParseTypes(metric.Connectors),
		Type:        metric.Type,
		Name:        metric.Name,
		Query:       metric.Query,
		Tables:      metric.Tables,
		FinderQuery: metric.FinderQuery,
		Tags:        metric.GetTagsMap(),
	}
	return ctx.JSON(http.StatusOK, apiMetric)
}

// ListAnalyticsSpendComposition godoc
//
//	@Summary		List cost composition
//	@Description	Retrieving the cost composition with respect to specified filters. Retrieving information such as the total cost for the given time range, and the top services by cost.
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by - mutually exclusive with connectionGroup"
//	@Param			connectionGroup	query		[]string		false	"Connection group to filter by - mutually exclusive with connectionId"
//	@Param			top				query		int				false	"How many top values to return default is 5"
//	@Param			startTime		query		int64			false	"timestamp for start in epoch seconds"
//	@Param			endTime			query		int64			false	"timestamp for end in epoch seconds"
//	@Success		200				{object}	inventoryApi.ListCostCompositionResponse
//	@Router			/inventory/api/v2/analytics/spend/composition [get]
func (h *HttpHandler) ListAnalyticsSpendComposition(ctx echo.Context) error {
	aDB := analyticsDB.NewDatabase(h.db.orm)
	var err error
	connectorTypes := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
	endTime, err := utils.TimeFromQueryParam(ctx, "endTime", time.Now())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	startTime, err := utils.TimeFromQueryParam(ctx, "startTime", endTime.AddDate(0, -1, 0))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	topStr := ctx.QueryParam("top")
	top := int64(5)
	if topStr != "" {
		top, err = strconv.ParseInt(topStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid top value")
		}
	}
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_ListFilteredMetrics", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_ListFilteredMetrics")

	metrics, err := aDB.ListFilteredMetrics(nil, analyticsDB.MetricTypeSpend, nil, nil, false)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.End()

	costMetricMap := make(map[string]inventoryApi.CostMetric)
	spends, err := es.FetchSpendByMetric(h.client, connectionIDs, connectorTypes, nil, startTime, endTime, EsFetchPageSize)
	if err != nil {
		return err
	}
	for metricID, spend := range spends {
		localSpend := spend

		var metric analyticsDB.AnalyticMetric
		for _, m := range metrics {
			if m.ID == metricID {
				metric = m
			}
		}

		categoryExists := false
		for _, tag := range metric.Tags {
			if tag.GetKey() == "category" {
				for _, value := range tag.GetValue() {
					categoryExists = true
					if v, ok := costMetricMap[value]; ok {
						v.TotalCost = utils.PAdd(v.TotalCost, &localSpend.CostValue)
						costMetricMap[value] = v
					} else {
						costMetricMap[value] = inventoryApi.CostMetric{
							CostDimensionName: value,
							TotalCost:         &localSpend.CostValue,
						}
					}
				}
			}
		}

		if !categoryExists {
			costMetricMap[metricID] = inventoryApi.CostMetric{
				CostDimensionName: localSpend.MetricName,
				TotalCost:         &localSpend.CostValue,
			}
		}
	}

	var costMetrics []inventoryApi.CostMetric
	totalCost := float64(0)
	for _, costMetric := range costMetricMap {
		costMetrics = append(costMetrics, costMetric)
		if costMetric.TotalCost != nil {
			totalCost += *costMetric.TotalCost
		}
	}

	sort.Slice(costMetrics, func(i, j int) bool {
		if costMetrics[i].TotalCost == nil {
			return false
		}
		if costMetrics[j].TotalCost == nil {
			return true
		}
		if *costMetrics[i].TotalCost != *costMetrics[j].TotalCost {
			return *costMetrics[i].TotalCost > *costMetrics[j].TotalCost
		}
		return costMetrics[i].CostDimensionName < costMetrics[j].CostDimensionName
	})

	topCostMap := make(map[string]float64)
	othersCost := float64(0)
	if top > int64(len(costMetrics)) {
		top = int64(len(costMetrics))
	}
	for _, costMetric := range costMetrics[:int(top)] {
		if costMetric.TotalCost != nil {
			topCostMap[costMetric.CostDimensionName] = *costMetric.TotalCost
		}
	}
	if len(costMetrics) > int(top) {
		for _, costMetric := range costMetrics[int(top):] {
			if costMetric.TotalCost != nil {
				othersCost += *costMetric.TotalCost
			}
		}
	}

	return ctx.JSON(http.StatusOK, inventoryApi.ListCostCompositionResponse{
		TotalCount:     len(costMetrics),
		TotalCostValue: totalCost,
		TopValues:      topCostMap,
		Others:         othersCost,
	})
}

// GetAnalyticsSpendTrend godoc
//
//	@Summary		Get Cost Trend
//	@Description	Retrieving a list of costs over the course of the specified time frame based on the given input filters. If startTime and endTime are empty, the API returns the last month trend.
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by - mutually exclusive with connectionGroup"
//	@Param			connectionGroup	query		[]string		false	"Connection group to filter by - mutually exclusive with connectionId"
//	@Param			metricIds		query		[]string		false	"Metrics IDs"
//	@Param			startTime		query		int64			false	"timestamp for start in epoch seconds"
//	@Param			endTime			query		int64			false	"timestamp for end in epoch seconds"
//	@Param			granularity		query		string			false	"Granularity of the table, default is daily"	Enums(monthly, daily, yearly)
//	@Success		200				{object}	[]inventoryApi.CostTrendDatapoint
//	@Router			/inventory/api/v2/analytics/spend/trend [get]
func (h *HttpHandler) GetAnalyticsSpendTrend(ctx echo.Context) error {
	var err error
	metricIds := httpserver.QueryArrayParam(ctx, "metricIds")
	connectorTypes := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))

	aDB := analyticsDB.NewDatabase(h.db.orm)
	metrics, err := aDB.ListFilteredMetrics(nil, analyticsDB.MetricTypeSpend, metricIds, connectorTypes, false)
	if err != nil {
		return err
	}
	metricIds = nil
	for _, m := range metrics {
		metricIds = append(metricIds, m.ID)
	}

	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}

	endTime, err := utils.TimeFromQueryParam(ctx, "endTime", time.Now())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	startTime, err := utils.TimeFromQueryParam(ctx, "startTime", endTime.AddDate(0, -1, 0))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	granularity := inventoryApi.SpendTableGranularity(ctx.QueryParam("granularity"))
	if granularity == "" {
		granularity = inventoryApi.SpendTableGranularityDaily
	}
	if granularity != inventoryApi.SpendTableGranularityDaily &&
		granularity != inventoryApi.SpendTableGranularityMonthly &&
		granularity != inventoryApi.SpendTableGranularityYearly {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid granularity")
	}

	timepointToCost := map[string]es.DatapointWithFailures{}
	if len(connectionIDs) > 0 {
		timepointToCost, err = es.FetchConnectionSpendTrend(h.client, granularity, metricIds, connectionIDs, connectorTypes, startTime, endTime)
	} else {
		timepointToCost, err = es.FetchConnectorSpendTrend(h.client, granularity, metricIds, connectorTypes, startTime, endTime)
	}
	if err != nil {
		return err
	}

	apiDatapoints := make([]inventoryApi.CostTrendDatapoint, 0, len(timepointToCost))
	for timeAt, costVal := range timepointToCost {
		format := "2006-01-02"
		if granularity == inventoryApi.SpendTableGranularityMonthly {
			format = "2006-01"
		} else if granularity == inventoryApi.SpendTableGranularityYearly {
			format = "2006"
		}
		dt, _ := time.Parse(format, timeAt)
		apiDatapoints = append(apiDatapoints, inventoryApi.CostTrendDatapoint{
			Cost:                                    costVal.Cost,
			TotalDescribedConnectionCount:           costVal.TotalConnections,
			TotalSuccessfulDescribedConnectionCount: costVal.TotalSuccessfulConnections,
			Date:                                    dt,
		})
	}
	sort.Slice(apiDatapoints, func(i, j int) bool {
		return apiDatapoints[i].Date.Before(apiDatapoints[j].Date)
	})

	return ctx.JSON(http.StatusOK, apiDatapoints)
}

// GetAnalyticsSpendMetricsTrend godoc
//
//	@Summary		Get Cost Trend
//	@Description	Retrieving a list of costs over the course of the specified time frame based on the given input filters. If startTime and endTime are empty, the API returns the last month trend.
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by - mutually exclusive with connectionGroup"
//	@Param			connectionGroup	query		[]string		false	"Connection group to filter by - mutually exclusive with connectionId"
//	@Param			metricIds		query		[]string		false	"Metrics IDs"
//	@Param			startTime		query		int64			false	"timestamp for start in epoch seconds"
//	@Param			endTime			query		int64			false	"timestamp for end in epoch seconds"
//	@Param			granularity		query		string			false	"Granularity of the table, default is daily"	Enums(monthly, daily, yearly)
//	@Success		200				{object}	[]inventoryApi.ListServicesCostTrendDatapoint
//	@Router			/inventory/api/v2/analytics/spend/metrics/trend [get]
func (h *HttpHandler) GetAnalyticsSpendMetricsTrend(ctx echo.Context) error {
	var err error
	metricIds := httpserver.QueryArrayParam(ctx, "metricIds")
	connectorTypes := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))

	aDB := analyticsDB.NewDatabase(h.db.orm)
	metrics, err := aDB.ListFilteredMetrics(nil, analyticsDB.MetricTypeSpend, metricIds, connectorTypes, false)
	if err != nil {
		return err
	}
	metricIds = nil
	for _, m := range metrics {
		metricIds = append(metricIds, m.ID)
	}

	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}

	endTime, err := utils.TimeFromQueryParam(ctx, "endTime", time.Now())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	startTime, err := utils.TimeFromQueryParam(ctx, "startTime", endTime.AddDate(0, -1, 0))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	granularity := inventoryApi.SpendTableGranularity(ctx.QueryParam("granularity"))
	if granularity == "" {
		granularity = inventoryApi.SpendTableGranularityDaily
	}
	if granularity != inventoryApi.SpendTableGranularityDaily &&
		granularity != inventoryApi.SpendTableGranularityMonthly &&
		granularity != inventoryApi.SpendTableGranularityYearly {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid granularity")
	}

	var mt []es.MetricTrend
	if len(connectionIDs) > 0 {
		mt, err = es.FetchConnectionSpendMetricTrend(h.client, granularity, metricIds, connectionIDs, connectorTypes, startTime, endTime)
	} else {
		mt, err = es.FetchConnectorSpendMetricTrend(h.client, granularity, metricIds, connectorTypes, startTime, endTime)
	}
	if err != nil {
		return err
	}

	var response []inventoryApi.ListServicesCostTrendDatapoint
	for _, m := range mt {
		apiDatapoints := make([]inventoryApi.CostTrendDatapoint, 0, len(m.Trend))
		for timeAt, costVal := range m.Trend {
			format := "2006-01-02"
			if granularity == inventoryApi.SpendTableGranularityMonthly {
				format = "2006-01"
			} else if granularity == inventoryApi.SpendTableGranularityYearly {
				format = "2006"
			}
			dt, _ := time.Parse(format, timeAt)
			apiDatapoints = append(apiDatapoints, inventoryApi.CostTrendDatapoint{Cost: costVal, Date: dt})
		}
		sort.Slice(apiDatapoints, func(i, j int) bool {
			return apiDatapoints[i].Date.Before(apiDatapoints[j].Date)
		})

		response = append(response, inventoryApi.ListServicesCostTrendDatapoint{
			ServiceName: m.MetricID,
			CostTrend:   apiDatapoints,
		})
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetSpendTable godoc
//
//	@Summary		Get Spend Trend
//	@Description	Returns spend table with respect to the dimension and granularity
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			startTime		query		int64		false	"timestamp for start in epoch seconds"
//	@Param			endTime			query		int64		false	"timestamp for end in epoch seconds"
//	@Param			granularity		query		string		false	"Granularity of the table, default is daily"	Enums(monthly, daily, yearly)
//	@Param			dimension		query		string		false	"Dimension of the table, default is metric"		Enums(connection, metric)
//	@Param			connectionId	query		[]string	false	"Connection IDs to filter by - mutually exclusive with connectionGroup"
//	@Param			connectionGroup	query		[]string	false	"Connection group to filter by - mutually exclusive with connectionId"
//	@Param			connector		query		string		false	"Connector"
//	@Param			metricIds		query		[]string	false	"Metrics IDs"
//
//	@Success		200				{object}	[]inventoryApi.SpendTableRow
//	@Router			/inventory/api/v2/analytics/spend/table [get]
func (h *HttpHandler) GetSpendTable(ctx echo.Context) error {
	aDB := analyticsDB.NewDatabase(h.db.orm)
	var err error
	metricIds := httpserver.QueryArrayParam(ctx, "metricIds")
	ms, err := aDB.ListFilteredMetrics(nil, analyticsDB.MetricTypeSpend, metricIds, nil, false)
	if err != nil {
		return err
	}
	metricIds = nil
	for _, m := range ms {
		metricIds = append(metricIds, m.ID)
	}

	connector, _ := source.ParseType(ctx.QueryParam("connector"))
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
	endTime, err := utils.TimeFromQueryParam(ctx, "endTime", time.Now())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	startTime, err := utils.TimeFromQueryParam(ctx, "startTime", endTime.AddDate(0, -1, 0))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	granularity := inventoryApi.SpendTableGranularity(ctx.QueryParam("granularity"))
	if granularity == "" {
		granularity = inventoryApi.SpendTableGranularityDaily
	}
	if granularity != inventoryApi.SpendTableGranularityDaily &&
		granularity != inventoryApi.SpendTableGranularityMonthly &&
		granularity != inventoryApi.SpendTableGranularityYearly {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid granularity")
	}
	dimension := inventoryApi.SpendDimension(ctx.QueryParam("dimension"))
	if dimension == "" {
		dimension = inventoryApi.SpendDimensionMetric
	}
	if dimension != inventoryApi.SpendDimensionMetric &&
		dimension != inventoryApi.SpendDimensionConnection {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid dimension")
	}

	connectionAccountIDMap := map[string]string{}
	var metrics []analyticsDB.AnalyticMetric

	if dimension == inventoryApi.SpendDimensionMetric {
		// trace :
		_, span := tracer.Start(ctx.Request().Context(), "new_ListFilteredMetrics", trace.WithSpanKind(trace.SpanKindServer))
		span.SetName("new_ListFilteredMetrics")

		metrics, err = aDB.ListFilteredMetrics(nil, analyticsDB.MetricTypeSpend, metricIds, nil, false)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		span.End()
	}

	mt, err := es.FetchSpendTableByDimension(h.client, dimension, connectionIDs, connector, metricIds, startTime, endTime)
	if err != nil {
		return err
	}

	fmt.Println("FetchSpendTableByDimension res = ", len(mt))
	var table []inventoryApi.SpendTableRow
	for _, m := range mt {
		costValue := map[string]float64{}
		for dateKey, costItem := range m.Trend {
			dt, _ := time.Parse("2006-01-02", dateKey)
			monthKey := dt.Format("2006-01")
			yearKey := dt.Format("2006")
			if granularity == "daily" {
				costValue[dateKey] = costItem
			} else if granularity == "monthly" {
				if v, ok := costValue[monthKey]; ok {
					costValue[monthKey] = v + costItem
				} else {
					costValue[monthKey] = costItem
				}
			} else if granularity == "yearly" {
				if v, ok := costValue[yearKey]; ok {
					costValue[yearKey] = v + costItem
				} else {
					costValue[yearKey] = costItem
				}
			}
		}

		var category, accountID string
		dimensionName := m.DimensionName
		if dimension == inventoryApi.SpendDimensionMetric {
			for _, metric := range metrics {
				if m.DimensionID == metric.ID {
					for _, tag := range metric.Tags {
						if tag.GetKey() == "category" {
							for _, v := range tag.GetValue() {
								category = v
								break
							}
							break
						}
					}
					break
				}
			}
		} else if dimension == inventoryApi.SpendDimensionConnection {
			if v, ok := connectionAccountIDMap[m.DimensionID]; ok {
				accountID = demo.EncodeResponseData(ctx, v)
			} else {
				src, err := h.onboardClient.GetSource(&httpclient.Context{UserRole: authApi.InternalRole}, m.DimensionID)
				if err != nil {
					return err
				}
				accountID = demo.EncodeResponseData(ctx, src.ConnectionID)
				connectionAccountIDMap[m.DimensionID] = accountID
			}
			dimensionName = demo.EncodeResponseData(ctx, dimensionName)
		}

		table = append(table, inventoryApi.SpendTableRow{
			DimensionID:   m.DimensionID,
			AccountID:     accountID,
			Connector:     m.Connector,
			Category:      category,
			DimensionName: dimensionName,
			CostValue:     costValue,
		})
	}
	return ctx.JSON(http.StatusOK, table)
}

// GetResourceTypeMetricsHandler godoc
//
//	@Summary		List resource-type metrics
//	@Description	Retrieving metrics for a specific resource type.
//	@Security		BearerToken
//	@Tags			resource
//	@Accept			json
//	@Produce		json
//	@Param			connectionId	query		[]string	false	"Connection IDs to filter by - mutually exclusive with connectionGroup"
//	@Param			connectionGroup	query		[]string	false	"Connection group to filter by - mutually exclusive with connectionId"
//	@Param			endTime			query		int64		false	"timestamp for resource count in epoch seconds"
//	@Param			startTime		query		int64		false	"timestamp for resource count change comparison in epoch seconds"
//	@Param			resourceType	path		string		true	"ResourceType"
//	@Success		200				{object}	inventoryApi.ResourceType
//	@Router			/inventory/api/v2/resources/metric/{resourceType} [get]
func (h *HttpHandler) GetResourceTypeMetricsHandler(ctx echo.Context) error {
	var err error
	resourceType := ctx.Param("resourceType")
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	if len(connectionIDs) > MaxConns {
		return ctx.JSON(http.StatusBadRequest, "too many connections")
	}
	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now().Unix()
	if endTimeStr != "" {
		endTime, err = strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "invalid endTime value")
		}
	}
	startTimeStr := ctx.QueryParam("startTime")
	startTime := time.Unix(endTime, 0).AddDate(0, 0, -7).Unix()
	if startTimeStr != "" {
		startTime, err = strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "invalid startTime value")
		}
	}
	// tracer :
	outputS1, span := tracer.Start(ctx.Request().Context(), "new_GetBenchmarkTreeIDs", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetBenchmarkTreeIDs")

	apiResourceType, err := h.GetResourceTypeMetric(outputS1, resourceType, connectionIDs, endTime)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	oldApiResourceType, err := h.GetResourceTypeMetric(outputS1, resourceType, connectionIDs, startTime)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.End()
	apiResourceType.OldCount = oldApiResourceType.Count

	return ctx.JSON(http.StatusOK, *apiResourceType)
}

func (h *HttpHandler) GetResourceTypeMetric(ctx context.Context, resourceTypeStr string, connectionIDs []string, timeAt int64) (*inventoryApi.ResourceType, error) {
	// tracer :
	_, span := tracer.Start(ctx, "new_GetResourceType", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetResourceType")

	resourceType, err := h.db.GetResourceType(resourceTypeStr)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if err == gorm.ErrRecordNotFound {
			return nil, echo.NewHTTPError(http.StatusNotFound, "resource type not found")
		}
		return nil, err
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("service_name", resourceType.ServiceName),
	))
	span.End()

	var metricIndexed map[string]int
	if len(connectionIDs) > 0 {
		metricIndexed, err = es.FetchConnectionResourceTypeCountAtTime(h.client, nil, connectionIDs, time.Unix(timeAt, 0), []string{resourceTypeStr}, EsFetchPageSize)
	} else {
		metricIndexed, err = es.FetchConnectorResourceTypeCountAtTime(h.client, nil, time.Unix(timeAt, 0), []string{resourceTypeStr}, EsFetchPageSize)
	}
	if err != nil {
		return nil, err
	}

	apiResourceType := resourceType.ToApi()
	if count, ok := metricIndexed[strings.ToLower(resourceType.ResourceType)]; ok {
		apiResourceType.Count = &count
	}

	return &apiResourceType, nil
}

func (h *HttpHandler) ListConnectionsData(ctx echo.Context) error {
	aDB := analyticsDB.NewDatabase(h.db.orm)
	performanceStartTime := time.Now()
	var err error
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	connectors, err := h.getConnectorTypesFromConnectionIDs(ctx, nil, connectionIDs)
	if err != nil {
		return err
	}
	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now()
	if endTimeStr != "" {
		endTimeUnix, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "endTime is not a valid integer")
		}
		endTime = time.Unix(endTimeUnix, 0)
	}
	startTimeStr := ctx.QueryParam("startTime")
	startTime := endTime.AddDate(0, 0, -7)
	if startTimeStr != "" {
		startTimeUnix, err := strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "startTime is not a valid integer")
		}
		startTime = time.Unix(startTimeUnix, 0)
	}
	needCostStr := ctx.QueryParam("needCost")
	needCost := true
	if needCostStr == "false" {
		needCost = false
	}
	needResourceCountStr := ctx.QueryParam("needResourceCount")
	needResourceCount := true
	if needResourceCountStr == "false" {
		needResourceCount = false
	}

	fmt.Println("ListConnectionsData part1 ", time.Now().Sub(performanceStartTime).Milliseconds())
	res := map[string]inventoryApi.ConnectionData{}
	if needResourceCount {
		metrics, err := aDB.ListFilteredMetrics(nil, analyticsDB.MetricTypeAssets, nil, connectors, false)
		if err != nil {
			return err
		}
		var metricIDs []string
		for _, m := range metrics {
			metricIDs = append(metricIDs, m.ID)
		}

		resourceCountsMap, err := es.FetchConnectionAnalyticsResourcesCountAtTime(h.client, connectors, connectionIDs, metricIDs, endTime, EsFetchPageSize)
		if err != nil {
			return err
		}
		for connectionId, resourceCountAndEvaluated := range resourceCountsMap {
			if _, ok := res[connectionId]; !ok {
				res[connectionId] = inventoryApi.ConnectionData{
					ConnectionID: connectionId,
				}
			}
			v := res[connectionId]
			localCount := resourceCountAndEvaluated
			v.Count = utils.PAdd(v.Count, &localCount.ResourceCountsSum)
			if v.LastInventory == nil || v.LastInventory.IsZero() || v.LastInventory.Before(time.UnixMilli(localCount.LatestEvaluatedAt)) {
				v.LastInventory = utils.GetPointer(time.UnixMilli(localCount.LatestEvaluatedAt))
			}
			res[connectionId] = v
		}
		fmt.Println("ListConnectionsData part2 ", time.Now().Sub(performanceStartTime).Milliseconds())
		oldResourceCount, err := es.FetchConnectionAnalyticsResourcesCountAtTime(h.client, connectors, connectionIDs, metricIDs, startTime, EsFetchPageSize)
		if err != nil {
			return err
		}
		for connectionId, resourceCountAndEvaluated := range oldResourceCount {
			if _, ok := res[connectionId]; !ok {
				res[connectionId] = inventoryApi.ConnectionData{
					ConnectionID:  connectionId,
					LastInventory: nil,
				}
			}
			v := res[connectionId]
			localCount := resourceCountAndEvaluated
			v.OldCount = utils.PAdd(v.OldCount, &localCount.ResourceCountsSum)
			if v.LastInventory == nil || v.LastInventory.IsZero() || v.LastInventory.Before(time.UnixMilli(localCount.LatestEvaluatedAt)) {
				v.LastInventory = utils.GetPointer(time.UnixMilli(localCount.LatestEvaluatedAt))
			}
			res[connectionId] = v
		}
		fmt.Println("ListConnectionsData part3 ", time.Now().Sub(performanceStartTime).Milliseconds())
	}

	if needCost {
		hits, err := es.FetchConnectionDailySpendHistory(h.client, connectionIDs, connectors, nil, startTime, endTime, EsFetchPageSize)
		if err != nil {
			return err
		}
		for _, hit := range hits {
			localHit := hit
			if v, ok := res[localHit.ConnectionID]; ok {
				v.TotalCost = utils.PAdd(v.TotalCost, &localHit.TotalCost)
				v.DailyCostAtStartTime = utils.PAdd(v.DailyCostAtStartTime, &localHit.StartDateCost)
				v.DailyCostAtEndTime = utils.PAdd(v.DailyCostAtEndTime, &localHit.EndDateCost)
				res[localHit.ConnectionID] = v
			} else {
				res[localHit.ConnectionID] = inventoryApi.ConnectionData{
					ConnectionID:         localHit.ConnectionID,
					Count:                nil,
					OldCount:             nil,
					LastInventory:        nil,
					TotalCost:            &localHit.TotalCost,
					DailyCostAtStartTime: &localHit.StartDateCost,
					DailyCostAtEndTime:   &localHit.EndDateCost,
				}
			}
		}
		fmt.Println("ListConnectionsData part4 ", time.Now().Sub(performanceStartTime).Milliseconds())
	}

	return ctx.JSON(http.StatusOK, res)
}

func (h *HttpHandler) GetConnectionData(ctx echo.Context) error {
	aDB := analyticsDB.NewDatabase(h.db.orm)
	connectionId := ctx.Param("connectionId")
	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now()
	if endTimeStr != "" {
		endTimeUnix, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "endTime is not a valid integer")
		}
		endTime = time.Unix(endTimeUnix, 0)
	}
	startTimeStr := ctx.QueryParam("startTime")
	startTime := endTime.AddDate(0, 0, -7)
	if startTimeStr != "" {
		startTimeUnix, err := strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "startTime is not a valid integer")
		}
		startTime = time.Unix(startTimeUnix, 0)
	}

	res := inventoryApi.ConnectionData{
		ConnectionID: connectionId,
	}

	metrics, err := aDB.ListFilteredMetrics(nil, analyticsDB.MetricTypeAssets, nil, nil, false)
	if err != nil {
		return err
	}
	var metricIDs []string
	for _, m := range metrics {
		metricIDs = append(metricIDs, m.ID)
	}

	resourceCounts, err := es.FetchConnectionAnalyticsResourcesCountAtTime(h.client, nil, []string{connectionId}, metricIDs, endTime, EsFetchPageSize)
	for esConnectionId, resourceCountAndEvaluated := range resourceCounts {
		if esConnectionId != connectionId {
			continue
		}
		localCount := resourceCountAndEvaluated
		res.Count = utils.PAdd(res.Count, &localCount.ResourceCountsSum)
		if res.LastInventory == nil || res.LastInventory.IsZero() || res.LastInventory.Before(time.UnixMilli(localCount.LatestEvaluatedAt)) {
			res.LastInventory = utils.GetPointer(time.UnixMilli(localCount.LatestEvaluatedAt))
		}
	}

	oldResourceCounts, err := es.FetchConnectionAnalyticsResourcesCountAtTime(h.client, nil, []string{connectionId}, metricIDs, startTime, EsFetchPageSize)
	for esConnectionId, resourceCountAndEvaluated := range oldResourceCounts {
		if esConnectionId != connectionId {
			continue
		}
		localCount := resourceCountAndEvaluated
		res.OldCount = utils.PAdd(res.OldCount, &localCount.ResourceCountsSum)
		if res.LastInventory == nil || res.LastInventory.IsZero() || res.LastInventory.Before(time.UnixMilli(localCount.LatestEvaluatedAt)) {
			res.LastInventory = utils.GetPointer(time.UnixMilli(localCount.LatestEvaluatedAt))
		}
	}

	costs, err := es.FetchDailyCostHistoryByAccountsBetween(h.client, nil, []string{connectionId}, endTime, startTime, EsFetchPageSize)
	if err != nil {
		return err
	}
	startTimeCosts, err := es.FetchDailyCostHistoryByAccountsAtTime(h.client, nil, []string{connectionId}, startTime)
	if err != nil {
		return err
	}
	endTimeCosts, err := es.FetchDailyCostHistoryByAccountsAtTime(h.client, nil, []string{connectionId}, endTime)
	if err != nil {
		return err
	}

	for costConnectionId, costValue := range costs {
		if costConnectionId != connectionId {
			continue
		}
		localValue := costValue
		res.TotalCost = utils.PAdd(res.TotalCost, &localValue)
	}
	for costConnectionId, costValue := range startTimeCosts {
		if costConnectionId != connectionId {
			continue
		}
		localValue := costValue
		res.DailyCostAtStartTime = utils.PAdd(res.DailyCostAtStartTime, &localValue)
	}
	for costConnectionId, costValue := range endTimeCosts {
		if costConnectionId != connectionId {
			continue
		}
		localValue := costValue
		res.DailyCostAtEndTime = utils.PAdd(res.DailyCostAtEndTime, &localValue)
	}

	return ctx.JSON(http.StatusOK, res)
}

// ListQueries godoc
//
//	@Summary		List smart queries
//	@Description	Retrieving list of smart queries by specified filters
//	@Security		BearerToken
//	@Tags			smart_query
//	@Produce		json
//	@Param			request	body		inventoryApi.ListQueryRequest	true	"Request Body"
//	@Success		200		{object}	[]inventoryApi.SmartQueryItem
//	@Router			/inventory/api/v1/query [get]
func (h *HttpHandler) ListQueries(ctx echo.Context) error {
	var req inventoryApi.ListQueryRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var search *string
	if len(req.TitleFilter) > 0 {
		search = &req.TitleFilter
	}
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_GetQueriesWithFilters", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetQueriesWithFilters")

	queries, err := h.db.GetQueriesWithFilters(search)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.End()

	var result []inventoryApi.SmartQueryItem
	for _, item := range queries {
		category := ""

		tags := map[string]string{}
		if item.IsPopular {
			tags["popular"] = "true"
		}
		result = append(result, inventoryApi.SmartQueryItem{
			ID:         item.ID,
			Connectors: source.ParseTypes(item.Connectors),
			Title:      item.Title,
			Category:   category,
			Query:      item.Query,
			Tags:       tags,
		})
	}
	return ctx.JSON(200, result)
}

// RunQuery godoc
//
//	@Summary		Run query
//	@Description	Run provided smart query and returns the result.
//	@Security		BearerToken
//	@Tags			smart_query
//	@Accepts		json
//	@Produce		json
//	@Param			request	body		inventoryApi.RunQueryRequest	true	"Request Body"
//	@Param			accept	header		string							true	"Accept header"	Enums(application/json,text/csv)
//	@Success		200		{object}	inventoryApi.RunQueryResponse
//	@Router			/inventory/api/v1/query/run [post]
func (h *HttpHandler) RunQuery(ctx echo.Context) error {
	var req inventoryApi.RunQueryRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if req.Query == nil || *req.Query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Query is required")
	}
	// tracer :
	outputS, span := tracer.Start(ctx.Request().Context(), "new_RunSmartQuery", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_RunSmartQuery")

	resp, err := h.RunSmartQuery(outputS, *req.Query, *req.Query, &req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("query title ", resp.Title),
	))
	span.End()
	return ctx.JSON(200, resp)
}

// GetRecentRanQueries godoc
//
//	@Summary		List recently ran queries
//	@Description	List queries which have been run recently
//	@Security		BearerToken
//	@Tags			smart_query
//	@Accepts		json
//	@Produce		json
//	@Success		200	{object}	[]inventoryApi.SmartQueryHistory
//	@Router			/inventory/api/v1/query/run/history [get]
func (h *HttpHandler) GetRecentRanQueries(ctx echo.Context) error {
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_GetQueryHistory", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetQueryHistory")

	smartQueryHistories, err := h.db.GetQueryHistory()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.logger.Error("Failed to get query history", zap.Error(err))
		return err
	}
	span.End()

	res := make([]inventoryApi.SmartQueryHistory, 0, len(smartQueryHistories))
	for _, history := range smartQueryHistories {
		res = append(res, history.ToApi())
	}

	return ctx.JSON(200, res)
}

func (h *HttpHandler) CountResources(ctx echo.Context) error {
	timeAt := time.Now()
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_ListFilteredResourceTypes", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_ListFilteredResourceTypes")

	resourceTypes, err := h.db.ListFilteredResourceTypes(nil, nil, nil, nil, true)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	span.End()

	resourceTypeNames := make([]string, 0, len(resourceTypes))
	for _, resourceType := range resourceTypes {
		resourceTypeNames = append(resourceTypeNames, resourceType.ResourceType)
	}

	metricsIndexed, err := es.FetchConnectorResourceTypeCountAtTime(h.client, nil, timeAt, resourceTypeNames, EsFetchPageSize)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	totalCount := 0
	for _, count := range metricsIndexed {
		totalCount += count
	}
	return ctx.JSON(http.StatusOK, totalCount)
}

func (h *HttpHandler) RunSmartQuery(ctx context.Context, title, query string, req *inventoryApi.RunQueryRequest) (*inventoryApi.RunQueryResponse, error) {
	var err error
	lastIdx := (req.Page.No - 1) * req.Page.Size

	direction := inventoryApi.DirectionType("")
	orderBy := ""
	if req.Sorts != nil && len(req.Sorts) > 0 {
		direction = req.Sorts[0].Direction
		orderBy = req.Sorts[0].Field
	}
	if len(req.Sorts) > 1 {
		return nil, errors.New("multiple sort items not supported")
	}

	h.logger.Info("executing smart query", zap.String("query", query))
	res, err := h.steampipeConn.Query(ctx, query, &lastIdx, &req.Page.Size, orderBy, steampipe.DirectionType(direction))
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	// tracer :
	_, span := tracer.Start(ctx, "new_UpdateQueryHistory", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_UpdateQueryHistory")

	err = h.db.UpdateQueryHistory(query)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.logger.Error("failed to update query history", zap.Error(err))
		return nil, err
	}
	span.End()

	resp := inventoryApi.RunQueryResponse{
		Title:   title,
		Query:   query,
		Headers: res.Headers,
		Result:  res.Data,
	}
	return &resp, nil
}

func (h *HttpHandler) ListInsightResults(ctx echo.Context) error {
	var err error
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	timeStr := ctx.QueryParam("time")
	timeAt := time.Now().Unix()
	if timeStr != "" {
		timeAt, err = strconv.ParseInt(timeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
	}
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")

	insightIdListStr := httpserver.QueryArrayParam(ctx, "insightId")
	if len(insightIdListStr) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "insight id is required")
	}
	insightIdList := make([]uint, 0, len(insightIdListStr))
	for _, idStr := range insightIdListStr {
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid insight id")
		}
		insightIdList = append(insightIdList, uint(id))
	}

	var insightValues map[uint][]insight.InsightResource
	if timeStr != "" {
		insightValues, err = es.FetchInsightValueAtTime(h.client, time.Unix(timeAt, 0), connectors, connectionIDs, insightIdList, true)
	} else {
		insightValues, err = es.FetchInsightValueAtTime(h.client, time.Unix(timeAt, 0), connectors, connectionIDs, insightIdList, false)
	}
	if err != nil {
		return err
	}

	firstAvailable, err := es.FetchInsightValueAfter(h.client, time.Unix(timeAt, 0), connectors, connectionIDs, insightIdList)
	if err != nil {
		return err
	}

	for insightId, _ := range firstAvailable {
		if results, ok := insightValues[insightId]; ok && len(results) > 0 {
			continue
		}
		insightValues[insightId] = firstAvailable[insightId]
	}

	return ctx.JSON(http.StatusOK, insightValues)
}

func (h *HttpHandler) GetInsightResult(ctx echo.Context) error {
	insightId, err := strconv.ParseUint(ctx.Param("insightId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid insight id")
	}
	timeStr := ctx.QueryParam("time")
	timeAt := time.Now().Unix()
	if timeStr != "" {
		timeAt, err = strconv.ParseInt(timeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
	}
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	if len(connectionIDs) == 0 {
		connectionIDs = nil
	}

	var insightResults map[uint][]insight.InsightResource
	if timeStr != "" {
		insightResults, err = es.FetchInsightValueAtTime(h.client, time.Unix(timeAt, 0), nil, connectionIDs, []uint{uint(insightId)}, true)
	} else {
		insightResults, err = es.FetchInsightValueAtTime(h.client, time.Unix(timeAt, 0), nil, connectionIDs, []uint{uint(insightId)}, false)
	}
	if err != nil {
		return err
	}

	firstAvailable, err := es.FetchInsightValueAfter(h.client, time.Unix(timeAt, 0), nil, connectionIDs, []uint{uint(insightId)})
	if err != nil {
		return err
	}

	for insightId, _ := range firstAvailable {
		if results, ok := insightResults[insightId]; ok && len(results) > 0 {
			continue
		}
		insightResults[insightId] = firstAvailable[insightId]
	}

	if insightResult, ok := insightResults[uint(insightId)]; ok {
		return ctx.JSON(http.StatusOK, insightResult)
	} else {
		return echo.NewHTTPError(http.StatusNotFound, "no data for insight found")
	}
}

func (h *HttpHandler) GetInsightTrendResults(ctx echo.Context) error {
	insightId, err := strconv.ParseUint(ctx.Param("insightId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid insight id")
	}
	var startTime, endTime time.Time
	endTime = time.Now()
	if timeStr := ctx.QueryParam("endTime"); timeStr != "" {
		timeInt, err := strconv.ParseInt(timeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
		endTime = time.Unix(timeInt, 0)
	}
	if timeStr := ctx.QueryParam("startTime"); timeStr != "" {
		timeInt, err := strconv.ParseInt(timeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
		startTime = time.Unix(timeInt, 0)
	} else {
		startTime = endTime.Add(-time.Hour * 24 * 30)
	}

	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")

	dataPointCount := int(endTime.Sub(startTime).Hours() / 24)
	insightResults, err := es.FetchInsightAggregatedPerQueryValuesBetweenTimes(h.client, startTime, endTime, dataPointCount, nil, connectionIDs, []uint{uint(insightId)})
	if err != nil {
		return err
	}
	if insightResult, ok := insightResults[uint(insightId)]; ok {
		return ctx.JSON(http.StatusOK, insightResult)
	} else {
		return echo.NewHTTPError(http.StatusNotFound, "no data for insight found")
	}
}

func (h *HttpHandler) ListResourceTypeMetadata(ctx echo.Context) error {
	tagMap := model.TagStringsToTagMap(httpserver.QueryArrayParam(ctx, "tag"))
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	serviceNames := httpserver.QueryArrayParam(ctx, "service")
	resourceTypeNames := httpserver.QueryArrayParam(ctx, "resourceType")
	summarized := strings.ToLower(ctx.QueryParam("summarized")) == "true"
	pageNumber, pageSize, err := utils.PageConfigFromStrings(ctx.QueryParam("pageNumber"), ctx.QueryParam("pageSize"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_ListFilteredResourceTypes", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_ListFilteredResourceTypes")

	resourceTypes, err := h.db.ListFilteredResourceTypes(tagMap, resourceTypeNames, serviceNames, connectors, summarized)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.End()

	var resourceTypeMetadata []inventoryApi.ResourceType
	tableCountMap := make(map[string]int)
	insightList, err := h.complianceClient.ListInsightsMetadata(httpclient.FromEchoContext(ctx), connectors)
	if err != nil {
		return err
	}
	for _, insightEntity := range insightList {
		for _, insightTable := range insightEntity.Query.ListOfTables {
			tableCountMap[insightTable]++
		}
	}

	for _, resourceType := range resourceTypes {
		apiResourceType := resourceType.ToApi()

		var table string
		switch resourceType.Connector {
		case source.CloudAWS:
			table = awsSteampipe.ExtractTableName(resourceType.ResourceType)
		case source.CloudAzure:
			table = azureSteampipe.ExtractTableName(resourceType.ResourceType)
		}
		insightTableCount := 0
		if table != "" {
			insightTableCount = tableCountMap[table]
		}
		apiResourceType.InsightsCount = utils.GetPointerOrNil(insightTableCount)

		// TODO: add compliance count

		resourceTypeMetadata = append(resourceTypeMetadata, apiResourceType)
	}

	sort.Slice(resourceTypeMetadata, func(i, j int) bool {
		return resourceTypeMetadata[i].ResourceType < resourceTypeMetadata[j].ResourceType
	})

	result := inventoryApi.ListResourceTypeMetadataResponse{
		TotalResourceTypeCount: len(resourceTypeMetadata),
		ResourceTypes:          utils.Paginate(pageNumber, pageSize, resourceTypeMetadata),
	}

	return ctx.JSON(http.StatusOK, result)
}

func (h *HttpHandler) connectionsFilter(filter map[string]interface{}) ([]string, error) {
	var connections []string
	allConnections, err := h.onboardClient.ListSources(&httpclient.Context{UserRole: authApi.KaytuAdminRole}, []source.Type{source.CloudAWS, source.CloudAzure})
	if err != nil {
		return nil, err
	}
	var allConnectionsStr []string
	for _, c := range allConnections {
		allConnectionsStr = append(allConnectionsStr, c.ID.String())
	}
	for key, value := range filter {
		if key == "Match" {
			dimFilter := value.(map[string]interface{})
			if dimKey, ok := dimFilter["Key"]; ok {
				if dimKey == "ConnectionID" {
					connections, err = dimFilterFunction(dimFilter, allConnectionsStr)
					if err != nil {
						return nil, err
					}
					h.logger.Warn(fmt.Sprintf("===Dim Filter Function on filter %v, result: %v", dimFilter, connections))
				} else if dimKey == "Provider" {
					providers, err := dimFilterFunction(dimFilter, []string{"AWS", "Azure"})
					if err != nil {
						return nil, err
					}
					h.logger.Warn(fmt.Sprintf("===Dim Filter Function on filter %v, result: %v", dimFilter, providers))
					for _, c := range allConnections {
						if arrayContains(providers, c.Connector.String()) {
							connections = append(connections, c.ID.String())
						}
					}
				} else if dimKey == "ConnectionGroup" {
					allGroups, err := h.onboardClient.ListConnectionGroups(&httpclient.Context{UserRole: authApi.KaytuAdminRole})
					if err != nil {
						return nil, err
					}
					allGroupsMap := make(map[string][]string)
					var allGroupsStr []string
					for _, g := range allGroups {
						allGroupsMap[g.Name] = make([]string, 0, len(g.ConnectionIds))
						for _, cid := range g.ConnectionIds {
							allGroupsMap[g.Name] = append(allGroupsMap[g.Name], cid)
							allGroupsStr = append(allGroupsStr, cid)
						}
					}
					groups, err := dimFilterFunction(dimFilter, allGroupsStr)
					if err != nil {
						return nil, err
					}
					h.logger.Warn(fmt.Sprintf("===Dim Filter Function on filter %v, result: %v", dimFilter, groups))

					for _, g := range groups {
						for _, conn := range allGroupsMap[g] {
							if !arrayContains(connections, conn) {
								connections = append(connections, conn)
							}
						}
					}
				} else if dimKey == "ConnectionName" {
					var allConnectionsNames []string
					for _, c := range allConnections {
						allConnectionsNames = append(allConnectionsNames, c.ConnectionName)
					}
					connectionNames, err := dimFilterFunction(dimFilter, allConnectionsNames)
					if err != nil {
						return nil, err
					}
					h.logger.Warn(fmt.Sprintf("===Dim Filter Function on filter %v, result: %v", dimFilter, connectionNames))
					for _, conn := range allConnections {
						if arrayContains(connectionNames, conn.ConnectionName) {
							connections = append(connections, conn.ID.String())
						}
					}

				}
			} else {
				return nil, fmt.Errorf("missing key")
			}
		} else if key == "AND" {
			var andFilters []map[string]interface{}
			for _, v := range value.([]interface{}) {
				andFilter := v.(map[string]interface{})
				andFilters = append(andFilters, andFilter)
			}
			counter := make(map[string]int)
			for _, f := range andFilters {
				values, err := h.connectionsFilter(f)
				if err != nil {
					return nil, err
				}
				for _, v := range values {
					if c, ok := counter[v]; ok {
						counter[v] = c + 1
					} else {
						counter[v] = 1
					}
					if counter[v] == len(andFilters) {
						connections = append(connections, v)
					}
				}
			}
		} else if key == "OR" {
			var orFilters []map[string]interface{}
			for _, v := range value.([]interface{}) {
				orFilter := v.(map[string]interface{})
				orFilters = append(orFilters, orFilter)
			}
			for _, f := range orFilters {
				values, err := h.connectionsFilter(f)
				if err != nil {
					return nil, err
				}
				for _, v := range values {
					if !arrayContains(connections, v) {
						connections = append(connections, v)
					}
				}
			}
		} else {
			return nil, fmt.Errorf("invalid key: ", key)
		}
	}
	return connections, nil
}

func dimFilterFunction(dimFilter map[string]interface{}, allValues []string) ([]string, error) {
	var values []string
	for _, v := range dimFilter["Values"].([]interface{}) {
		values = append(values, fmt.Sprintf("%v", v))
	}
	var output []string
	if matchOption, ok := dimFilter["MatchOption"]; ok {
		switch {
		case strings.Contains(matchOption.(string), "EQUAL"):
			output = values
		case strings.Contains(matchOption.(string), "STARTS_WITH"):
			for _, v := range values {
				for _, conn := range allValues {
					if strings.HasPrefix(conn, v) {
						if !arrayContains(output, conn) {
							output = append(output, conn)
						}
					}
				}
			}
		case strings.Contains(matchOption.(string), "ENDS_WITH"):
			for _, v := range values {
				for _, conn := range allValues {
					if strings.HasSuffix(conn, v) {
						if !arrayContains(output, conn) {
							output = append(output, conn)
						}
					}
				}
			}
		case strings.Contains(matchOption.(string), "CONTAINS"):
			for _, v := range values {
				for _, conn := range allValues {
					if strings.Contains(conn, v) {
						if !arrayContains(output, conn) {
							output = append(output, conn)
						}
					}
				}
			}
		default:
			return nil, fmt.Errorf("invalid option")
		}
		if strings.HasPrefix(matchOption.(string), "~") {
			var notOutput []string
			for _, v := range allValues {
				if !arrayContains(output, v) {
					notOutput = append(notOutput, v)
				}
			}
			return notOutput, nil
		}
	} else {
		output = values
	}
	return output, nil
}

func arrayContains(array []string, key string) bool {
	for _, v := range array {
		if v == key {
			return true
		}
	}
	return false
}
