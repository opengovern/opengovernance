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

	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	"gorm.io/gorm"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/es"

	"github.com/turbot/steampipe-plugin-sdk/grpc/proto"
	"gitlab.com/keibiengine/keibi-engine/pkg/steampipe"

	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
	"github.com/jackc/pgx/v4"
	"github.com/labstack/echo/v4"
	"github.com/turbot/steampipe-plugin-sdk/logging"
	"github.com/turbot/steampipe-plugin-sdk/plugin/context_key"
	compliance_report "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report"
	compliance_es "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report/es"
	pagination "gitlab.com/keibiengine/keibi-engine/pkg/internal/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/client"
)

const EsFetchPageSize = 10000

var (
	ErrInternalServer = errors.New("internal server error")
)

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
	v1.GET("/resources/categories", h.GetCategories)
	v1.GET("/accounts/resource/count", h.GetAccountsResourceCount)
	v1.GET("/services/distribution", h.GetServiceDistribution)

	v1.GET("/query", h.ListQueries)
	v1.GET("/query/count", h.CountQueries)
	v1.POST("/query/:queryId", h.RunQuery)

	//TODO- Deprecated
	/*
		v1.GET("/reports/compliance/:sourceId", h.GetComplianceReports)
		v1.GET("/reports/compliance/:sourceId/:reportId", h.GetComplianceReports)
		v1.GET("/benchmarks/:benchmarkId", h.GetBenchmarkDetails)
		v1.GET("/benchmarks/:benchmarkId/:sourceId/result", h.GetBenchmarkResult)
		v1.GET("/benchmarks/:benchmarkId/:sourceId/result/policies", h.GetResultPolicies)
	*/

	// benchmark details
	v1.GET("/benchmarks", h.GetBenchmarks)
	v1.GET("/benchmarks/tags", h.GetBenchmarkTags)
	v1.GET("/benchmarks/:benchmarkId/policies", h.GetPolicies)

	v1.GET("/benchmarks/:benchmarkId/result/summary", h.GetBenchmarkResultSummary)
	v1.GET("/benchmarks/:benchmarkId/result/policies", h.GetBenchmarkResultPolicies)
	v1.GET("/benchmarks/:benchmarkId/result/compliancy", h.GetBenchmarkResultCompliancy)
	v1.GET("/benchmarks/:benchmarkId/result/policies/:policyId/findings", h.GetBenchmarkResultPolicyFindings)
	v1.GET("/benchmarks/:benchmarkId/result/policies/:policyId/resources/summary", h.GetBenchmarkResultPolicyResourcesSummary)

	// benchmark dashboard
	v1.GET("/benchmarks/history/list/:provider/:createdAt", h.GetBenchmarksInTime)
	v1.GET("/benchmarks/:benchmarkId/:sourceId/compliance/trend", h.GetBenchmarkComplianceTrend)
	v1.GET("/benchmarks/:benchmarkId/:createdAt/accounts/compliance", h.GetBenchmarkAccountCompliance)
	v1.GET("/benchmarks/:benchmarkId/:createdAt/accounts", h.GetBenchmarkAccounts)

	// benchmark assignment
	v1.POST("/benchmarks/:benchmark_id/source/:source_id", h.CreateBenchmarkAssignment)
	v1.GET("/benchmarks/source/:source_id", h.GetAllBenchmarkAssignmentsBySourceId)
	v1.GET("/benchmarks/:benchmark_id/sources", h.GetAllBenchmarkAssignedSourcesByBenchmarkId)
	v1.DELETE("/benchmarks/:benchmark_id/source/:source_id", h.DeleteBenchmarkAssignment)

	// policy dashboard
	v1.GET("/benchmarks/compliancy/:provider/top/accounts", h.GetTopAccountsByBenchmarkCompliancy)
	v1.GET("/benchmarks/compliancy/:provider/top/services", h.GetTopServicesByBenchmarkCompliancy)
	v1.GET("/benchmarks/:provider/list", h.GetListOfBenchmarks)
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
// @Param        createdAt  path      string  true   "CreatedAt"
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

// GetBenchmarkResultSummary godoc
// @Summary  Returns summary of result of benchmark
// @Tags         benchmarks
// @Accept       json
// @Produce      json
// @Param    benchmarkId  path      string  true  "BenchmarkID"
// @Success  200          {object}  compliance_report.SummaryStatus
// @Router   /inventory/api/v1/benchmarks/{benchmarkId}/result/summary [get]
func (h *HttpHandler) GetBenchmarkResultSummary(ctx echo.Context) error {
	benchmarkID := ctx.Param("benchmarkId")

	reportID, err := client.GetLastComplianceReportID(h.schedulerBaseUrl)
	if err != nil {
		return err
	}

	resp := compliance_report.SummaryStatus{}
	var searchAfter []interface{}
	for {
		query, err := compliance_es.FindingsByBenchmarkID(benchmarkID, nil, reportID, EsFetchPageSize, searchAfter)
		if err != nil {
			return err
		}

		var response compliance_es.FindingsQueryResponse
		err = h.client.Search(context.Background(), compliance_es.FindingsIndex, query, &response)
		if err != nil {
			return err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			switch hit.Source.Status {
			case compliance_report.ResultStatusOK:
				resp.OK++
			case compliance_report.ResultStatusAlarm:
				resp.Alarm++
			case compliance_report.ResultStatusInfo:
				resp.Info++
			case compliance_report.ResultStatusSkip:
				resp.Skip++
			case compliance_report.ResultStatusError:
				resp.Error++
			}
			searchAfter = hit.Sort
		}
	}

	return ctx.JSON(http.StatusOK, resp)
}

// GetBenchmarkResultPolicies godoc
// @Summary  Returns policies of result of benchmark
// @Tags         benchmarks
// @Accept       json
// @Produce      json
// @Param    benchmarkId  path      string  true   "BenchmarkID"
// @Param    category     query     string  false  "Category Filter"
// @Param    subcategory  query     string  false  "Subcategory Filter"
// @Param    section      query     string  false  "Section Filter"
// @Param    severity     query     string  false  "Severity Filter"
// @Param    status       query     string  false  "Status Filter"  Enums(passed,failed)
// @Success  200          {object}  []api.ResultPolicy
// @Router   /inventory/api/v1/benchmarks/{benchmarkId}/result/policies [get]
func (h *HttpHandler) GetBenchmarkResultPolicies(ctx echo.Context) error {
	benchmarkID := ctx.Param("benchmarkId")
	reportID, err := client.GetLastComplianceReportID(h.schedulerBaseUrl)
	if err != nil {
		return err
	}

	var status, severity, category, subcategory, section *string
	if v := ctx.QueryParam("status"); len(v) > 0 {
		status = &v
	}
	if v := ctx.QueryParam("severity"); len(v) > 0 {
		severity = &v
	}
	if v := ctx.QueryParam("category"); len(v) > 0 {
		category = &v
	}
	if v := ctx.QueryParam("subcategory"); len(v) > 0 {
		subcategory = &v
	}
	if v := ctx.QueryParam("section"); len(v) > 0 {
		section = &v
	}

	policies, err := h.db.GetPoliciesWithFilters(benchmarkID, category, subcategory,
		section, severity)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "benchmark not found")
	}

	var resp []api.ResultPolicy
	for _, policy := range policies {
		resp = append(resp, api.ResultPolicy{
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
		query, err := compliance_es.FindingsByBenchmarkID(benchmarkID, status, reportID, EsFetchPageSize, searchAfter)
		if err != nil {
			return err
		}

		var response compliance_es.FindingsQueryResponse
		err = h.client.Search(context.Background(), compliance_es.FindingsIndex, query, &response)
		if err != nil {
			return err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			for idx, r := range resp {
				if r.ID == hit.Source.ControlID {
					resp[idx].DescribedAt = hit.Source.DescribedAt
					if hit.Source.Status != compliance_report.ResultStatusOK {
						resp[idx].Status = api.PolicyResultStatusFailed
					}
				}
			}
			searchAfter = hit.Sort
		}
	}

	return ctx.JSON(http.StatusOK, resp)
}

// GetBenchmarkResultCompliancy godoc
// @Summary  Returns compliancy of policies in result of benchmark
// @Tags     benchmarks
// @Accept   json
// @Produce  json
// @Param    benchmarkId  path      string  true   "BenchmarkID"
// @Param    category     query     string  false  "Category Filter"
// @Param    subcategory  query     string  false  "Subcategory Filter"
// @Param    section      query     string  false  "Section Filter"
// @Param    severity     query     string  false  "Severity Filter"
// @Param    status       query     string  false  "Status Filter"  Enums(passed,failed)
// @Success  200          {object}  []api.ResultCompliancy
// @Router   /inventory/api/v1/benchmarks/{benchmarkId}/result/findings [get]
func (h *HttpHandler) GetBenchmarkResultCompliancy(ctx echo.Context) error {
	benchmarkID := ctx.Param("benchmarkId")
	reportID, err := client.GetLastComplianceReportID(h.schedulerBaseUrl)
	if err != nil {
		return err
	}

	var status, severity, category, subcategory, section *string
	if v := ctx.QueryParam("status"); len(v) > 0 {
		status = &v
	}
	if v := ctx.QueryParam("severity"); len(v) > 0 {
		severity = &v
	}
	if v := ctx.QueryParam("category"); len(v) > 0 {
		category = &v
	}
	if v := ctx.QueryParam("subcategory"); len(v) > 0 {
		subcategory = &v
	}
	if v := ctx.QueryParam("section"); len(v) > 0 {
		section = &v
	}

	policies, err := h.db.GetPoliciesWithFilters(benchmarkID, category, subcategory,
		section, severity)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "benchmark not found")
	}

	var resp []api.ResultCompliancy
	for _, policy := range policies {
		resp = append(resp, api.ResultCompliancy{
			ID:          policy.ID,
			Title:       policy.Title,
			Category:    policy.Category,
			Subcategory: policy.SubCategory,
			Section:     policy.Section,
			Severity:    policy.Severity,
			Provider:    policy.Provider,
			Status:      api.PolicyResultStatusPassed,
		})
	}

	var searchAfter []interface{}
	for {
		query, err := compliance_es.FindingsByBenchmarkID(benchmarkID, status, reportID, EsFetchPageSize, searchAfter)
		if err != nil {
			return err
		}

		var response compliance_es.FindingsQueryResponse
		err = h.client.Search(context.Background(), compliance_es.FindingsIndex, query, &response)
		if err != nil {
			return err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			for idx, r := range resp {
				if r.ID == hit.Source.ControlID {
					resp[idx].TotalResources++
					if hit.Source.Status != compliance_report.ResultStatusOK {
						resp[idx].Status = api.PolicyResultStatusFailed
						resp[idx].ResourcesWithIssue++
					}
				}
			}
			searchAfter = hit.Sort
		}
	}

	return ctx.JSON(http.StatusOK, resp)
}

// GetBenchmarkResultPolicyResourcesSummary godoc
// @Summary  Returns summary of resources of a policy in results of a benchmark
// @Tags     benchmarks
// @Accept   json
// @Produce  json
// @Success  200  {object}  api.ResultPolicyResourceSummary
// @Router   /inventory/api/v1/benchmarks/{benchmarkId}/result/policies/{policy_id}/resources/summary [get]
func (h *HttpHandler) GetBenchmarkResultPolicyResourcesSummary(ctx echo.Context) error {
	benchmarkID := ctx.Param("benchmarkId")
	policyID := ctx.Param("policyId")
	reportID, err := client.GetLastComplianceReportID(h.schedulerBaseUrl)
	if err != nil {
		return err
	}

	res := api.ResultPolicyResourceSummary{
		ResourcesByLocation: make(map[string]int),
	}
	var searchAfter []interface{}
	for {
		query, err := compliance_es.FindingsByPolicyID(benchmarkID, policyID, reportID, EsFetchPageSize, searchAfter)
		if err != nil {
			return err
		}

		var response compliance_es.FindingsQueryResponse
		err = h.client.Search(context.Background(), compliance_es.FindingsIndex, query, &response)
		if err != nil {
			return err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			if hit.Source.Status == compliance_report.ResultStatusOK {
				res.CompliantResourceCount++
			} else {
				res.NonCompliantResourceCount++
			}
			res.ResourcesByLocation[hit.Source.ResourceLocation]++

			searchAfter = hit.Sort
		}
	}

	return ctx.JSON(http.StatusOK, res)
}

// GetBenchmarkResultPolicyFindings godoc
// @Summary  Returns findings of a policy in results of a benchmark
// @Tags     benchmarks
// @Accept   json
// @Produce  json
// @Success  200  {object}  []compliance_es.Finding
// @Router   /inventory/api/v1/benchmarks/{benchmarkId}/result/policies/{policy_id}/findings [get]
func (h *HttpHandler) GetBenchmarkResultPolicyFindings(ctx echo.Context) error {
	benchmarkID := ctx.Param("benchmarkId")
	policyID := ctx.Param("policyId")
	reportID, err := client.GetLastComplianceReportID(h.schedulerBaseUrl)
	if err != nil {
		return err
	}

	var findings []compliance_es.Finding
	var searchAfter []interface{}
	for {
		query, err := compliance_es.FindingsByPolicyID(benchmarkID, policyID, reportID, EsFetchPageSize, searchAfter)
		if err != nil {
			return err
		}

		var response compliance_es.FindingsQueryResponse
		err = h.client.Search(context.Background(), compliance_es.FindingsIndex, query, &response)
		if err != nil {
			return err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			findings = append(findings, hit.Source)
			searchAfter = hit.Sort
		}
	}

	return ctx.JSON(http.StatusOK, findings)
}

// GetListOfBenchmarks godoc
// @Summary      Returns all benchmark existed at the specified time
// @Description  You should fetch the benchmark report times from /benchmarks/history/:year/:month/:day
// @Tags     benchmarks
// @Accept   json
// @Produce  json
// @Param        count      query     int     true   "count"
// @Param        sourceId   query     string  false  "SourceID"
// @Param        provider   path      string  true   "Provider"  Enums(AWS,Azure)
// @Param        createdAt  path      string  true  "CreatedAt"
// @Success      200        {object}  []api.BenchmarkScoreResponse
// @Router       /inventory/api/v1/benchmarks/{provider}/list [get]
func (h *HttpHandler) GetListOfBenchmarks(ctx echo.Context) error {
	provider, err := source.ParseType(ctx.Param("provider"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid provider")
	}
	count, err := strconv.Atoi(ctx.QueryParam("count"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid count")
	}

	sID := ctx.QueryParam("sourceId")
	var sourceID *string
	if sID != "" {
		sourceUUID, err := uuid.Parse(sID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
		}
		s := sourceUUID.String()
		sourceID = &s
	}

	var searchAfter []interface{}
	benchmarkScore := map[string]int{}
	for {
		query, err := compliance_es.ComplianceScoreByProviderQuery(provider, sourceID, EsFetchPageSize, "desc", searchAfter)
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
			nonCompliant := hit.Source.TotalResources - hit.Source.TotalCompliant
			benchmarkScore[hit.Source.BenchmarkID] += nonCompliant
			searchAfter = hit.Sort
		}
	}

	var res []api.BenchmarkScoreResponse
	for id, score := range benchmarkScore {
		res = append(res, api.BenchmarkScoreResponse{
			BenchmarkID:       id,
			NonCompliantCount: score,
		})
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].NonCompliantCount < res[j].NonCompliantCount
	})
	if len(res) > count {
		res = res[:count]
	}
	return ctx.JSON(http.StatusOK, res)
}

// GetTopAccountsByBenchmarkCompliancy godoc
// @Summary  Return top accounts by benchmark compliancy
// @Tags     provider_dashboard
// @Accept   json
// @Produce  json
// @Param    count     query     int     true  "Count"
// @Param    order     query     string  true  "Order"     Enums(asc,desc)
// @Param    provider  path      string  true  "Provider"  Enums(AWS,Azure)
// @Success  200       {object}  []api.AccountCompliancyResponse
// @Router   /benchmarks/compliancy/{provider}/top/accounts [get]
func (h *HttpHandler) GetTopAccountsByBenchmarkCompliancy(ctx echo.Context) error {
	provider, err := source.ParseType(ctx.Param("provider"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid provider")
	}
	count, err := strconv.Atoi(ctx.QueryParam("count"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid count")
	}
	order := strings.ToLower(ctx.QueryParam("order"))
	if order != "asc" && order != "desc" {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid order")
	}

	var searchAfter []interface{}
	accountTotal := map[uuid.UUID]int{}
	accountCompliant := map[uuid.UUID]int{}
	for {
		query, err := compliance_es.ComplianceScoreByProviderQuery(provider, nil, EsFetchPageSize, order, searchAfter)
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
			accountTotal[hit.Source.SourceID] += hit.Source.TotalResources
			accountCompliant[hit.Source.SourceID] += hit.Source.TotalCompliant

			searchAfter = hit.Sort
		}
	}
	var res []api.AccountCompliancyResponse
	for k, v := range accountTotal {
		res = append(res, api.AccountCompliancyResponse{
			SourceID:       k,
			TotalResources: v,
			TotalCompliant: accountCompliant[k],
		})
	}
	sort.Slice(res, func(i, j int) bool {
		return (float64(res[i].TotalCompliant) / float64(res[i].TotalResources)) <
			(float64(res[j].TotalCompliant) / float64(res[j].TotalResources))
	})

	if len(res) > count {
		res = res[:count]
	}
	return ctx.JSON(http.StatusOK, res)
}

// GetTopServicesByBenchmarkCompliancy godoc
// @Summary  Return top accounts by benchmark compliancy
// @Tags     provider_dashboard
// @Accept   json
// @Produce  json
// @Param    count     query     int     true  "Count"
// @Param    order     query     string  true  "Order"     Enums(asc,desc)
// @Param    provider  path      string  true  "Provider"  Enums(AWS,Azure)
// @Success  200       {object}  []api.ServiceCompliancyResponse
// @Router   /benchmarks/compliancy/{provider}/top/services [get]
func (h *HttpHandler) GetTopServicesByBenchmarkCompliancy(ctx echo.Context) error {
	provider, err := source.ParseType(ctx.Param("provider"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid provider")
	}
	count, err := strconv.Atoi(ctx.QueryParam("count"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid count")
	}
	order := strings.ToLower(ctx.QueryParam("order"))
	if order != "asc" && order != "desc" {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid order")
	}

	var searchAfter []interface{}
	serviceTotal := map[string]int{}
	serviceCompliant := map[string]int{}
	for {
		query, err := compliance_es.ServiceComplianceScoreByProviderQuery(provider, EsFetchPageSize, order, searchAfter)
		if err != nil {
			return err
		}

		var response compliance_es.ServiceCompliancySummaryQueryResponse
		err = h.client.Search(context.Background(), compliance_es.CompliancySummaryIndex, query, &response)
		if err != nil {
			return err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			serviceTotal[hit.Source.ServiceName] += hit.Source.TotalResources
			serviceCompliant[hit.Source.ServiceName] += hit.Source.TotalCompliant

			searchAfter = hit.Sort
		}
	}
	var res []api.ServiceCompliancyResponse
	for k, v := range serviceTotal {
		res = append(res, api.ServiceCompliancyResponse{
			ServiceName:    k,
			TotalResources: v,
			TotalCompliant: serviceCompliant[k],
		})
	}
	sort.Slice(res, func(i, j int) bool {
		return (float64(res[i].TotalCompliant) / float64(res[i].TotalResources)) <
			(float64(res[j].TotalCompliant) / float64(res[j].TotalResources))
	})

	if len(res) > count {
		res = res[:count]
	}
	return ctx.JSON(http.StatusOK, res)
}

// GetBenchmarkComplianceTrend godoc
// @Summary  Returns trend of a benchmark compliance for specific account
// @Tags     benchmarks
// @Accept   json
// @Produce  json
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
// @Param    sourceId  query     string  true  "SourceID"
// @Param    provider  query     string  true  "Provider"
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
	tw, err := ParseTimeWindow(ctx.QueryParam("timeWindow"))
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
// @Tags     benchmarks
// @Accept   json
// @Produce  json
// @Param    sourceId    query     string  true  "SourceID"
// @Param    provider    query     string  true  "Provider"
// @Param    timeWindow  query     string  true  "Time Window"  Enums(24h,1w,3m,1y,max)
// @Success  200         {object}  []api.TrendDataPoint
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
	tw, err := ParseTimeWindow(ctx.QueryParam("timeWindow"))
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
// @Param    count     query     int     true   "count"
// @Param    provider  query     string  true   "Provider"
// @Success  200       {object}  []api.TopAccountResponse
// @Router   /inventory/api/v1/resources/top/accounts [get]
func (h *HttpHandler) GetTopAccountsByResourceCount(ctx echo.Context) error {
	provider, _ := source.ParseType(ctx.QueryParam("provider"))
	count, err := strconv.Atoi(ctx.QueryParam("count"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid count")
	}

	query, err := es.FindTopAccountsQuery(string(provider), count)
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
// @Param    sourceId  query     string  false  "SourceID"
// @Success  200       {object}  []api.TopServicesResponse
// @Router   /inventory/api/v1/resources/top/services [get]
func (h *HttpHandler) GetTopServicesByResourceCount(ctx echo.Context) error {
	provider, _ := source.ParseType(ctx.QueryParam("provider"))
	count, err := strconv.Atoi(ctx.QueryParam("count"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid count")
	}

	var sourceID *string
	sID := ctx.QueryParam("sourceId")
	if sID != "" {
		sourceUUID, err := uuid.Parse(sID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid sourceID")
		}
		s := sourceUUID.String()
		sourceID = &s
	}

	query, err := es.FindTopServicesQuery(string(provider), sourceID, count)
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

// GetCategories godoc
// @Summary  Return resource categories and number of resources
// @Tags     inventory
// @Accept   json
// @Produce  json
// @Param    provider  query     string  true  "Provider"
// @Success  200       {object}  []api.TopAccountResponse
// @Router   /inventory/api/v1/resources/categories [get]
func (h *HttpHandler) GetCategories(ctx echo.Context) error {
	provider, _ := source.ParseType(ctx.QueryParam("provider"))

	query, err := es.GetCategoriesQuery(string(provider), EsFetchPageSize)
	if err != nil {
		return err
	}

	var response es.CategoriesQueryResponse
	err = h.client.Search(context.Background(), describe.SourceResourcesSummary, query, &response)
	if err != nil {
		return err
	}

	var res []api.CategoriesResponse
	for _, hit := range response.Hits.Hits {
		res = append(res, api.CategoriesResponse{
			CategoryName:  hit.Source.CategoryName,
			ResourceCount: hit.Source.ResourceCount,
		})
	}
	return ctx.JSON(http.StatusOK, res)
}

// GetAccountsResourceCount godoc
// @Summary  Returns resource count of accounts
// @Tags     benchmarks
// @Accept   json
// @Produce  json
// @Param    provider  query     string  true  "Provider"
// @Success  200       {object}  []api.TopAccountResponse
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

// GetServiceDistribution godoc
// @Summary  Returns distribution of services for specific account
// @Tags     benchmarks
// @Accept   json
// @Produce  json
// @Param    sourceId    query     string  true  "SourceID"
// @Param    provider    query     string  true  "Provider"
// @Success  200       {object}  []api.ServiceDistributionItem
// @Router   /inventory/api/v1/services/distribution [get]
func (h *HttpHandler) GetServiceDistribution(ctx echo.Context) error {
	sourceID := ctx.QueryParam("sourceId")
	sourceUUID, err := uuid.Parse(sourceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
	}

	var res []api.ServiceDistributionItem
	var searchAfter []interface{}
	for {
		query, err := es.FindSourceServiceDistributionQuery(sourceUUID, EsFetchPageSize, searchAfter)
		if err != nil {
			return err
		}

		var response es.ServiceDistributionQueryResponse
		err = h.client.Search(context.Background(), describe.SourceResourcesSummary, query, &response)
		if err != nil {
			return err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			res = append(res, api.ServiceDistributionItem{
				ServiceName:  hit.Source.ServiceName,
				Distribution: hit.Source.LocationDistribution,
			})
			searchAfter = hit.Sort
		}
	}
	return ctx.JSON(http.StatusOK, res)
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
	jobs, err := client.ListComplianceReportJobs(h.schedulerBaseUrl, sourceUUID, nil)
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
	jobs, err := client.ListComplianceReportJobs(h.schedulerBaseUrl, sourceUUID, nil)
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

	resp := map[string]interface{}{}
	for k, v := range cells {
		if k == "tags" {
			var respTags []interface{}
			if jsonBytes := v.GetJsonValue(); jsonBytes != nil {
				var tags map[string]interface{}
				err = json.Unmarshal(jsonBytes, &tags)
				if err != nil {
					return err
				}
				for tagKey, tagValue := range tags {
					respTags = append(respTags, map[string]interface{}{
						"key":   tagKey,
						"value": tagValue,
					})
				}
			}
			resp["tags"] = respTags
		}

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
		uniqueSourceIds := map[string]string{}
		for _, resource := range res.AllResources {
			uniqueSourceIds[resource.SourceID] = ""
		}
		for sourceId := range uniqueSourceIds {
			src, err := api.GetSource(h.schedulerBaseUrl, sourceId)
			if err != nil {
				return err
			}

			uniqueSourceIds[sourceId] = src.Name
		}
		for idx := range res.AllResources {
			res.AllResources[idx].SourceName = uniqueSourceIds[res.AllResources[idx].SourceID]
		}
		return cc.JSON(http.StatusOK, api.GetResourcesResponse{
			Resources: res.AllResources,
			Page:      res.Page,
			ResultCount: api.ResultCount{
				Value:    res.ResultCount.Value,
				Relation: api.Relation(res.ResultCount.Relation),
			},
		})
	} else if *provider == api.SourceCloudAWS {
		uniqueSourceIds := map[string]string{}
		for _, resource := range res.AWSResources {
			uniqueSourceIds[resource.AccountID] = ""
		}
		for sourceId := range uniqueSourceIds {
			src, err := api.GetSource(h.schedulerBaseUrl, sourceId)
			if err != nil {
				return err
			}

			uniqueSourceIds[sourceId] = src.Name
		}
		for _, resource := range res.AWSResources {
			resource.AccountName = uniqueSourceIds[resource.AccountID]
		}
		return cc.JSON(http.StatusOK, api.GetAWSResourceResponse{
			Resources: res.AWSResources,
			Page:      res.Page,
			ResultCount: api.ResultCount{
				Value:    res.ResultCount.Value,
				Relation: api.Relation(res.ResultCount.Relation),
			},
		})
	} else if *provider == api.SourceCloudAzure {
		uniqueSourceIds := map[string]string{}
		for _, resource := range res.AzureResources {
			uniqueSourceIds[resource.SubscriptionID] = ""
		}
		for sourceId := range uniqueSourceIds {
			src, err := api.GetSource(h.schedulerBaseUrl, sourceId)
			if err != nil {
				return err
			}

			uniqueSourceIds[sourceId] = src.Name
		}
		for _, resource := range res.AzureResources {
			resource.SubscriptionName = uniqueSourceIds[resource.SubscriptionID]
		}
		return cc.JSON(http.StatusOK, api.GetAzureResourceResponse{
			Resources: res.AzureResources,
			Page:      res.Page,
			ResultCount: api.ResultCount{
				Value:    res.ResultCount.Value,
				Relation: api.Relation(res.ResultCount.Relation),
			},
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
		reports, err := client.ListComplianceReportJobs(h.schedulerBaseUrl, sourceUUID, req.Filters.TimeRange)
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

// CreateBenchmarkAssignment godoc
// @Summary      Create benchmark assignment for inventory service
// @Description  Returns benchmark assignment which insert
// @Tags         benchmarks_assignment
// @Accept       json
// @Produce      json
// @Param        benchmark_id  path      string  true  "Benchmark ID"
// @Param        source_id     path      string  true  "Source ID"
// @Success      200           {object}  api.BenchmarkAssignment
// @Router       /inventory/api/v1/benchmarks/{benchmark_id}/source/{source_id} [post]
func (h *HttpHandler) CreateBenchmarkAssignment(ctx echo.Context) error {
	sourceId := ctx.Param("source_id")
	if sourceId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "source id is empty")
	}
	sourceUUID, err := uuid.Parse(sourceId)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
	}

	benchmarkId := ctx.Param("benchmark_id")
	if benchmarkId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "benchmark id is empty")
	}
	benchmark, err := h.db.GetBenchmark(benchmarkId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("benchmark %s not found", benchmarkId))
		}
		ctx.Logger().Errorf("find benchmark assignment: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}

	source, err := client.GetSource(h.schedulerBaseUrl, sourceUUID)
	if err != nil {
		ctx.Logger().Errorf(fmt.Sprintf("request source: %v", err))
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}
	if benchmark.Provider != string(source.Type) {
		return echo.NewHTTPError(http.StatusBadRequest, "source type not match")
	}

	assignment := &BenchmarkAssignment{
		BenchmarkId: benchmarkId,
		SourceId:    sourceUUID,
		AssignedAt:  time.Now(),
	}
	if err := h.db.AddBenchmarkAssignment(assignment); err != nil {
		ctx.Logger().Errorf("add benchmark assignment: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}

	return ctx.JSON(http.StatusOK, api.BenchmarkAssignment{
		BenchmarkId: benchmarkId,
		SourceId:    sourceUUID.String(),
		AssignedAt:  assignment.AssignedAt.Unix(),
	})
}

// GetAllBenchmarkAssignmentsBySourceId godoc
// @Summary      Get all benchmark assignments with source id
// @Description  Returns all benchmark assignments with source id
// @Tags         benchmarks_assignment
// @Accept       json
// @Produce      json
// @Param        source_id  path      string  true  "Source ID"
// @Success      200        {object}  []api.BenchmarkAssignment
// @Router       /inventory/api/v1/benchmarks/source/{source_id} [get]
func (h *HttpHandler) GetAllBenchmarkAssignmentsBySourceId(ctx echo.Context) error {
	sourceId := ctx.Param("source_id")
	if sourceId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "source id is empty")
	}
	sourceUUID, err := uuid.Parse(sourceId)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
	}

	dbAssignments, err := h.db.GetBenchmarkAssignmentsBySourceId(sourceUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("benchmark assignments for %s not found", sourceId))
		}
		ctx.Logger().Errorf("find benchmark assignments by source %s: %v", sourceId, err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}

	assignments := []api.BenchmarkAssignment{}
	for _, assignment := range dbAssignments {
		assignments = append(assignments, api.BenchmarkAssignment{
			BenchmarkId: assignment.BenchmarkId,
			SourceId:    assignment.SourceId.String(),
			AssignedAt:  assignment.AssignedAt.Unix(),
		})
	}

	return ctx.JSON(http.StatusOK, assignments)
}

// GetAllBenchmarkAssignedSourcesByBenchmarkId godoc
// @Summary      Get all benchmark assigned sources with benchmark id
// @Description  Returns all benchmark assigned sources with benchmark id
// @Tags         benchmarks_assignment
// @Accept       json
// @Produce      json
// @Param        benchmark_id  path      string  true  "Benchmark ID"
// @Success      200           {object}  []api.BenchmarkAssignedSource
// @Router       /inventory/api/v1/benchmarks/{benchmark_id}/sources [get]
func (h *HttpHandler) GetAllBenchmarkAssignedSourcesByBenchmarkId(ctx echo.Context) error {
	benchmarkId := ctx.Param("benchmark_id")
	if benchmarkId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "benchmark id is empty")
	}

	dbAssignments, err := h.db.GetBenchmarkAssignmentsByBenchmarkId(benchmarkId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("benchmark assignments for %s not found", benchmarkId))
		}
		ctx.Logger().Errorf("find benchmark assignments by benchmark %s: %v", benchmarkId, err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}

	sources := []api.BenchmarkAssignedSource{}
	for _, assignment := range dbAssignments {
		sources = append(sources, api.BenchmarkAssignedSource{
			SourceId:   assignment.SourceId.String(),
			AssignedAt: assignment.AssignedAt.Unix(),
		})
	}

	return ctx.JSON(http.StatusOK, sources)
}

// DeleteBenchmarkAssignment godoc
// @Summary      Delete benchmark assignment for inventory service
// @Description  Delete benchmark assignment with source id and benchmark id
// @Tags         benchmarks_assignment
// @Accept       json
// @Produce      json
// @Param        benchmark_id  path  string  true  "Benchmark ID"
// @Param        source_id     path  string  true  "Source ID"
// @Success      200
// @Router       /inventory/api/v1/benchmarks/{benchmark_id}/source/{source_id} [delete]
func (h *HttpHandler) DeleteBenchmarkAssignment(ctx echo.Context) error {
	sourceId := ctx.Param("source_id")
	if sourceId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "source id is empty")
	}
	sourceUUID, err := uuid.Parse(sourceId)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
	}
	benchmarkId := ctx.Param("benchmark_id")
	if benchmarkId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "benchmark id is empty")
	}

	if _, err := h.db.GetBenchmarkAssignmentByIds(sourceUUID, benchmarkId); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusFound, "benchmark assignment not found")
		}
		ctx.Logger().Errorf("find benchmark assignment: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}

	if err := h.db.DeleteBenchmarkAssignmentById(sourceUUID, benchmarkId); err != nil {
		ctx.Logger().Errorf("delete benchmark assignment: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrInternalServer)
	}

	return ctx.JSON(http.StatusOK, nil)
}
