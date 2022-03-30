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
	"io"
	"log"
	"mime"
	"net/http"
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
}

// GetResource godoc
// @Summary      Get details of a Resource
// @Description  Getting resource details by id and resource type
// @Tags         resource
// @Accepts      json
// @Produce      json
// @Param        request	body	GetResourceRequest	true	"Request Body"
// @Router       /inventory/api/v1/resource [post]
func (h *HttpHandler) GetResource(ectx echo.Context) error {
	ctx := ectx.(*Context)
	cc := extractContext(ctx)

	req := &GetResourceRequest{}
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

	var response GenericQueryResponse
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
// @Success      200  {object}  []SmartQueryItem
// @Router       /inventory/api/v1/query [get]
func (h *HttpHandler) ListQueries(ectx echo.Context) error {
	ctx := ectx.(*Context)

	queries, err := h.db.GetQueries()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, err)
	}

	var result []SmartQueryItem
	for _, item := range queries {
		result = append(result, SmartQueryItem{
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
// @Param        queryId	path	string			true	"QueryID"
// @Param        request	body	RunQueryRequest	true	"Request Body"
// @Param        accept		header	string				true	"Accept header"		Enums(application/json,text/csv)
// @Success      200  {object}  RunQueryResponse
// @Router       /inventory/api/v1/query/{queryId} [post]
func (h *HttpHandler) RunQuery(ectx echo.Context) error {
	ctx := ectx.(*Context)

	req := &RunQueryRequest{}
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
			req.Page = Page{
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
// @Success      200  {object}  []LocationByProviderResponse
// @Router       /inventory/api/v1/locations/{provider} [get]
func (h *HttpHandler) GetLocations(ctx echo.Context) error {
	cc := extractContext(ctx)
	provider := ctx.Param("provider")

	var locations []LocationByProviderResponse

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
				locations = append(locations, LocationByProviderResponse{
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
				locations = append(locations, LocationByProviderResponse{
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
// @Param        request	body	GetResourcesRequest	true	"Request Body"
// @Param        accept		header	string				true	"Accept header"		Enums(application/json,text/csv)
// @Success      200  {object}  GetAzureResourceResponse
// @Router       /inventory/api/v1/resources/azure [post]
func (h *HttpHandler) GetAzureResources(ectx echo.Context) error {
	provider := SourceCloudAzure
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
// @Param        request	body	GetResourcesRequest	true	"Request Body"
// @Param        accept		header	string				true	"Accept header"		Enums(application/json,text/csv)
// @Success      200  {object}  GetAWSResourceResponse
// @Router       /inventory/api/v1/resources/aws [post]
func (h *HttpHandler) GetAWSResources(ectx echo.Context) error {
	provider := SourceCloudAWS
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
// @Param        request	body	GetResourcesRequest	true	"Request Body"
// @Param        accept		header	string				true	"Accept header"		Enums(application/json,text/csv)
// @Success      200  {object}  GetResourcesResponse
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
	req *RunQueryRequest) (*RunQueryResponse, error) {

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
		req.Sorts = []SmartQuerySortItem{
			{
				Field:     "1",
				Direction: DirectionAscending,
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
	newPage := Page{
		NextMarker: BuildMarker(newIdx),
		Size:       req.Page.Size,
	}
	resp := RunQueryResponse{
		Page:    newPage,
		Headers: res.headers,
		Result:  res.data,
	}
	return &resp, nil
}

func (h *HttpHandler) GetResources(ectx echo.Context, provider *SourceType) error {
	var err error
	cc := ectx.(*Context)
	req := &GetResourcesRequest{}
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

	page := Page{
		Size:       req.Page.Size,
		NextMarker: BuildMarker(lastIdx + req.Page.Size),
	}

	if provider != nil && *provider == SourceCloudAWS {
		var awsResources []AWSResource
		for _, resource := range resources {
			awsResources = append(awsResources, AWSResource{
				Name:         resource.Name,
				ResourceType: resource.ResourceType,
				ResourceID:   resource.ResourceID,
				Region:       resource.Location,
				AccountID:    resource.SourceID,
			})
		}
		return cc.JSON(http.StatusOK, GetAWSResourceResponse{
			Resources: awsResources,
			Page:      page,
		})
	}

	if provider != nil && *provider == SourceCloudAzure {
		var azureResources []AzureResource
		for _, resource := range resources {
			azureResources = append(azureResources, AzureResource{
				Name:           resource.Name,
				ResourceType:   resource.ResourceType,
				ResourceGroup:  resource.ResourceGroup,
				Location:       resource.Location,
				ResourceID:     resource.ResourceID,
				SubscriptionID: resource.SourceID,
			})
		}
		return cc.JSON(http.StatusOK, GetAzureResourceResponse{
			Resources: azureResources,
			Page:      page,
		})
	}

	var allResources []AllResource
	for _, resource := range resources {
		allResources = append(allResources, AllResource{
			Name:         resource.Name,
			Provider:     SourceType(resource.SourceType),
			ResourceType: resource.ResourceType,
			Location:     resource.Location,
			ResourceID:   resource.ResourceID,
			SourceID:     resource.SourceID,
		})
	}
	return cc.JSON(http.StatusOK, GetResourcesResponse{
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

func (h *HttpHandler) GetResourcesCSV(ectx echo.Context, provider *SourceType) error {
	cc := ectx.(*Context)

	req := &GetResourcesRequest{}
	if err := cc.BindValidate(req); err != nil {
		return cc.JSON(http.StatusBadRequest, err)
	}

	req.Page = Page{
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

func (h *HttpHandler) GetResourcesCSVPage(ectx echo.Context, req *GetResourcesRequest, provider *SourceType, writeHeaders bool) (int, Page, error) {
	var err error

	ctx := extractContext(ectx)
	cc := ectx.(*Context)

	var lastIdx int
	if req.Page.NextMarker != "" && len(req.Page.NextMarker) > 0 {
		lastIdx, err = MarkerToIdx(req.Page.NextMarker)
		if err != nil {
			return 0, Page{}, err
		}
	} else {
		lastIdx = 0
	}
	page := Page{
		Size:       req.Page.Size,
		NextMarker: BuildMarker(lastIdx + req.Page.Size),
	}

	resources, err := QuerySummaryResources(ctx, h.client, req.Query, req.Filters, provider, req.Page.Size, lastIdx, req.Sorts)
	if err != nil {
		return 0, Page{}, err
	}

	if provider != nil && *provider == SourceCloudAWS {
		var awsResources []AWSResource
		if writeHeaders {
			err := Csv(AWSResource{}.ToCSVHeaders(), cc.Response())
			if err != nil {
				return 0, Page{}, err
			}
			writeHeaders = false
		}
		for _, resource := range resources {
			awsResource := AWSResource{
				Name:         resource.Name,
				ResourceType: resource.ResourceType,
				ResourceID:   resource.ResourceID,
				Region:       resource.Location,
				AccountID:    resource.SourceID,
			}
			awsResources = append(awsResources, awsResource)

			err := Csv(awsResource.ToCSVRecord(), cc.Response())
			if err != nil {
				return 0, Page{}, err
			}
		}
		return len(resources), page, nil
	}

	if provider != nil && *provider == SourceCloudAzure {
		var azureResources []AzureResource
		if writeHeaders {
			err := Csv(AzureResource{}.ToCSVHeaders(), cc.Response())
			if err != nil {
				return 0, Page{}, err
			}

			writeHeaders = false
		}
		for _, resource := range resources {
			azureResource := AzureResource{
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
				return 0, Page{}, err
			}
		}
		return len(resources), page, nil
	}

	var allResources []AllResource
	for _, resource := range resources {
		if writeHeaders {
			err := Csv(AllResource{}.ToCSVHeaders(), cc.Response())
			if err != nil {
				return 0, Page{}, err
			}
			writeHeaders = false
		}
		allResource := AllResource{
			Name:         resource.Name,
			Provider:     SourceType(resource.SourceType),
			ResourceType: resource.ResourceType,
			Location:     resource.Location,
			ResourceID:   resource.ResourceID,
			SourceID:     resource.SourceID,
		}
		allResources = append(allResources, allResource)

		err := Csv(allResource.ToCSVRecord(), cc.Response())
		if err != nil {
			return 0, Page{}, err
		}
	}
	return len(resources), page, nil
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
