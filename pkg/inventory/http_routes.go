package inventory

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/utils"

	"github.com/turbot/steampipe-plugin-sdk/grpc/proto"
	"gitlab.com/keibiengine/keibi-engine/pkg/steampipe"

	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
	"github.com/jackc/pgx/v4"
	"github.com/labstack/echo/v4"
	"github.com/turbot/steampipe-plugin-sdk/logging"
	"github.com/turbot/steampipe-plugin-sdk/plugin/context_key"
	compliance_report "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report"
	pagination "gitlab.com/keibiengine/keibi-engine/pkg/internal/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
)

const EsFetchPageSize = 10000

func extractContext(ctx echo.Context) context.Context {
	cc := ctx.Request().Context()
	logger := logging.NewLogger(&hclog.LoggerOptions{DisableTime: true})
	log.SetOutput(logger.StandardWriter(&hclog.StandardLoggerOptions{InferLevels: true}))
	log.SetPrefix("")
	log.SetFlags(0)
	return context.WithValue(cc, context_key.Logger, logger)
}

func (h *HttpHandler) Register(v1 *echo.Group) {
	v1.GET("/locations/:provider", h.GetLocations)

	v1.POST("/resources", h.GetAllResources)
	v1.POST("/resources/azure", h.GetAzureResources)
	v1.POST("/resources/aws", h.GetAWSResources)

	v1.POST("/resource", h.GetResource)

	v1.GET("/resources/trend", h.GetResourceGrowthTrend)
	v1.GET("/resources/distribution", h.GetResourceDistribution)
	v1.GET("/resources/top/accounts", h.GetTopAccountsByResourceCount)
	v1.GET("/resources/top/services", h.GetTopServicesByResourceCount)
	v1.GET("/accounts/resource/count", h.GetAccountsResourceCount)

	v1.GET("/query", h.ListQueries)
	v1.GET("/query/count", h.CountQueries)
	v1.POST("/query/:queryId", h.RunQuery)

	v1.GET("/reports/compliance/:sourceId", h.GetComplianceReports)
	v1.GET("/reports/compliance/:sourceId/:reportId", h.GetComplianceReports)

	v1.GET("/benchmarks", h.GetBenchmarks)
	v1.GET("/benchmarks/tags", h.GetBenchmarkTags)
	v1.GET("/benchmarks/:benchmarkId", h.GetBenchmarkDetails)
	v1.GET("/benchmarks/:benchmarkId/policies", h.GetPolicies)
	v1.GET("/benchmarks/:benchmarkId/:sourceId/result", h.GetBenchmarkResult)
	v1.GET("/benchmarks/:benchmarkId/:sourceId/result/policies", h.GetResultPolicies)

	v1.GET("/benchmarks/history/list/:provider/:createdAt", h.GetBenchmarksInTime)
	v1.GET("/benchmarks/:benchmarkId/:sourceId/compliance/trend", h.GetBenchmarkComplianceTrend)
	v1.GET("/benchmarks/:benchmarkId/:createdAt/accounts/compliance", h.GetBenchmarkAccountCompliance)
	v1.GET("/benchmarks/:benchmarkId/:createdAt/accounts", h.GetBenchmarkAccounts)

	v1.GET("/benchmarks/:provider/list", h.GetBenchmarkDetails)
	v1.GET("/compliancy/trend", h.GetCompliancyTrend)

	v1.GET("/benchmarks/count", h.CountBenchmarks)
	v1.GET("/policies/count", h.CountPolicies)
}

// GetBenchmarksInTime godoc
// @Summary      Returns all benchmark existed at the specified time
// @Description  You should fetch the benchmark report times from /benchmarks/history/:year/:month/:day
// @Tags         benchmarks
// @Accept       json
// @Produce      json
// @Param        provider   path      string  true  "Provider"  Enums(AWS,Azure,All)
// @Param        createdAt  path      string  true  "CreatedAt"
// @Success      200        {object}  []api.Benchmark
// @Router       /inventory/api/v1/benchmarks/history/list/{provider}/{createdAt} [get]
func (h *HttpHandler) GetBenchmarksInTime(ctx echo.Context) error {
	providerStr := ctx.Param("provider")
	tim := ctx.Param("createdAt")
	timeInt, err := strconv.ParseInt(tim, 10, 64)
	if err != nil {
		return err
	}

	var provider *string
	if strings.ToLower(providerStr) == "aws" {
		providerStr = "AWS"
		provider = &providerStr
	} else if strings.ToLower(providerStr) == "azure" {
		providerStr = "Azure"
		provider = &providerStr
	} else if strings.ToLower(providerStr) == "all" {
	} else {
		return echo.NewHTTPError(400, "Invalid provider")
	}

	uniqueBenchmarkIDs := map[string]api.Benchmark{}
	var searchAfter []interface{}
	for {
		query, err := compliance_report.QueryBenchmarks(provider, timeInt, 2, EsFetchPageSize, searchAfter)
		if err != nil {
			return err
		}

		var response compliance_report.ReportQueryResponse
		err = h.client.Search(context.Background(), compliance_report.ComplianceReportIndex, query, &response)
		if err != nil {
			return err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			uniqueBenchmarkIDs[hit.Source.Group.ID] = api.Benchmark{
				ID:          hit.Source.Group.ID,
				Title:       hit.Source.Group.Title,
				Description: hit.Source.Group.Description,
				Provider:    api.SourceType(hit.Source.Provider),
				State:       api.BenchmarkStateEnabled,
				Tags:        nil,
			}
			searchAfter = hit.Sort
		}
	}

	var resp []api.Benchmark
	for _, v := range uniqueBenchmarkIDs {
		resp = append(resp, v)
	}
	return ctx.JSON(http.StatusOK, resp)
}

// GetBenchmarkComplianceTrend godoc
// @Summary  Returns trend of a benchmark compliance for specific account
// @Tags         benchmarks
// @Accept       json
// @Produce      json
// @Param    benchmarkId  path      string  true  "BenchmarkID"
// @Param    sourceId     path      string  true  "SourceID"
// @Param    timeWindow  query     string  true  "Time Window"  Enums(24h,1w,3m,1y,max)
// @Success  200          {object}  []api.ComplianceTrendDataPoint
// @Router   /inventory/api/v1/benchmarks/{benchmarkId}/{sourceId}/compliance/trend [get]
func (h *HttpHandler) GetBenchmarkComplianceTrend(ctx echo.Context) error {
	benchmarkId := ctx.Param("benchmarkId")
	sourceUUID, err := uuid.Parse(ctx.Param("sourceId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
	}

	var fromTime, toTime int64
	toTime = time.Now().UnixMilli()

	switch ctx.QueryParam("timeWindow") {
	case "24h":
		fromTime = time.Now().Add(-24 * time.Hour).UnixMilli()
	case "1w":
		fromTime = time.Now().Add(-24 * 7 * time.Hour).UnixMilli()
	case "3m":
		fromTime = time.Now().Add(-24 * 30 * 3 * time.Hour).UnixMilli()
	case "1y":
		fromTime = time.Now().Add(-24 * 365 * time.Hour).UnixMilli()
	case "max":
		fromTime = 0
	default:
		fromTime = time.Now().Add(-24 * time.Hour).UnixMilli()
	}

	var hits []compliance_report.ReportQueryHit
	var searchAfter []interface{}
	for {
		query, err := compliance_report.QueryTrend(sourceUUID, benchmarkId, fromTime, toTime, EsFetchPageSize, searchAfter)
		if err != nil {
			return err
		}

		var response compliance_report.ReportQueryResponse
		err = h.client.Search(context.Background(), compliance_report.ComplianceReportIndex, query, &response)
		if err != nil {
			return err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			hits = append(hits, hit)
			searchAfter = hit.Sort
		}
	}

	var rhits []es.ResourceGrowthQueryHit
	searchAfter = nil
	for {
		query, err := es.FindResourceGrowthTrendQuery(&sourceUUID, nil,
			fromTime, toTime, EsFetchPageSize, searchAfter)
		if err != nil {
			return err
		}

		var response es.ResourceGrowthQueryResponse
		err = h.client.Search(context.Background(), describe.SourceResourcesSummary, query, &response)
		if err != nil {
			return err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			rhits = append(rhits, hit)
			searchAfter = hit.Sort
		}
	}

	var resp []api.ComplianceTrendDataPoint
	for _, hit := range hits {
		var total int64 = 0
		for _, rhit := range rhits {
			if rhit.Source.DescribedAt == hit.Source.DescribedAt {
				total = int64(rhit.Source.ResourceCount)
				break
			}
		}

		resp = append(resp, api.ComplianceTrendDataPoint{
			Timestamp:      hit.Source.DescribedAt,
			Compliant:      int64(hit.Source.Group.Summary.Status.OK),
			TotalResources: total,
		})
	}
	return ctx.JSON(http.StatusOK, resp)
}

// GetResourceGrowthTrend godoc
// @Summary  Returns trend of resource growth for specific account
// @Tags         benchmarks
// @Accept       json
// @Produce      json
// @Param    sourceId    query     string  true  "SourceID"
// @Param    provider    query     string  true  "Provider"
// @Param    timeWindow  query     string  true  "Time Window"  Enums(24h,1w,3m,1y,max)
// @Success  200         {object}  []api.TrendDataPoint
// @Router   /inventory/api/v1/resources/trend [get]
func (h *HttpHandler) GetResourceGrowthTrend(ctx echo.Context) error {
	provider := ctx.QueryParam("provider")
	sourceID := ctx.QueryParam("sourceId")
	timeWindow := ctx.QueryParam("timeWindow")

	if timeWindow == "" {
		timeWindow = "24h"
	}

	if provider == "" && sourceID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "you should specify either provider or sourceId")
	}

	var providerPtr *string
	if provider != "" {
		providerPtr = &provider
	}

	var sourceUUID *uuid.UUID
	var err error
	if sourceID != "" {
		u, err := uuid.Parse(sourceID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
		}
		sourceUUID = &u
	}

	var fromTime, toTime int64
	toTime = time.Now().UnixMilli()
	tw, err := utils.ParseTimeWindow(ctx.QueryParam("timeWindow"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid timeWindow")
	}
	fromTime = time.Now().Add(-1 * tw).UnixMilli()

	var hits []es.ResourceGrowthQueryHit
	var searchAfter []interface{}
	for {
		query, err := es.FindResourceGrowthTrendQuery(sourceUUID, providerPtr,
			fromTime, toTime, EsFetchPageSize, searchAfter)
		if err != nil {
			return err
		}

		var response es.ResourceGrowthQueryResponse
		err = h.client.Search(context.Background(), describe.SourceResourcesSummary, query, &response)
		if err != nil {
			return err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			hits = append(hits, hit)
			searchAfter = hit.Sort
		}
	}

	datapoints := map[int64]int{}
	for _, hit := range hits {
		datapoints[hit.Source.DescribedAt] += hit.Source.ResourceCount
	}

	var resp []api.TrendDataPoint
	for k, v := range datapoints {
		resp = append(resp, api.TrendDataPoint{
			Timestamp: k,
			Value:     int64(v),
		})
	}
	sort.SliceStable(resp, func(i, j int) bool {
		return resp[i].Timestamp < resp[j].Timestamp
	})
	return ctx.JSON(http.StatusOK, resp)
}

// GetCompliancyTrend godoc
// @Summary  Returns trend of compliancy for specific account
// @Tags         benchmarks
// @Accept       json
// @Produce      json
// @Param    sourceId     query     string  true  "SourceID"
// @Param    provider     query     string  true  "Provider"
// @Param    timeWindow   query     string  true  "Time Window"  Enums(24h,1w,3m,1y,max)
// @Success  200          {object}  []api.TrendDataPoint
// @Router   /inventory/api/v1/compliancy/trend [get]
func (h *HttpHandler) GetCompliancyTrend(ctx echo.Context) error {
	provider := ctx.QueryParam("provider")
	sourceID := ctx.QueryParam("sourceId")
	timeWindow := ctx.QueryParam("timeWindow")

	if timeWindow == "" {
		timeWindow = "24h"
	}

	if provider == "" && sourceID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "you should specify either provider or sourceId")
	}

	var providerPtr *string
	if provider != "" {
		providerPtr = &provider
	}

	var sourceUUID *uuid.UUID
	var err error
	if sourceID != "" {
		u, err := uuid.Parse(sourceID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
		}
		sourceUUID = &u
	}

	var fromTime, toTime int64
	toTime = time.Now().UnixMilli()
	tw, err := utils.ParseTimeWindow(ctx.QueryParam("timeWindow"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid timeWindow")
	}
	fromTime = time.Now().Add(-1 * tw).UnixMilli()

	var hits []es.ComplianceTrendQueryHit
	var searchAfter []interface{}
	for {
		query, err := es.FindCompliancyTrendQuery(sourceUUID, providerPtr,
			fromTime, toTime, EsFetchPageSize, searchAfter)
		if err != nil {
			return err
		}

		var response es.ComplianceTrendQueryResponse
		err = h.client.Search(context.Background(), describe.SourceResourcesSummary, query, &response)
		if err != nil {
			return err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			hits = append(hits, hit)
			searchAfter = hit.Sort
		}
	}

	datapoints := map[int64]int{}
	for _, hit := range hits {
		datapoints[hit.Source.DescribedAt] += hit.Source.CompliantResourceCount
	}

	var resp []api.TrendDataPoint
	for k, v := range datapoints {
		resp = append(resp, api.TrendDataPoint{
			Timestamp: k,
			Value:     int64(v),
		})
	}
	sort.SliceStable(resp, func(i, j int) bool {
		return resp[i].Timestamp < resp[j].Timestamp
	})
	return ctx.JSON(http.StatusOK, resp)
}

// GetTopAccountsByResourceCount godoc
// @Summary  Returns top n accounts of specified provider by resource count
// @Tags     benchmarks
// @Accept   json
// @Produce  json
// @Param    count     query     int     true  "count"
// @Param    provider  query     string  true  "Provider"
// @Success  200       {object}  []api.TopAccountResponse
// @Router   /inventory/api/v1/resources/top/accounts [get]
func (h *HttpHandler) GetTopAccountsByResourceCount(ctx echo.Context) error {
	provider := ctx.QueryParam("provider")
	count, err := strconv.Atoi(ctx.QueryParam("count"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid count")
	}

	query, err := es.FindTopAccountsQuery(provider, count)
	if err != nil {
		return err
	}

	var response es.ResourceGrowthQueryResponse
	err = h.client.Search(context.Background(), describe.SourceResourcesSummary, query, &response)
	if err != nil {
		return err
	}

	var res []api.TopAccountResponse
	for _, hit := range response.Hits.Hits {
		res = append(res, api.TopAccountResponse{
			SourceID:      hit.Source.SourceID,
			ResourceCount: hit.Source.ResourceCount,
		})
	}
	return ctx.JSON(http.StatusOK, res)
}

// GetTopServicesByResourceCount godoc
// @Summary  Returns top n services of specified provider by resource count
// @Tags     benchmarks
// @Accept   json
// @Produce  json
// @Param    count     query     int     true  "count"
// @Param    provider  query     string  true  "Provider"
// @Success  200       {object}  []api.TopAccountResponse
// @Router   /inventory/api/v1/resources/top/services [get]
func (h *HttpHandler) GetTopServicesByResourceCount(ctx echo.Context) error {
	provider := ctx.QueryParam("provider")
	count, err := strconv.Atoi(ctx.QueryParam("count"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid count")
	}

	query, err := es.FindTopServicesQuery(provider, count)
	if err != nil {
		return err
	}

	var response es.TopServicesQueryResponse
	err = h.client.Search(context.Background(), describe.SourceResourcesSummary, query, &response)
	if err != nil {
		return err
	}

	var res []api.TopServicesResponse
	for _, hit := range response.Hits.Hits {
		res = append(res, api.TopServicesResponse{
			ServiceName:   hit.Source.ServiceName,
			ResourceCount: hit.Source.ResourceCount,
		})
	}
	return ctx.JSON(http.StatusOK, res)
}

// GetAccountsResourceCount godoc
// @Summary  Returns resource count of accounts
// @Tags         benchmarks
// @Accept       json
// @Produce      json
// @Param    provider     query     string  true  "Provider"
// @Success  200          {object}  []api.TopAccountResponse
// @Router   /inventory/api/v1/accounts/resource/count [get]
func (h *HttpHandler) GetAccountsResourceCount(ctx echo.Context) error {
	provider := ctx.QueryParam("provider")

	var searchAfter []interface{}
	var res []api.TopAccountResponse

	for {
		query, err := es.ListAccountResourceCountQuery(provider, EsFetchPageSize, searchAfter)
		if err != nil {
			return err
		}

		var response es.ResourceGrowthQueryResponse
		err = h.client.Search(context.Background(), describe.SourceResourcesSummary, query, &response)
		if err != nil {
			return err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			searchAfter = hit.Sort
			res = append(res, api.TopAccountResponse{
				SourceID:      hit.Source.SourceID,
				ResourceCount: hit.Source.ResourceCount,
			})
		}
	}
	return ctx.JSON(http.StatusOK, res)
}

// GetResourceDistribution godoc
// @Summary  Returns distribution of resource for specific account
// @Tags     benchmarks
// @Accept   json
// @Produce  json
// @Param    sourceId    query     string  true  "SourceID"
// @Param    provider    query     string  true  "Provider"
// @Param    timeWindow   query     string  true  "Time Window"  Enums(24h,1w,3m,1y,max)
// @Success  200         {object}  map[string]int
// @Router   /inventory/api/v1/resources/distribution [get]
func (h *HttpHandler) GetResourceDistribution(ctx echo.Context) error {
	provider := ctx.QueryParam("provider")
	sourceID := ctx.QueryParam("sourceId")

	if provider == "" && sourceID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "you should specify either provider or sourceId")
	}

	var providerPtr *string
	if provider != "" {
		providerPtr = &provider
	}

	var sourceUUID *uuid.UUID
	if sourceID != "" {
		u, err := uuid.Parse(sourceID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
		}
		sourceUUID = &u
	}

	locationDistribution := map[string]int{}
	var searchAfter []interface{}
	for {
		query, err := es.FindLocationDistributionQuery(sourceUUID, providerPtr, EsFetchPageSize, searchAfter)
		if err != nil {
			return err
		}

		var response es.LocationDistributionQueryResponse
		err = h.client.Search(context.Background(), describe.SourceResourcesSummary, query, &response)
		if err != nil {
			return err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			for k, v := range hit.Source.LocationDistribution {
				locationDistribution[k] += v
			}
			searchAfter = hit.Sort
		}
	}
	return ctx.JSON(http.StatusOK, locationDistribution)
}

// GetBenchmarkAccountCompliance godoc
// @Summary  Returns no of compliant & non-compliant accounts
// @Tags         benchmarks
// @Accept       json
// @Produce      json
// @Param    benchmarkId  path      string  true  "BenchmarkID"
// @Param    createdAt    path      string  true  "CreatedAt"
// @Success  200          {object}  api.BenchmarkAccountComplianceResponse
// @Router   /inventory/api/v1/benchmarks/{benchmarkId}/{createdAt}/accounts/compliance [get]
func (h *HttpHandler) GetBenchmarkAccountCompliance(ctx echo.Context) error {
	benchmarkId := ctx.Param("benchmarkId")
	tim, err := strconv.ParseInt(ctx.Param("createdAt"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid time")
	}

	var searchAfter []interface{}
	var resp api.BenchmarkAccountComplianceResponse
	for {
		query, err := compliance_report.QueryProviderResult(benchmarkId, tim, "asc", EsFetchPageSize, searchAfter)
		if err != nil {
			return err
		}

		var response compliance_report.AccountReportQueryResponse
		err = h.client.Search(context.Background(), compliance_report.AccountReportIndex, query, &response)
		if err != nil {
			return err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			if hit.Source.TotalResources == hit.Source.TotalCompliant {
				resp.TotalCompliantAccounts++
			} else {
				resp.TotalNonCompliantAccounts++
			}
			searchAfter = hit.Sort
		}
	}

	return ctx.JSON(http.StatusOK, resp)
}

// GetBenchmarkAccounts godoc
// @Summary  Returns list of accounts compliance scores ordered by compliance ratio
// @Tags     benchmarks
// @Accept   json
// @Produce  json
// @Param    benchmarkId  path      string  true  "BenchmarkID"
// @Param    createdAt    path      string  true  "CreatedAt"
// @Param    order        query     string  true  "Order"  Enums(asc,desc)
// @Param    size         query     int64   true  "Size"
// @Success  200          {object}  api.BenchmarkAccountComplianceResponse
// @Router   /inventory/api/v1/benchmarks/{benchmarkId}/{createdAt}/accounts [get]
func (h *HttpHandler) GetBenchmarkAccounts(ctx echo.Context) error {
	benchmarkId := ctx.Param("benchmarkId")
	order := ctx.QueryParam("order")
	order = strings.ToLower(order)
	if order != "asc" && order != "desc" {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid order:"+order)
	}

	tim, err := strconv.ParseInt(ctx.Param("createdAt"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid createdAt")
	}

	size, err := strconv.ParseInt(ctx.QueryParam("size"), 10, 64)
	if err != nil || size <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid size")
	}

	query, err := compliance_report.QueryProviderResult(benchmarkId, tim, order, int32(size), nil)
	if err != nil {
		return err
	}

	var response compliance_report.AccountReportQueryResponse
	err = h.client.Search(context.Background(), compliance_report.AccountReportIndex, query, &response)
	if err != nil {
		return err
	}

	var reports []compliance_report.AccountReport
	for _, hit := range response.Hits.Hits {
		reports = append(reports, hit.Source)
	}
	return ctx.JSON(http.StatusOK, reports)
}

// GetBenchmarks godoc
// @Summary      Returns list of benchmarks
// @Description  In order to filter benchmarks by tags provide the tag key-value as query param
// @Tags     benchmarks
// @Accept   json
// @Produce  json
// @Param    provider  query     string  false  "Provider"  Enums(AWS,Azure)
// @Param        tags      query     string  false  "Tags in key-value query param"
// @Success  200       {object}  []api.Benchmark
// @Router       /inventory/api/v1/benchmarks [get]
func (h *HttpHandler) GetBenchmarks(ctx echo.Context) error {
	var provider *string
	tagFilters := make(map[string]string)
	for k, v := range ctx.QueryParams() {
		if k == "provider" {
			if len(v) == 1 {
				provider = &v[0]
			}
			continue
		}
		if len(v) == 1 {
			tagFilters[k] = v[0]
		}
	}
	benchmarks, err := h.db.ListBenchmarksWithFilters(provider, tagFilters)
	if err != nil {
		return err
	}

	var response []api.Benchmark
	for _, benchmark := range benchmarks {
		tags := make(map[string]string)
		for _, tag := range benchmark.Tags {
			tags[tag.Key] = tag.Value
		}
		response = append(response, api.Benchmark{
			ID:          benchmark.ID,
			Title:       benchmark.Title,
			Description: benchmark.Description,
			Provider:    api.SourceType(benchmark.Provider),
			State:       api.BenchmarkState(benchmark.State),
			Tags:        tags,
		})
	}

	return ctx.JSON(http.StatusOK, response)
}

// CountBenchmarks godoc
// @Summary      Returns count of benchmarks
// @Description  In order to filter benchmarks by tags provide the tag key-value as query param
// @Tags     benchmarks
// @Accept   json
// @Produce  json
// @Param        provider  query     string  false  "Provider"  Enums(AWS,Azure)
// @Param        tags      query     string  false  "Tags in key-value query param"
// @Success      200       {object}  []api.Benchmark
// @Router       /inventory/api/v1/benchmarks/count [get]
func (h *HttpHandler) CountBenchmarks(ctx echo.Context) error {
	var provider *string
	tagFilters := make(map[string]string)
	for k, v := range ctx.QueryParams() {
		if k == "provider" {
			if len(v) == 1 {
				provider = &v[0]
			}
			continue
		}
		if len(v) == 1 {
			tagFilters[k] = v[0]
		}
	}
	benchmarks, err := h.db.CountBenchmarksWithFilters(provider, tagFilters)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, benchmarks)
}

// CountPolicies godoc
// @Summary  Returns count of policies
// @Tags     benchmarks
// @Accept   json
// @Produce  json
// @Param        provider  query     string  false  "Provider"  Enums(AWS,Azure)
// @Success      200       {object}  []api.Benchmark
// @Router   /inventory/api/v1/policies/count [get]
func (h *HttpHandler) CountPolicies(ctx echo.Context) error {
	c, err := h.db.CountPolicies(ctx.QueryParam("provider"))
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, c)
}

// GetBenchmarkTags godoc
// @Summary  Returns list of benchmark tags
// @Tags     benchmarks
// @Accept   json
// @Produce  json
// @Success  200  {object}  []api.GetBenchmarkTag
// @Router   /inventory/api/v1/benchmarks/tags [get]
func (h *HttpHandler) GetBenchmarkTags(ctx echo.Context) error {
	tags, err := h.db.ListBenchmarkTags()
	if err != nil {
		return err
	}

	var response []api.GetBenchmarkTag
	for _, tag := range tags {
		response = append(response, api.GetBenchmarkTag{
			Key:   tag.Key,
			Value: tag.Value,
			Count: len(tag.Benchmarks),
		})
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetBenchmarkDetails godoc
// @Summary  Returns details of a given benchmark
// @Tags     benchmarks
// @Accept   json
// @Produce  json
// @Param    benchmarkId  path      int  true  "BenchmarkID"
// @Success  200          {object}  api.GetBenchmarkDetailsResponse
// @Router   /inventory/api/v1/benchmarks/{benchmarkId} [get]
func (h *HttpHandler) GetBenchmarkDetails(ctx echo.Context) error {
	benchmarkId := ctx.Param("benchmarkId")

	benchmark, err := h.db.GetBenchmark(benchmarkId)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "benchmark not found")
	}

	resp := api.GetBenchmarkDetailsResponse{}

	categories := make(map[string]string)
	subcategories := make(map[string]string)
	sections := make(map[string]string)
	for _, policy := range benchmark.Policies {
		categories[policy.Category] = ""
		subcategories[policy.SubCategory] = ""
		sections[policy.Section] = ""
	}

	for k := range categories {
		resp.Categories = append(resp.Categories, k)
	}
	for k := range subcategories {
		resp.Subcategories = append(resp.Subcategories, k)
	}
	for k := range sections {
		resp.Sections = append(resp.Sections, k)
	}

	return ctx.JSON(http.StatusOK, resp)
}

// GetPolicies godoc
// @Summary  Returns list of policies of a given benchmark
// @Tags         benchmarks
// @Accept       json
// @Produce      json
// @Param    benchmarkId  path      int     true   "BenchmarkID"
// @Param    category     query     string  false  "Category Filter"
// @Param    subcategory  query     string  false  "Subcategory Filter"
// @Param    section      query     string  false  "Section Filter"
// @Success  200          {object}  []api.Policy
// @Router   /inventory/api/v1/benchmarks/{benchmarkId}/policies [get]
func (h *HttpHandler) GetPolicies(ctx echo.Context) error {
	benchmarkId := ctx.Param("benchmarkId")

	var category, subcategory, section *string
	if len(ctx.QueryParam("category")) > 0 {
		temp := ctx.QueryParam("category")
		category = &temp
	}
	if len(ctx.QueryParam("subcategory")) > 0 {
		temp := ctx.QueryParam("subcategory")
		subcategory = &temp
	}
	if len(ctx.QueryParam("section")) > 0 {
		temp := ctx.QueryParam("section")
		section = &temp
	}

	policies, err := h.db.GetPoliciesWithFilters(benchmarkId, category, subcategory, section, nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "benchmark not found")
	}

	var resp []api.Policy
	for _, policy := range policies {
		tags := make(map[string]string)
		for _, tag := range policy.Tags {
			tags[tag.Key] = tag.Value
		}
		resp = append(resp, api.Policy{
			ID:                    policy.ID,
			Title:                 policy.Title,
			Description:           policy.Description,
			Category:              policy.Category,
			Subcategory:           policy.SubCategory,
			Section:               policy.Section,
			Severity:              policy.Severity,
			Provider:              policy.Provider,
			ManualVerification:    policy.ManualVerification,
			ManualRemedation:      policy.ManualRemedation,
			CommandLineRemedation: policy.CommandLineRemedation,
			QueryToRun:            policy.QueryToRun,
			Tags:                  nil,
		})
	}

	return ctx.JSON(http.StatusOK, resp)
}

// GetBenchmarkResult godoc
// @Summary      Returns summary of benchmark result
// @Description  Returns summary of benchmark, category, subcategory or section's result
// @Tags     benchmarks
// @Accept   json
// @Produce  json
// @Param        benchmarkId  query     string  false  "ID of Benchmark/Category/Subcategory/Section"
// @Param        sourceId     query     string  false  "SourceID"
// @Success      200          {object}  compliance_report.ReportGroupObj
// @Router       /inventory/api/v1/benchmarks/{benchmarkId}/{sourceId}/result [get]
func (h *HttpHandler) GetBenchmarkResult(ctx echo.Context) error {
	benchmarkID := ctx.Param("benchmarkId")

	sourceUUID, err := uuid.Parse(ctx.Param("sourceId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
	}

	var jobIDs []int
	jobs, err := api.ListComplianceReportJobs(h.schedulerBaseUrl, sourceUUID, nil)
	if err != nil {
		return err
	}
	for _, report := range jobs {
		jobIDs = append(jobIDs, int(report.ID))
	}

	// Since benchmark ID must be unique the result must be one record.
	// Keeping size to 2 and returning error on length != 1 if there's a mistake
	query := compliance_report.QueryReports(sourceUUID, jobIDs,
		[]compliance_report.ReportType{compliance_report.ReportTypeBenchmark},
		&benchmarkID, nil, 2, nil)
	b, err := json.Marshal(query)
	if err != nil {
		return err
	}

	var response compliance_report.ReportQueryResponse
	err = h.client.Search(context.Background(), compliance_report.ComplianceReportIndex,
		string(b), &response)
	if err != nil {
		return err
	}

	if len(response.Hits.Hits) != 1 {
		return echo.NewHTTPError(http.StatusNotFound, "benchmark not found")
	}

	if response.Hits.Hits[0].Source.Group == nil {
		return errors.New("benchmark doesnt have group")
	}

	return ctx.JSON(http.StatusOK, *response.Hits.Hits[0].Source.Group)
}

// GetResultPolicies godoc
// @Summary      Returns policy results of specific benchmark
// @Description  Returns policy results of specific benchmark
// @Tags     benchmarks
// @Accept   json
// @Produce  json
// @Param        benchmarkId  query     string  false  "ID of Benchmark/Category/Subcategory/Section"
// @Param        sourceId     query     string  false  "SourceID"
// @Param        category     query     string  false  "Category Filter"
// @Param        subcategory  query     string  false  "Subcategory Filter"
// @Param        section      query     string  false  "Section Filter"
// @Param        severity     query     string  false  "Severity Filter"
// @Param        status       query     string  false  "Status Filter"  Enums(passed,failed)
// @Success      200          {object}  []api.PolicyResult
// @Router       /inventory/api/v1/benchmarks/{benchmarkId}/{sourceId}/result/policies [get]
func (h *HttpHandler) GetResultPolicies(ctx echo.Context) error {
	benchmarkID := ctx.Param("benchmarkId")

	sourceUUID, err := uuid.Parse(ctx.Param("sourceId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
	}

	var status, severity, category, subcategory, section *string
	if len(ctx.QueryParam("status")) > 0 {
		temp := ctx.QueryParam("status")
		status = &temp
	}
	if len(ctx.QueryParam("severity")) > 0 {
		temp := ctx.QueryParam("severity")
		severity = &temp
	}
	if len(ctx.QueryParam("category")) > 0 {
		temp := ctx.QueryParam("category")
		category = &temp
	}
	if len(ctx.QueryParam("subcategory")) > 0 {
		temp := ctx.QueryParam("subcategory")
		subcategory = &temp
	}
	if len(ctx.QueryParam("section")) > 0 {
		temp := ctx.QueryParam("section")
		section = &temp
	}

	var jobIDs []int
	jobs, err := api.ListComplianceReportJobs(h.schedulerBaseUrl, sourceUUID, nil)
	if err != nil {
		return err
	}

	for _, report := range jobs {
		jobIDs = append(jobIDs, int(report.ID))
	}

	policies, err := h.db.GetPoliciesWithFilters(benchmarkID, category, subcategory,
		section, severity)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "benchmark not found")
	}

	var resp []*api.PolicyResult
	for _, policy := range policies {
		resp = append(resp, &api.PolicyResult{
			ID:          policy.ID,
			Title:       policy.Title,
			Category:    policy.Category,
			Subcategory: policy.SubCategory,
			Section:     policy.Section,
			Severity:    policy.Severity,
			Provider:    policy.Provider,
			Status:      api.PolicyResultStatusPassed,
			CreatedAt:   policy.CreatedAt.UnixMilli(),
		})
	}

	var searchAfter []interface{}
	for {
		query := compliance_report.QueryReports(sourceUUID, jobIDs,
			[]compliance_report.ReportType{compliance_report.ReportTypeResult},
			nil, &benchmarkID, EsFetchPageSize, searchAfter)
		b, err := json.Marshal(query)
		if err != nil {
			return err
		}

		var response compliance_report.ReportQueryResponse
		err = h.client.Search(context.Background(), compliance_report.ComplianceReportIndex,
			string(b), &response)
		if err != nil {
			return err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			if hit.Source.Result != nil {
				res := *hit.Source.Result
				for _, r := range resp {
					if r.ID == res.ControlId {
						r.TotalResources++
						r.DescribedAt = hit.Source.DescribedAt

						switch res.Result.Status {
						case compliance_report.ResultStatusOK:

							r.CompliantResources++
						case compliance_report.ResultStatusAlarm,
							compliance_report.ResultStatusError,
							compliance_report.ResultStatusSkip,
							compliance_report.ResultStatusInfo:

							r.Status = api.PolicyResultStatusFailed
						}
					}
				}
			}
			searchAfter = hit.Sort
		}
	}

	if status != nil {
		var temp []*api.PolicyResult
		for _, res := range resp {
			if *status == string(res.Status) {
				temp = append(temp, res)
			}
		}
		resp = temp
	}

	return ctx.JSON(http.StatusOK, resp)
}

// GetResource godoc
// @Summary      Get details of a Resource
// @Description  Getting resource details by id and resource type
// @Tags         resource
// @Accepts      json
// @Produce      json
// @Param        request  body  api.GetResourceRequest  true  "Request Body"
// @Router       /inventory/api/v1/resource [post]
func (h *HttpHandler) GetResource(ectx echo.Context) error {
	ctx := ectx.(*Context)
	cc := extractContext(ctx)

	req := &api.GetResourceRequest{}
	if err := ctx.BindValidate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request")
	}

	hash := sha256.New()
	hash.Write([]byte(req.ID))

	index := strings.ToLower(req.ResourceType)
	index = strings.ReplaceAll(index, "::", "_")
	index = strings.ReplaceAll(index, ".", "_")
	index = strings.ReplaceAll(index, "/", "_")
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"_id": fmt.Sprintf("%x", hash.Sum(nil)),
			},
		},
	}
	queryBytes, err := json.Marshal(query)
	if err != nil {
		return err
	}

	var response api.GenericQueryResponse
	err = h.client.Search(cc, index, string(queryBytes), &response)
	if err != nil {
		return err
	}

	var source map[string]interface{}
	for _, hit := range response.Hits.Hits {
		source = hit.Source
	}

	var cells map[string]*proto.Column
	pluginProvider := steampipe.ExtractPlugin(req.ResourceType)
	pluginTableName := steampipe.ExtractTableName(req.ResourceType)
	if pluginProvider == steampipe.SteampipePluginAWS {
		desc, err := api.ConvertToDescription(req.ResourceType, source)
		if err != nil {
			return err
		}

		cells, err = steampipe.AWSDescriptionToRecord(desc, pluginTableName)
		if err != nil {
			return err
		}
	} else if pluginProvider == steampipe.SteampipePluginAzure || pluginProvider == steampipe.SteampipePluginAzureAD {
		desc, err := api.ConvertToDescription(req.ResourceType, source)
		if err != nil {
			return err
		}

		if pluginProvider == steampipe.SteampipePluginAzure {
			cells, err = steampipe.AzureDescriptionToRecord(desc, pluginTableName)
			if err != nil {
				return err
			}
		} else {
			cells, err = steampipe.AzureADDescriptionToRecord(desc, pluginTableName)
			if err != nil {
				return err
			}
		}
	} else {
		return errors.New("invalid provider")
	}

	resp := map[string]string{}
	for k, v := range cells {
		val := v.GetStringValue()
		if len(val) > 0 {
			resp[k] = v.GetStringValue()
		}
	}

	return ctx.JSON(200, resp)
}

// ListQueries godoc
// @Summary      List smart queries
// @Description  Listing smart queries
// @Tags         smart_query
// @Produce      json
// @Param        request  body      api.ListQueryRequest  true  "Request Body"
// @Success      200      {object}  []api.SmartQueryItem
// @Router       /inventory/api/v1/query [get]
func (h *HttpHandler) ListQueries(ectx echo.Context) error {
	ctx := ectx.(*Context)

	req := &api.ListQueryRequest{}
	if err := ctx.BindValidate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var search *string
	if len(req.TitleFilter) > 0 {
		search = &req.TitleFilter
	}

	queries, err := h.db.GetQueriesWithFilters(search, req.Labels, req.ProviderFilter)
	if err != nil {
		return err
	}

	var result []api.SmartQueryItem
	for _, item := range queries {
		tags := map[string]string{}
		for _, tag := range item.Tags {
			tags[tag.Key] = tag.Value
		}
		result = append(result, api.SmartQueryItem{
			ID:          item.Model.ID,
			Provider:    item.Provider,
			Title:       item.Title,
			Description: item.Description,
			Query:       item.Query,
			Tags:        tags,
		})
	}
	return ctx.JSON(200, result)
}

// CountQueries godoc
// @Summary      Count smart queries
// @Description  Counting smart queries
// @Tags         smart_query
// @Produce      json
// @Param        request  body      api.ListQueryRequest  true  "Request Body"
// @Success      200      {object}  int
// @Router       /inventory/api/v1/query/count [get]
func (h *HttpHandler) CountQueries(ectx echo.Context) error {
	ctx := ectx.(*Context)

	req := &api.ListQueryRequest{}
	if err := ctx.BindValidate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var search *string
	if len(req.TitleFilter) > 0 {
		search = &req.TitleFilter
	}

	c, err := h.db.CountQueriesWithFilters(search, req.Labels, req.ProviderFilter)
	if err != nil {
		return err
	}
	return ctx.JSON(200, *c)
}

// RunQuery godoc
// @Summary      Run a specific smart query
// @Description  Run a specific smart query.
// @Description  In order to get the results in CSV format, Accepts header must be filled with `text/csv` value.
// @Description  Note that csv output doesn't process pagination and returns first 5000 records.
// @Tags         smart_query
// @Accepts      json
// @Produce      json,text/csv
// @Param        queryId  path      string               true  "QueryID"
// @Param        request  body      api.RunQueryRequest  true  "Request Body"
// @Param        accept   header    string               true  "Accept header"  Enums(application/json,text/csv)
// @Success      200      {object}  api.RunQueryResponse
// @Router       /inventory/api/v1/query/{queryId} [post]
func (h *HttpHandler) RunQuery(ectx echo.Context) error {
	ctx := ectx.(*Context)

	req := &api.RunQueryRequest{}
	if err := ctx.BindValidate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	queryId := ctx.Param("queryId")

	if accepts := ectx.Request().Header.Get("accept"); accepts != "" {
		mediaType, _, err := mime.ParseMediaType(accepts)
		if err == nil && mediaType == "text/csv" {
			req.Page = pagination.Page{
				NextMarker: "",
				Size:       5000,
			}

			ectx.Response().Header().Set(echo.HeaderContentType, "text/csv")
			ectx.Response().WriteHeader(http.StatusOK)

			query, err := h.db.GetQuery(queryId)
			if err != nil {
				if err == pgx.ErrNoRows {
					return echo.NewHTTPError(http.StatusNotFound, "Query not found")
				}
				return err
			}

			resp, err := h.RunSmartQuery(query.Query, req)
			if err != nil {
				return err
			}

			err = Csv(resp.Headers, ctx.Response())
			if err != nil {
				return err
			}

			for _, row := range resp.Result {
				var cells []string
				for _, cell := range row {
					cells = append(cells, fmt.Sprint(cell))
				}
				err := Csv(cells, ctx.Response())
				if err != nil {
					return err
				}
			}

			ectx.Response().Flush()
			return nil
		}
	}

	query, err := h.db.GetQuery(queryId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "Query not found")
		}
		return err
	}
	resp, err := h.RunSmartQuery(query.Query, req)
	if err != nil {
		return err
	}
	return ctx.JSON(200, resp)
}

// GetLocations godoc
// @Summary      Get locations
// @Description  Getting locations by provider
// @Tags         location
// @Produce      json
// @Param        provider  path      string  true  "Provider"  Enums(aws,azure)
// @Success      200       {object}  []api.LocationByProviderResponse
// @Router       /inventory/api/v1/locations/{provider} [get]
func (h *HttpHandler) GetLocations(ctx echo.Context) error {
	cc := extractContext(ctx)
	provider := ctx.Param("provider")

	var locations []api.LocationByProviderResponse

	if provider == "aws" || provider == "all" {
		regions, err := h.client.NewEC2RegionPaginator(nil, nil)
		if err != nil {
			return err
		}

		for regions.HasNext() {
			regions, err := regions.NextPage(cc)
			if err != nil {
				return err
			}

			for _, region := range regions {
				locations = append(locations, api.LocationByProviderResponse{
					Name: *region.Description.Region.RegionName,
				})
			}
		}
	}

	if provider == "azure" || provider == "all" {
		locs, err := h.client.NewLocationPaginator(nil, nil)
		if err != nil {
			return err
		}

		for locs.HasNext() {
			locpage, err := locs.NextPage(cc)
			if err != nil {
				return err
			}

			for _, location := range locpage {
				locations = append(locations, api.LocationByProviderResponse{
					Name: *location.Description.Location.Name,
				})
			}
		}
	}

	return ctx.JSON(http.StatusOK, locations)
}

// GetAzureResources godoc
// @Summary      Get Azure resources
// @Description  Getting Azure resources by filters.
// @Description  In order to get the results in CSV format, Accepts header must be filled with `text/csv` value.
// @Description  Note that csv output doesn't process pagination and returns first 5000 records.
// @Tags         inventory
// @Accept       json
// @Produce      json,text/csv
// @Param        request  body      api.GetResourcesRequest  true  "Request Body"
// @Param        accept   header    string                   true  "Accept header"  Enums(application/json,text/csv)
// @Success      200      {object}  api.GetAzureResourceResponse
// @Router       /inventory/api/v1/resources/azure [post]
func (h *HttpHandler) GetAzureResources(ectx echo.Context) error {
	provider := api.SourceCloudAzure
	if accepts := ectx.Request().Header.Get("accept"); accepts != "" {
		mediaType, _, err := mime.ParseMediaType(accepts)
		if err == nil && mediaType == "text/csv" {
			return h.GetResourcesCSV(ectx, &provider)
		}
	}
	return h.GetResources(ectx, &provider)
}

// GetAWSResources godoc
// @Summary      Get AWS resources
// @Description  Getting AWS resources by filters.
// @Description  In order to get the results in CSV format, Accepts header must be filled with `text/csv` value.
// @Description  Note that csv output doesn't process pagination and returns first 5000 records.
// @Tags         inventory
// @Accept       json
// @Produce      json,text/csv
// @Param        request  body      api.GetResourcesRequest  true  "Request Body"
// @Param        accept   header    string                   true  "Accept header"  Enums(application/json,text/csv)
// @Success      200      {object}  api.GetAWSResourceResponse
// @Router       /inventory/api/v1/resources/aws [post]
func (h *HttpHandler) GetAWSResources(ectx echo.Context) error {
	provider := api.SourceCloudAWS
	if accepts := ectx.Request().Header.Get("accept"); accepts != "" {
		mediaType, _, err := mime.ParseMediaType(accepts)
		if err == nil && mediaType == "text/csv" {
			return h.GetResourcesCSV(ectx, &provider)
		}
	}
	return h.GetResources(ectx, &provider)
}

// GetAllResources godoc
// @Summary      Get resources
// @Description  Getting all cloud providers resources by filters.
// @Description  In order to get the results in CSV format, Accepts header must be filled with `text/csv` value.
// @Description  Note that csv output doesn't process pagination and returns first 5000 records.
// @Description  If sort by is empty, result will be sorted by the first column in ascending order.
// @Tags         inventory
// @Accept       json
// @Produce      json,text/csv
// @Param        request  body      api.GetResourcesRequest  true  "Request Body"
// @Param        accept   header    string                   true  "Accept header"  Enums(application/json,text/csv)
// @Success      200      {object}  api.GetResourcesResponse
// @Router       /inventory/api/v1/resources [post]
func (h *HttpHandler) GetAllResources(ectx echo.Context) error {
	if accepts := ectx.Request().Header.Get("accept"); accepts != "" {
		mediaType, _, err := mime.ParseMediaType(accepts)
		if err == nil && mediaType == "text/csv" {
			return h.GetResourcesCSV(ectx, nil)
		}
	}
	return h.GetResources(ectx, nil)
}

func (h *HttpHandler) RunSmartQuery(query string,
	req *api.RunQueryRequest) (*api.RunQueryResponse, error) {

	var err error
	var lastIdx int
	if req.Page.NextMarker != "" && len(req.Page.NextMarker) > 0 {
		lastIdx, err = pagination.MarkerToIdx(req.Page.NextMarker)
		if err != nil {
			return nil, err
		}
	} else {
		lastIdx = 0
	}

	if req.Sorts == nil || len(req.Sorts) == 0 {
		req.Sorts = []api.SmartQuerySortItem{
			{
				Field:     "1",
				Direction: api.DirectionAscending,
			},
		}
	}
	if len(req.Sorts) > 1 {
		return nil, errors.New("multiple sort items not supported")
	}

	res, err := h.steampipeConn.Query(query, lastIdx, req.Page.Size, req.Sorts[0].Field, req.Sorts[0].Direction)
	if err != nil {
		return nil, err
	}

	newPage, err := pagination.NextPage(req.Page)
	if err != nil {
		return nil, err
	}

	resp := api.RunQueryResponse{
		Page:    newPage,
		Headers: res.headers,
		Result:  res.data,
	}
	return &resp, nil
}

func (h *HttpHandler) GetResources(ectx echo.Context, provider *api.SourceType) error {
	var err error
	cc := ectx.(*Context)
	req := &api.GetResourcesRequest{}
	if err := cc.BindValidate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	ctx := extractContext(ectx)

	res, err := api.QueryResources(ctx, h.client, req, provider)
	if err != nil {
		return err
	}

	if provider == nil {
		return cc.JSON(http.StatusOK, api.GetResourcesResponse{
			Resources: res.AllResources,
			Page:      res.Page,
		})
	} else if *provider == api.SourceCloudAWS {
		return cc.JSON(http.StatusOK, api.GetAWSResourceResponse{
			Resources: res.AWSResources,
			Page:      res.Page,
		})
	} else if *provider == api.SourceCloudAzure {
		return cc.JSON(http.StatusOK, api.GetAzureResourceResponse{
			Resources: res.AzureResources,
			Page:      res.Page,
		})
	} else {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid provider")
	}
}

func Csv(record []string, w io.Writer) error {
	wr := csv.NewWriter(w)
	err := wr.Write(record)
	if err != nil {
		return err
	}
	wr.Flush()
	return nil
}

func (h *HttpHandler) GetResourcesCSV(ectx echo.Context, provider *api.SourceType) error {
	var err error
	cc := ectx.(*Context)
	ctx := extractContext(ectx)

	req := &api.GetResourcesRequest{}
	if err := cc.BindValidate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	req.Page = pagination.Page{
		NextMarker: "",
		Size:       5000,
	}

	ectx.Response().Header().Set(echo.HeaderContentType, "text/csv")
	ectx.Response().WriteHeader(http.StatusOK)

	res, err := api.QueryResources(ctx, h.client, req, provider)
	if err != nil {
		return err
	}

	if provider == nil {
		err := Csv(api.AllResource{}.ToCSVHeaders(), cc.Response())
		if err != nil {
			return err
		}

		for _, resource := range res.AllResources {
			err := Csv(resource.ToCSVRecord(), cc.Response())
			if err != nil {
				return err
			}
		}
	} else if *provider == api.SourceCloudAWS {
		err := Csv(api.AWSResource{}.ToCSVHeaders(), cc.Response())
		if err != nil {
			return err
		}

		for _, resource := range res.AWSResources {
			err := Csv(resource.ToCSVRecord(), cc.Response())
			if err != nil {
				return err
			}
		}
	} else if *provider == api.SourceCloudAzure {
		err := Csv(api.AzureResource{}.ToCSVHeaders(), cc.Response())
		if err != nil {
			return err
		}

		for _, resource := range res.AzureResources {
			err := Csv(resource.ToCSVRecord(), cc.Response())
			if err != nil {
				return err
			}
		}
	} else {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid provider")
	}
	cc.Response().Flush()
	return nil
}

// GetComplianceReports godoc
// @Summary      Returns list of compliance report groups
// @Description  Returns list of compliance report groups of specified job id (if not specified, last one will be returned)
// @Tags         compliance_report
// @Accept       json
// @Produce      json
// @Param        source_id  path      string                          true   "Source ID"
// @Param        report_id  path      string                          false  "Report Job ID"
// @Param        request    body      api.GetComplianceReportRequest  true   "Request Body"
// @Success      200        {object}  []compliance_report.Report
// @Router       /inventory/api/v1/reports/compliance/{source_id} [get]
// @Router       /inventory/api/v1/reports/compliance/{source_id}/{report_id} [get]
func (h *HttpHandler) GetComplianceReports(ctx echo.Context) error {
	cc := ctx.(*Context)

	sourceUUID, err := uuid.Parse(ctx.Param("sourceId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
	}

	req := &api.GetComplianceReportRequest{}
	if err := cc.BindValidate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	lastIdx, err := pagination.MarkerToIdx(req.Page.NextMarker)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid page")
	}

	nextPage, err := pagination.NextPage(req.Page)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid page")
	}

	var jobIDs []int
	jobIDStr := ctx.Param("reportId")
	if jobIDStr != "" {
		jobID, err := strconv.Atoi(jobIDStr)
		if err != nil {
			ctx.Logger().Errorf("parsing jobid: %v", err)
			return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
		}
		jobIDs = append(jobIDs, jobID)
	} else {
		reports, err := api.ListComplianceReportJobs(h.schedulerBaseUrl, sourceUUID, req.Filters.TimeRange)
		if err != nil {
			return err
		}

		for _, report := range reports {
			jobIDs = append(jobIDs, int(report.ID))
		}
	}

	query := compliance_report.QueryReportsFrom(sourceUUID, jobIDs,
		[]compliance_report.ReportType{req.ReportType},
		req.Filters.GroupID, nil, req.Page.Size, lastIdx)
	b, err := json.Marshal(query)
	if err != nil {
		return err
	}

	var response compliance_report.ReportQueryResponse
	err = h.client.Search(context.Background(), compliance_report.ComplianceReportIndex,
		string(b), &response)
	if err != nil {
		return err
	}

	var reports []compliance_report.Report
	for _, hits := range response.Hits.Hits {
		reports = append(reports, hits.Source)
	}

	resp := api.GetComplianceReportResponse{
		Reports: reports,
		Page:    nextPage,
	}

	return ctx.JSON(http.StatusOK, resp)
}

func (c *Context) BindValidate(i interface{}) error {
	if err := c.Bind(i); err != nil {
		return err
	}

	if err := c.Validate(i); err != nil {
		return err
	}

	return nil
}
