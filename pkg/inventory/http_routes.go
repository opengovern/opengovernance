package inventory

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-hclog"
	"github.com/turbot/steampipe-plugin-sdk/logging"
	"github.com/turbot/steampipe-plugin-sdk/plugin/context_key"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
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
	v1.POST("/resource", h.GetResource)
	v1.GET("/locations/:provider", h.GetLocations)
	v1.POST("/resources", h.GetAllResources)
	v1.POST("/resources/azure", h.GetAzureResources)
	v1.POST("/resources/aws", h.GetAWSResources)
}

// GetResource godoc
// @Summary      Get details of a Resource
// @Description  Getting resource details by id and resource type
// @Tags         resource
// @Produce      json
// @Param        id         body      string  true  "Id"
// @Param        resourceType         body      string  true  "ResourceType"
// @Success      200  {object}
// @Router       /resource [post]
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
// @Router       /locations/{provider} [get]
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
// @Success      200  {object}  GetAzureResourceResponse
// @Router       /resources/azure [post]
func (h *HttpHandler) GetAzureResources(ectx echo.Context) error {
	cc := ectx.(*Context)
	req := &GetResourcesRequest{}
	if err := cc.BindValidate(req); err != nil {
		return cc.JSON(http.StatusBadRequest, err)
	}

	provider := SourceCloudAzure
	return h.GetResources(ectx, req, &provider)
}

// GetAWSResources godoc
// @Summary      Get AWS resources
// @Description  Getting AWS resources by filters
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        filters   body      Filters  true  "Filters"
// @Param        page      body      Page     true  "Page"
// @Success      200  {object}  GetAWSResourceResponse
// @Router       /resources/aws [post]
func (h *HttpHandler) GetAWSResources(ectx echo.Context) error {
	cc := ectx.(*Context)
	req := &GetResourcesRequest{}
	if err := cc.BindValidate(req); err != nil {
		return cc.JSON(http.StatusBadRequest, err)
	}

	provider := SourceCloudAWS
	return h.GetResources(ectx, req, &provider)
}

// GetAllResources godoc
// @Summary      Get resources
// @Description  Getting all cloud providers resources by filters
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        filters   body      Filters  true  "Filters"
// @Param        page      body      Page     true  "Page"
// @Success      200  {object}  GetResourceResponse
// @Router       /resources [post]
func (h *HttpHandler) GetAllResources(ectx echo.Context) error {
	cc := ectx.(*Context)
	req := &GetResourcesRequest{}
	if err := cc.BindValidate(req); err != nil {
		return cc.JSON(http.StatusBadRequest, err)
	}

	return h.GetResources(ectx, req, nil)
}

func (h *HttpHandler) GetResources(ectx echo.Context, req *GetResourcesRequest, provider *SourceType) error {
	var err error

	ctx := extractContext(ectx)
	cc := ectx.(*Context)

	var lastIdx int
	if req.Page.NextMarker != nil && len(req.Page.NextMarker) > 0 {
		lastIdx, err = strconv.Atoi(string(req.Page.NextMarker))
		if err != nil {
			return err
		}
	} else {
		lastIdx = 0
	}

	var terms []keibi.BoolFilter
	if !FilterIsEmpty(req.Filters.Location) {
		terms = append(terms, keibi.TermsFilter("location", req.Filters.Location))
	}

	if !FilterIsEmpty(req.Filters.ResourceType) {
		terms = append(terms, keibi.TermsFilter("resourceType", req.Filters.ResourceType))
	}

	if !FilterIsEmpty(req.Filters.KeibiSource) {
		terms = append(terms, keibi.TermsFilter("sourceID", req.Filters.KeibiSource))
	}

	if provider != nil {
		terms = append(terms, keibi.TermsFilter("provider", []string{string(*provider)}))
	}

	var queryStr string
	if len(terms) > 0 {
		query := BuildBoolFilter(terms)
		var shouldTerms []interface{}
		shouldTerms = append(shouldTerms, query)

		query = BuildQuery(shouldTerms, req.Page.Size, lastIdx)
		queryBytes, err := json.Marshal(query)
		if err != nil {
			return err
		}

		queryStr = string(queryBytes)
	}

	resources, err := GetResourcesPageFromES(h.client, ctx, "inventory_summary", queryStr)
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

func GetResourcesPageFromES(client keibi.Client, ctx context.Context, index string, query string) ([]describe.KafkaLookupResource, error) {
	var response SummaryQueryResponse
	err := client.Search(ctx, index, query, &response)
	if err != nil {
		return nil, err
	}

	var resources []describe.KafkaLookupResource
	for _, hits := range response.Hits.Hits {
		resources = append(resources, hits.Source)
	}

	return resources, nil
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
