package inventory

import (
	"context"
	"encoding/json"
	"github.com/hashicorp/go-hclog"
	"github.com/turbot/steampipe-plugin-sdk/logging"
	"github.com/turbot/steampipe-plugin-sdk/plugin/context_key"
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
	v1.GET("/locations/:provider", h.GetLocations)
	v1.POST("/resources", h.GetResources)
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

// GetResources godoc
// @Summary      Get resources
// @Description  Getting resources by filters
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        filters   body      Filters  true  "Filters"
// @Param        page      body      Page     true  "Page"
// @Success      200  {object}  GetResourceResponse
// @Router       /resources [post]
func (h *HttpHandler) GetResources(ectx echo.Context) error {
	var err error

	ctx := extractContext(ectx)
	cc := ectx.(*Context)
	req := &GetResourceRequest{}
	if err := cc.BindValidate(req); err != nil {
		return cc.JSON(http.StatusBadRequest, err)
	}

	var lastIdx int
	if req.Page.NextMarker != nil || len(req.Page.NextMarker) > 0 {
		lastIdx, err = strconv.Atoi(string(req.Page.NextMarker))
		if err != nil {
			return err
		}
	} else {
		lastIdx = 0
	}

	indexName := "_all"
	if FilterIsEmpty(req.Filters.Provider) ||
		(FilterContains(req.Filters.Provider, "aws") && FilterContains(req.Filters.Provider, "azure")) {
		// index is still _all
	} else if FilterContains(req.Filters.Provider, "aws") {
		indexName = "aws_*"
	} else if FilterContains(req.Filters.Provider, "azure") {
		indexName = "microsoft_*"
	}

	if !FilterIsEmpty(req.Filters.ResourceType) {
		var indexes []string
		for _, resourceType := range req.Filters.ResourceType {
			resourceType = strings.ToLower(resourceType)
			resourceType = strings.ReplaceAll(resourceType, "::", "_") // aws
			resourceType = strings.ReplaceAll(resourceType, ".", "_")  // azure
			resourceType = strings.ReplaceAll(resourceType, "/", "_")  // azure
			if indexName == "aws_*" && !strings.HasPrefix(resourceType, "aws_") {
				continue
			}
			if indexName == "microsoft_*" && !strings.HasPrefix(resourceType, "microsoft_") {
				continue
			}

			indexes = append(indexes, resourceType)
		}

		indexName = strings.Join(indexes, ",")
	}

	var awsTerms []keibi.BoolFilter
	var azureTerms []keibi.BoolFilter
	if !FilterIsEmpty(req.Filters.Location) {
		awsTerms = append(awsTerms, keibi.TermsFilter("metadata.region", req.Filters.Location))
		azureTerms = append(azureTerms, keibi.TermsFilter("metadata.location", req.Filters.Location))
	}

	if !FilterIsEmpty(req.Filters.KeibiSource) {
		awsTerms = append(awsTerms, keibi.TermsFilter("metadata.account_id", req.Filters.KeibiSource))
		azureTerms = append(azureTerms, keibi.TermsFilter("metadata.subscription", req.Filters.KeibiSource))
	}

	var queryStr string
	if len(azureTerms) > 0 || len(awsTerms) > 0 {
		azureQuery := BuildBoolFilter(azureTerms)
		awsQuery := BuildBoolFilter(awsTerms)
		var shouldTerms []interface{}
		if len(azureTerms) > 0 {
			shouldTerms = append(shouldTerms, azureQuery)
		}
		if len(awsTerms) > 0 {
			shouldTerms = append(shouldTerms, awsQuery)
		}

		query := BuildQuery(shouldTerms, req.Page.Size, lastIdx)
		queryBytes, err := json.Marshal(query)
		if err != nil {
			return err
		}

		queryStr = string(queryBytes)
	}

	resources, err := h.GetResourcesPageFromES(ctx, indexName, queryStr)
	if err != nil {
		return err
	}

	return cc.JSON(http.StatusOK, GetResourceResponse{
		Resources: resources,
		Page: Page{
			Size:       req.Page.Size,
			NextMarker: []byte(strconv.Itoa(lastIdx + req.Page.Size)),
		},
	})
}

type QueryResponse struct {
	Hits QueryHits `json:"hits"`
}
type QueryHits struct {
	Total keibi.SearchTotal `json:"total"`
	Hits  []QueryHit        `json:"hits"`
}
type QueryHit struct {
	ID      string        `json:"_id"`
	Score   float64       `json:"_score"`
	Index   string        `json:"_index"`
	Type    string        `json:"_type"`
	Version int64         `json:"_version,omitempty"`
	Source  Source        `json:"_source"`
	Sort    []interface{} `json:"sort"`
}
type Source struct {
	ID           string   `json:"id"`
	Metadata     Metadata `json:"metadata"`
	ResourceType string   `json:"resource_type"`
	SourceType   string   `json:"source_type"`
}
type Metadata struct {
	AccountID string `json:"account_id"`
	Region    string `json:"region"`

	SubscriptionID string `json:"subscription_id"`
	Location       string `json:"location"`
}

func (h *HttpHandler) GetResourcesPageFromES(ctx context.Context, index string, query string) ([]Resource, error) {
	var response QueryResponse
	err := h.client.Search(ctx, index, query, &response)
	if err != nil {
		return nil, err
	}

	var resources []Resource
	for _, hits := range response.Hits.Hits {
		resource := Resource{
			ID:           hits.Source.ID,
			ResourceType: hits.Source.ResourceType,
		}
		if strings.HasPrefix(strings.ToLower(hits.Index), "aws") {
			resource.Location = hits.Source.Metadata.Region
			resource.KeibiSourceID = hits.Source.Metadata.AccountID
		} else if strings.HasPrefix(strings.ToLower(hits.Index), "microsoft") {
			resource.Location = hits.Source.Metadata.Location
			resource.KeibiSourceID = hits.Source.Metadata.SubscriptionID
		}

		resources = append(resources, resource)
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
