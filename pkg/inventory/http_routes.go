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
	"strconv"
	"strings"

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

	v1.GET("/query", h.ListQueries)
	v1.POST("/query/:queryId", h.RunQuery)

	v1.GET("/reports/compliance/:sourceId", h.GetComplianceReports)
	v1.GET("/reports/compliance/:sourceId/:reportId", h.GetComplianceReports)

	v1.GET("/benchmarks", h.GetBenchmarks)
	v1.GET("/benchmarks/tags", h.GetBenchmarkTags)
	v1.GET("/benchmarks/:benchmarkId", h.GetBenchmarkDetails)
	v1.GET("/benchmarks/:benchmarkId/policies", h.GetPolicies)
}

// GetBenchmarks godoc
// @Summary      Returns list of benchmarks
// @Description  In order to filter benchmarks by tags provide the tag key-value as query param
// @Tags         benchmarks
// @Accept       json
// @Produce      json
// @Param        provider	query	string	false	"Provider"	Enums(AWS,Azure)
// @Param        tags		query	string	false	"Tags in key-value query param"
// @Success      200  {object}  []api.Benchmark
// @Router       /benchmarks [get]
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

// GetBenchmarkTags godoc
// @Summary      Returns list of benchmark tags
// @Tags         benchmarks
// @Accept       json
// @Produce      json
// @Success      200  {object}  []api.GetBenchmarkTag
// @Router       /benchmarks/tags [get]
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
// @Summary      Returns details of a given benchmark
// @Tags         benchmarks
// @Accept       json
// @Produce      json
// @Param        benchmarkId	path	int	true	"BenchmarkID"
// @Success      200  {object}  api.GetBenchmarkDetailsResponse
// @Router       /benchmarks/{benchmarkId} [get]
func (h *HttpHandler) GetBenchmarkDetails(ctx echo.Context) error {
	benchmarkId, err := strconv.Atoi(ctx.Param("benchmarkId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid benchmark id")
	}

	benchmark, err := h.db.GetBenchmark(uint(benchmarkId))
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
// @Summary      Returns list of policies of a given benchmark
// @Tags         benchmarks
// @Accept       json
// @Produce      json
// @Param        benchmarkId	path	int		true	"BenchmarkID"
// @Param        category		query	string	false	"Category Filter"
// @Param        subcategory	query	string	false	"Subcategory Filter"
// @Param        section		query	string	false	"Section Filter"
// @Success      200  {object}  []api.Policy
// @Router       /benchmarks/{benchmarkId}/policies [get]
func (h *HttpHandler) GetPolicies(ctx echo.Context) error {
	benchmarkId, err := strconv.Atoi(ctx.Param("benchmarkId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid benchmark id")
	}
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

	policies, err := h.db.GetPoliciesWithFilters(uint(benchmarkId), category, subcategory, section)
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

// GetResource godoc
// @Summary      Get details of a Resource
// @Description  Getting resource details by id and resource type
// @Tags         resource
// @Accepts      json
// @Produce      json
// @Param        request	body	api.GetResourceRequest	true	"Request Body"
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

	var resp interface{}
	if source != nil {
		resp = source["description"]
	}
	return ctx.JSON(200, resp)
}

// ListQueries godoc
// @Summary      List smart queries
// @Description  Listing smart queries
// @Tags         smart_query
// @Produce      json
// @Success      200  {object}  []api.SmartQueryItem
// @Router       /inventory/api/v1/query [get]
func (h *HttpHandler) ListQueries(ectx echo.Context) error {
	ctx := ectx.(*Context)

	queries, err := h.db.GetQueries()
	if err != nil {
		return err
	}

	var result []api.SmartQueryItem
	for _, item := range queries {
		var tags []string
		for _, tag := range item.Tags {
			tags = append(tags, tag.Value)
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

// RunQuery godoc
// @Summary      Run a specific smart query
// @Description  Run a specific smart query.
// @Description  In order to get the results in CSV format, Accepts header must be filled with `text/csv` value.
// @Description  Note that csv output doesn't process pagination and returns first 5000 records.
// @Tags         smart_query
// @Accepts      json
// @Produce      json,text/csv
// @Param        queryId	path	string				true	"QueryID"
// @Param        request	body	api.RunQueryRequest	true	"Request Body"
// @Param        accept		header	string				true	"Accept header"		Enums(application/json,text/csv)
// @Success      200  {object}  api.RunQueryResponse
// @Router       /inventory/api/v1/query/{queryId} [post]
func (h *HttpHandler) RunQuery(ectx echo.Context) error {
	ctx := ectx.(*Context)

	req := &api.RunQueryRequest{}
	if err := ctx.BindValidate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request")
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
// @Param        provider   path      string  true  "Provider" Enums(aws,azure)
// @Success      200  {object}  []api.LocationByProviderResponse
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
// @Param        request	body	api.GetResourcesRequest	true	"Request Body"
// @Param        accept		header	string					true	"Accept header"		Enums(application/json,text/csv)
// @Success      200  {object}  api.GetAzureResourceResponse
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
// @Param        request	body	api.GetResourcesRequest	true	"Request Body"
// @Param        accept		header	string					true	"Accept header"		Enums(application/json,text/csv)
// @Success      200  {object}  api.GetAWSResourceResponse
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
// @Param        request	body	api.GetResourcesRequest	true	"Request Body"
// @Param        accept		header	string					true	"Accept header"		Enums(application/json,text/csv)
// @Success      200  {object}  api.GetResourcesResponse
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
// @Param        source_id		path	string							true	"Source ID"
// @Param        report_id		path	string							false	"Report Job ID"
// @Param        request		body	api.GetComplianceReportRequest	true	"Request Body"
// @Success      200  {object}  []compliance_report.Report
// @Router       /reports/compliance/{source_id} [get]
// @Router       /reports/compliance/{source_id}/{report_id} [get]
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

	query := compliance_report.QueryReports(sourceUUID, jobIDs, req.ReportType,
		req.Filters.GroupID, req.Page.Size, lastIdx)
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
