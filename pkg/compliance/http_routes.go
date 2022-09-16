package compliance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/inventory"

	es2 "gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	"gorm.io/gorm"

	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/es"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	compliance_es "gitlab.com/keibiengine/keibi-engine/pkg/compliance/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
)

const EsFetchPageSize = 10000

var (
	ErrInternalServer = errors.New("internal server error")
)

func (h *HttpHandler) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")
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

func bindValidate(ctx echo.Context, i interface{}) error {
	if err := ctx.Bind(i); err != nil {
		return err
	}

	if err := ctx.Validate(i); err != nil {
		return err
	}

	return nil
}

// GetBenchmarksInTime godoc
// @Summary      Returns all benchmark existed at the specified time
// @Description  You should fetch the benchmark report times from /benchmarks/history/:year/:month/:day
// @Tags     benchmarks
// @Accept   json
// @Produce  json
// @Param        provider   path      string  true  "Provider"  Enums(AWS,Azure,All)
// @Param        createdAt  path      string  true   "CreatedAt"
// @Success      200        {object}  []api.Benchmark
// @Router       /inventory/api/v1/benchmarks/history/list/{provider}/{createdAt} [get]
func (h *HttpHandler) GetBenchmarksInTime(ctx echo.Context) error {
	provider, _ := source.ParseType(ctx.Param("provider"))
	tim := ctx.Param("createdAt")
	timeInt, err := strconv.ParseInt(tim, 10, 64)
	if err != nil {
		return err
	}

	uniqueBenchmarkIDs := map[string]api.Benchmark{}
	var searchAfter []interface{}
	for {
		query, err := QueryBenchmarks(provider, timeInt, 2, EsFetchPageSize, searchAfter)
		if err != nil {
			return err
		}

		var response ReportQueryResponse
		err = h.client.Search(context.Background(), ComplianceReportIndex, query, &response)
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
// @Accept   json
// @Produce      json
// @Param    benchmarkId  path      string  true  "BenchmarkID"
// @Success  200          {object}  SummaryStatus
// @Router   /inventory/api/v1/benchmarks/{benchmarkId}/result/summary [get]
func (h *HttpHandler) GetBenchmarkResultSummary(ctx echo.Context) error {
	benchmarkID := ctx.Param("benchmarkId")

	reportID, err := h.schedulerClient.GetLastComplianceReportID(httpclient.FromEchoContext(ctx))
	if err != nil {
		return err
	}

	resp := SummaryStatus{}
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
			case ResultStatusOK:
				resp.OK++
			case ResultStatusAlarm:
				resp.Alarm++
			case ResultStatusInfo:
				resp.Info++
			case ResultStatusSkip:
				resp.Skip++
			case ResultStatusError:
				resp.Error++
			}
			searchAfter = hit.Sort
		}
	}

	return ctx.JSON(http.StatusOK, resp)
}

// GetBenchmarkResultPolicies godoc
// @Summary  Returns policies of result of benchmark
// @Tags     benchmarks
// @Accept   json
// @Produce  json
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

	reportID, err := h.schedulerClient.GetLastComplianceReportID(httpclient.FromEchoContext(ctx))
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
					if hit.Source.Status != ResultStatusOK {
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

	reportID, err := h.schedulerClient.GetLastComplianceReportID(httpclient.FromEchoContext(ctx))
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
					if hit.Source.Status != ResultStatusOK {
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

	reportID, err := h.schedulerClient.GetLastComplianceReportID(httpclient.FromEchoContext(ctx))
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
			if hit.Source.Status == ResultStatusOK {
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

	reportID, err := h.schedulerClient.GetLastComplianceReportID(httpclient.FromEchoContext(ctx))
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

		var response AccountReportQueryResponse
		err = h.client.Search(context.Background(), AccountReportIndex, query, &response)
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

		var response AccountReportQueryResponse
		err = h.client.Search(context.Background(), AccountReportIndex, query, &response)
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

	var hits []ReportQueryHit
	var searchAfter []interface{}
	for {
		query, err := QueryTrend(sourceUUID, benchmarkId, fromTime, toTime, EsFetchPageSize, searchAfter)
		if err != nil {
			return err
		}

		var response ReportQueryResponse
		err = h.client.Search(context.Background(), ComplianceReportIndex, query, &response)
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

	sortMap := []map[string]interface{}{
		{
			"described_at": "asc",
		},
	}
	sourceId := sourceUUID.String()
	rhits, err := es.FetchConnectionTrendSummaryPage(h.client, &sourceId, fromTime, toTime, sortMap, EsFetchPageSize)
	if err != nil {
		return err
	}

	var resp []api.ComplianceTrendDataPoint
	for _, hit := range hits {
		var total int64 = 0
		for _, rhit := range rhits {
			if rhit.DescribedAt == hit.Source.DescribedAt {
				total = int64(rhit.ResourceCount)
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

// GetCompliancyTrend godoc
// @Summary  Returns trend of compliancy for specific account
// @Tags     benchmarks
// @Accept   json
// @Produce  json
// @Param    sourceId  query     string  true  "SourceID"
// @Param    provider  query     string  true  "Provider"
// @Param    timeWindow  query     string  true  "Time Window"  Enums(24h,1w,3m,1y,max)
// @Success  200         {object}  []api.TrendDataPoint
// @Router   /inventory/api/v1/compliancy/trend [get]
func (h *HttpHandler) GetCompliancyTrend(ctx echo.Context) error {
	provider, _ := source.ParseType(ctx.QueryParam("provider"))
	sourceID := ctx.QueryParam("sourceId")
	timeWindow := ctx.QueryParam("timeWindow")

	if timeWindow == "" {
		timeWindow = "24h"
	}

	if provider == "" && sourceID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "you should specify either provider or sourceId")
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
	tw, err := inventory.ParseTimeWindow(ctx.QueryParam("timeWindow"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid timeWindow")
	}
	fromTime = time.Now().Add(-1 * tw).UnixMilli()

	var hits []es.ComplianceTrendQueryHit
	var searchAfter []interface{}
	for {
		query, err := es.FindCompliancyTrendQuery(sourceUUID, provider,
			fromTime, toTime, EsFetchPageSize, searchAfter)
		if err != nil {
			return err
		}

		var response es.ComplianceTrendQueryResponse
		err = h.client.Search(context.Background(), es2.SourceResourcesSummaryIndex, query, &response)
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

// GetBenchmarkAccountCompliance godoc
// @Summary  Returns no of compliant & non-compliant accounts
// @Tags     benchmarks
// @Accept   json
// @Produce  json
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
		query, err := QueryProviderResult(benchmarkId, tim, "asc", EsFetchPageSize, searchAfter)
		if err != nil {
			return err
		}

		var response AccountReportQueryResponse
		err = h.client.Search(context.Background(), AccountReportIndex, query, &response)
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

	query, err := QueryProviderResult(benchmarkId, tim, order, int32(size), nil)
	if err != nil {
		return err
	}

	var response AccountReportQueryResponse
	err = h.client.Search(context.Background(), AccountReportIndex, query, &response)
	if err != nil {
		return err
	}

	var reports []AccountReport
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
	var provider source.Type
	tagFilters := make(map[string]string)
	for k, v := range ctx.QueryParams() {
		if k == "provider" {
			if len(v) == 1 {
				provider, _ = source.ParseType(v[0])
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
// @Tags         benchmarks
// @Accept       json
// @Produce      json
// @Param        provider  query     string  false  "Provider"  Enums(AWS,Azure)
// @Param        tags      query     string  false  "Tags in key-value query param"
// @Success      200       {object}  []api.Benchmark
// @Router       /inventory/api/v1/benchmarks/count [get]
func (h *HttpHandler) CountBenchmarks(ctx echo.Context) error {
	var provider source.Type
	tagFilters := make(map[string]string)
	for k, v := range ctx.QueryParams() {
		if k == "provider" {
			if len(v) == 1 {
				provider, _ = source.ParseType(v[0])
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
	sourceType, _ := source.ParseType(ctx.QueryParam("provider"))
	c, err := h.db.CountPolicies(string(sourceType))
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

	source, err := h.schedulerClient.GetSource(httpclient.FromEchoContext(ctx), sourceUUID.String())
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
