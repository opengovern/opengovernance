package inventory

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
	"github.com/turbot/steampipe-plugin-sdk/logging"
	"github.com/turbot/steampipe-plugin-sdk/plugin/context_key"
	compliance_report "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report"
	describeAPI "gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	pagination "gitlab.com/keibiengine/keibi-engine/pkg/internal/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
	"io"
	"log"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
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
		return ctx.JSON(http.StatusBadRequest, err)
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
		return ctx.JSON(http.StatusInternalServerError, err)
	}

	var response api.GenericQueryResponse
	err = h.client.Search(cc, index, string(queryBytes), &response)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, err)
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
		return ctx.JSON(http.StatusInternalServerError, err)
	}

	var result []api.SmartQueryItem
	for _, item := range queries {
		result = append(result, api.SmartQueryItem{
			ID:          item.ID,
			Provider:    item.Provider,
			Title:       item.Title,
			Description: item.Description,
			Query:       item.Query,
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
		return ctx.JSON(http.StatusBadRequest, err)
	}

	queryId := ctx.Param("queryId")
	queryUUID, err := uuid.Parse(queryId)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err)
	}

	if accepts := ectx.Request().Header.Get("accept"); accepts != "" {
		mediaType, _, err := mime.ParseMediaType(accepts)
		if err == nil && mediaType == "text/csv" {
			req.Page = api.Page{
				NextMarker: "",
				Size:       5000,
			}

			ectx.Response().Header().Set(echo.HeaderContentType, "text/csv")
			ectx.Response().WriteHeader(http.StatusOK)

			query, err := h.db.GetQuery(queryUUID)
			if err != nil {
				return ctx.JSON(http.StatusNotFound, err)
			}

			resp, err := h.RunSmartQuery(query.Query, req)
			if err != nil {
				return ctx.JSON(http.StatusInternalServerError, err)
			}

			err = Csv(resp.Headers, ctx.Response())
			if err != nil {
				return ctx.JSON(http.StatusInternalServerError, err)
			}

			for _, row := range resp.Result {
				var cells []string
				for _, cell := range row {
					cells = append(cells, fmt.Sprint(cell))
				}
				err := Csv(cells, ctx.Response())
				if err != nil {
					return ctx.JSON(http.StatusInternalServerError, err)
				}
			}

			ectx.Response().Flush()
			return nil
		}
	}

	query, err := h.db.GetQuery(queryUUID)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, err)
	}
	resp, err := h.RunSmartQuery(query.Query, req)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, err)
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
		lastIdx, err = MarkerToIdx(req.Page.NextMarker)
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

	newIdx := lastIdx + req.Page.Size
	newPage := api.Page{
		NextMarker: BuildMarker(newIdx),
		Size:       req.Page.Size,
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
		return cc.JSON(http.StatusBadRequest, err)
	}

	ctx := extractContext(ectx)

	var lastIdx int
	if req.Page.NextMarker != "" && len(req.Page.NextMarker) > 0 {
		lastIdx, err = MarkerToIdx(req.Page.NextMarker)
		if err != nil {
			return err
		}
	} else {
		lastIdx = 0
	}

	resources, err := QuerySummaryResources(ctx, h.client, req.Query, req.Filters, provider, req.Page.Size, lastIdx, req.Sorts)
	if err != nil {
		return err
	}

	page := api.Page{
		Size:       req.Page.Size,
		NextMarker: BuildMarker(lastIdx + req.Page.Size),
	}

	if provider != nil && *provider == api.SourceCloudAWS {
		var awsResources []api.AWSResource
		for _, resource := range resources {
			awsResources = append(awsResources, api.AWSResource{
				Name:         resource.Name,
				ResourceType: resource.ResourceType,
				ResourceID:   resource.ResourceID,
				Region:       resource.Location,
				AccountID:    resource.SourceID,
			})
		}
		return cc.JSON(http.StatusOK, api.GetAWSResourceResponse{
			Resources: awsResources,
			Page:      page,
		})
	}

	if provider != nil && *provider == api.SourceCloudAzure {
		var azureResources []api.AzureResource
		for _, resource := range resources {
			azureResources = append(azureResources, api.AzureResource{
				Name:           resource.Name,
				ResourceType:   resource.ResourceType,
				ResourceGroup:  resource.ResourceGroup,
				Location:       resource.Location,
				ResourceID:     resource.ResourceID,
				SubscriptionID: resource.SourceID,
			})
		}
		return cc.JSON(http.StatusOK, api.GetAzureResourceResponse{
			Resources: azureResources,
			Page:      page,
		})
	}

	var allResources []api.AllResource
	for _, resource := range resources {
		allResources = append(allResources, api.AllResource{
			Name:         resource.Name,
			Provider:     api.SourceType(resource.SourceType),
			ResourceType: resource.ResourceType,
			Location:     resource.Location,
			ResourceID:   resource.ResourceID,
			SourceID:     resource.SourceID,
		})
	}
	return cc.JSON(http.StatusOK, api.GetResourcesResponse{
		Resources: allResources,
		Page:      page,
	})
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
	cc := ectx.(*Context)

	req := &api.GetResourcesRequest{}
	if err := cc.BindValidate(req); err != nil {
		return cc.JSON(http.StatusBadRequest, err)
	}

	req.Page = api.Page{
		NextMarker: "",
		Size:       1000,
	}

	ectx.Response().Header().Set(echo.HeaderContentType, "text/csv")
	ectx.Response().WriteHeader(http.StatusOK)

	total := 0
	writeHeaders := true
	for {
		n, nextPage, err := h.GetResourcesCSVPage(ectx, req, provider, writeHeaders)
		if err != nil {
			return err
		}
		writeHeaders = false

		if n == 0 {
			break
		}

		ectx.Response().Flush()

		total = total + n
		if total >= 4999 {
			break
		}

		req.Page = nextPage
	}

	return nil
}

func (h *HttpHandler) GetResourcesCSVPage(ectx echo.Context, req *api.GetResourcesRequest, provider *api.SourceType, writeHeaders bool) (int, api.Page, error) {
	var err error

	ctx := extractContext(ectx)
	cc := ectx.(*Context)

	var lastIdx int
	if req.Page.NextMarker != "" && len(req.Page.NextMarker) > 0 {
		lastIdx, err = MarkerToIdx(req.Page.NextMarker)
		if err != nil {
			return 0, api.Page{}, err
		}
	} else {
		lastIdx = 0
	}
	page := api.Page{
		Size:       req.Page.Size,
		NextMarker: BuildMarker(lastIdx + req.Page.Size),
	}

	resources, err := QuerySummaryResources(ctx, h.client, req.Query, req.Filters, provider, req.Page.Size, lastIdx, req.Sorts)
	if err != nil {
		return 0, api.Page{}, err
	}

	if provider != nil && *provider == api.SourceCloudAWS {
		var awsResources []api.AWSResource
		if writeHeaders {
			err := Csv(api.AWSResource{}.ToCSVHeaders(), cc.Response())
			if err != nil {
				return 0, api.Page{}, err
			}
			writeHeaders = false
		}
		for _, resource := range resources {
			awsResource := api.AWSResource{
				Name:         resource.Name,
				ResourceType: resource.ResourceType,
				ResourceID:   resource.ResourceID,
				Region:       resource.Location,
				AccountID:    resource.SourceID,
			}
			awsResources = append(awsResources, awsResource)

			err := Csv(awsResource.ToCSVRecord(), cc.Response())
			if err != nil {
				return 0, api.Page{}, err
			}
		}
		return len(resources), page, nil
	}

	if provider != nil && *provider == api.SourceCloudAzure {
		var azureResources []api.AzureResource
		if writeHeaders {
			err := Csv(api.AzureResource{}.ToCSVHeaders(), cc.Response())
			if err != nil {
				return 0, api.Page{}, err
			}

			writeHeaders = false
		}
		for _, resource := range resources {
			azureResource := api.AzureResource{
				Name:           resource.Name,
				ResourceType:   resource.ResourceType,
				ResourceGroup:  resource.ResourceGroup,
				Location:       resource.Location,
				ResourceID:     resource.ResourceID,
				SubscriptionID: resource.SourceID,
			}
			azureResources = append(azureResources, azureResource)

			err := Csv(azureResource.ToCSVRecord(), cc.Response())
			if err != nil {
				return 0, api.Page{}, err
			}
		}
		return len(resources), page, nil
	}

	var allResources []api.AllResource
	for _, resource := range resources {
		if writeHeaders {
			err := Csv(api.AllResource{}.ToCSVHeaders(), cc.Response())
			if err != nil {
				return 0, api.Page{}, err
			}
			writeHeaders = false
		}
		allResource := api.AllResource{
			Name:         resource.Name,
			Provider:     api.SourceType(resource.SourceType),
			ResourceType: resource.ResourceType,
			Location:     resource.Location,
			ResourceID:   resource.ResourceID,
			SourceID:     resource.SourceID,
		}
		allResources = append(allResources, allResource)

		err := Csv(allResource.ToCSVRecord(), cc.Response())
		if err != nil {
			return 0, api.Page{}, err
		}
	}
	return len(resources), page, nil
}

// GetComplianceReports godoc
// @Summary      Returns list of compliance report groups
// @Description  Returns list of compliance report groups of specified job id (if not specified, last one will be returned)
// @Tags         compliance_report
// @Accept       json
// @Produce      json
// @Param        source_id		path	string							true	"Source ID"
// @Param        report_id		path	string							true	"Report Job ID"
// @Param        request		body	api.GetComplianceReportRequest	true	"Request Body"
// @Success      200  {object}  []compliance_report.Report
// @Router       /reports/compliance/{source_id} [get]
// @Router       /reports/compliance/{source_id}/{report_id} [get]
func (h *HttpHandler) GetComplianceReports(ctx echo.Context) error {
	sourceUUID, err := uuid.Parse(ctx.Param("sourceId"))
	if err != nil {
		ctx.Logger().Errorf("parsing uuid: %v", err)
		return ctx.JSON(http.StatusBadRequest, describeAPI.ErrorResponse{Message: "invalid source uuid"})
	}

	req := &api.GetComplianceReportRequest{}
	if err := ctx.Bind(req); err != nil {
		return ctx.JSON(http.StatusBadRequest, err)
	}

	nextPage, err := pagination.NextPage(req.Page)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err)
	}

	var jobIDs []int
	jobIDStr := ctx.Param("reportId")
	if jobIDStr != "" {
		jobID, err := strconv.Atoi(jobIDStr)
		if err != nil {
			ctx.Logger().Errorf("parsing jobid: %v", err)
			return ctx.JSON(http.StatusBadRequest, describeAPI.ErrorResponse{Message: "invalid job id"})
		}
		jobIDs = append(jobIDs, jobID)
	} else {
		reports, err := api.ListComplianceReportJobs(h.schedulerBaseUrl, sourceUUID, req.Filters.TimeRange)
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, describeAPI.ErrorResponse{Message: "failed to fetch reports"})
		}

		for _, report := range reports {
			jobIDs = append(jobIDs, int(report.ID))
		}
	}

	lastIdx, err := pagination.MarkerToIdx(req.Page.NextMarker)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err)
	}

	query := compliance_report.QueryReports(sourceUUID, jobIDs, req.ReportType,
		req.Filters.GroupID, req.Page.Size, lastIdx)
	b, err := json.Marshal(query)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err)
	}

	var response compliance_report.ReportQueryResponse
	err = h.client.Search(context.Background(), compliance_report.ComplianceReportIndex,
		string(b), &response)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err)
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
