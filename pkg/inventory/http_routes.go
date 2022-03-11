package inventory

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-hclog"
	"github.com/turbot/steampipe-plugin-sdk/logging"
	"github.com/turbot/steampipe-plugin-sdk/plugin/context_key"
	"io"
	"log"
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

	v1.POST("/resources/csv", h.GetAllResourcesCSV)
	v1.POST("/resources/azure/csv", h.GetAzureResourcesCSV)
	v1.POST("/resources/aws/csv", h.GetAWSResourcesCSV)
}

// GetResource godoc
// @Summary      Get details of a Resource
// @Description  Getting resource details by id and resource type
// @Tags         resource
// @Accepts      json
// @Produce      json
// @Param        id            body      string  true  "Id"
// @Param        resourceType  body      string  true  "ResourceType"
// @Router       /inventory/resource [post]
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

// GetLocations godoc
// @Summary      Get locations
// @Description  Getting locations by provider
// @Tags         location
// @Produce      json
// @Param        provider   path      string  true  "Provider" Enums(aws,azure)
// @Success      200  {object}  []LocationByProviderResponse
// @Router       /inventory/locations/{provider} [get]
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
// @Description  Getting Azure resources by filters
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        filters   body      Filters  true  "Filters"
// @Param        page      body      Page     true  "Page"
// @Param        sort      body      Sort     true  "Sort"
// @Success      200  {object}  GetAzureResourceResponse
// @Router       /inventory/resources/azure [post]
func (h *HttpHandler) GetAzureResources(ectx echo.Context) error {
	provider := SourceCloudAzure
	return h.GetResources(ectx, &provider)
}

// GetAzureResourcesCSV godoc
// @Summary      Get Azure resources in csv file
// @Description  Getting Azure resources by filters in csv file
// @Tags         inventory
// @Accept       json
// @Produce      plain
// @Param        filters   body      Filters  true  "Filters"
// @Param        sort      body      Sort     true  "Sort"
// @Success      200
// @Router       /inventory/resources/azure/csv [post]
func (h *HttpHandler) GetAzureResourcesCSV(ectx echo.Context) error {
	provider := SourceCloudAzure
	return h.GetResourcesCSV(ectx, &provider)
}

// GetAWSResources godoc
// @Summary      Get AWS resources
// @Description  Getting AWS resources by filters
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        filters   body      Filters  true  "Filters"
// @Param        page      body      Page     true  "Page"
// @Param        sort      body      Sort     true  "Sort"
// @Success      200  {object}  GetAWSResourceResponse
// @Router       /inventory/resources/aws [post]
func (h *HttpHandler) GetAWSResources(ectx echo.Context) error {
	provider := SourceCloudAWS
	return h.GetResources(ectx, &provider)
}

// GetAWSResourcesCSV godoc
// @Summary      Get AWS resources in csv file
// @Description  Getting AWS resources by filters in csv file
// @Tags         inventory
// @Accept       json
// @Produce      plain
// @Param        filters   body      Filters  true  "Filters"
// @Param        sort      body      Sort     true  "Sort"
// @Success      200
// @Router       /inventory/resources/aws/csv [post]
func (h *HttpHandler) GetAWSResourcesCSV(ectx echo.Context) error {
	provider := SourceCloudAWS
	return h.GetResourcesCSV(ectx, &provider)
}

// GetAllResources godoc
// @Summary      Get resources
// @Description  Getting all cloud providers resources by filters
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        filters   body      Filters  true  "Filters"
// @Param        page      body      Page     true  "Page"
// @Param        sort      body      Sort     true  "Sort"
// @Success      200  {object}  GetResourcesResponse
// @Router       /inventory/resources [post]
func (h *HttpHandler) GetAllResources(ectx echo.Context) error {
	return h.GetResources(ectx, nil)
}

// GetAllResourcesCSV godoc
// @Summary      Get all resources in csv file
// @Description  Getting all resources by filters in csv file
// @Tags         inventory
// @Accept       json
// @Produce      plain
// @Param        filters   body      Filters  true  "Filters"
// @Param        sort      body      Sort     true  "Sort"
// @Success      200
// @Router       /inventory/resources/csv [post]
func (h *HttpHandler) GetAllResourcesCSV(ectx echo.Context) error {
	return h.GetResourcesCSV(ectx, nil)
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
	if req.Page.NextMarker != nil && len(req.Page.NextMarker) > 0 {
		lastIdx, err = strconv.Atoi(string(req.Page.NextMarker))
		if err != nil {
			return err
		}
	} else {
		lastIdx = 0
	}

	resources, err := QuerySummaryResources(ctx, h.client, req.Filters, provider, req.Page.Size, lastIdx, req.Sort)
	if err != nil {
		return err
	}

	page := Page{
		Size:       req.Page.Size,
		NextMarker: []byte(strconv.Itoa(lastIdx + req.Page.Size)),
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

	req := &GetResourcesRequestCSV{}
	if err := cc.BindValidate(req); err != nil {
		return cc.JSON(http.StatusBadRequest, err)
	}

	reqCSV := GetResourcesRequest{Filters: req.Filters, Sort: req.Sort, Page: Page{
		NextMarker: nil,
		Size:       1000,
	}}

	ectx.Response().Header().Set(echo.HeaderContentType, echo.MIMETextPlain)
	ectx.Response().WriteHeader(http.StatusOK)

	total := 0
	writeHeaders := true
	for {
		n, nextPage, err := h.GetResourcesCSVPage(ectx, &reqCSV, provider, writeHeaders)
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

		reqCSV.Page = nextPage
	}

	return nil
}

func (h *HttpHandler) GetResourcesCSVPage(ectx echo.Context, req *GetResourcesRequest, provider *SourceType, writeHeaders bool) (int, Page, error) {
	var err error

	ctx := extractContext(ectx)
	cc := ectx.(*Context)

	var lastIdx int
	if req.Page.NextMarker != nil && len(req.Page.NextMarker) > 0 {
		lastIdx, err = strconv.Atoi(string(req.Page.NextMarker))
		if err != nil {
			return 0, Page{}, err
		}
	} else {
		lastIdx = 0
	}
	page := Page{
		Size:       req.Page.Size,
		NextMarker: []byte(strconv.Itoa(lastIdx + req.Page.Size)),
	}

	resources, err := QuerySummaryResources(ctx, h.client, req.Filters, provider, req.Page.Size, lastIdx, req.Sort)
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
			fmt.Println(AzureResource{}.ToCSVHeaders())
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
