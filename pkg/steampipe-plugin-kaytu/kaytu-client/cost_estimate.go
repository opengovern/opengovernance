package kaytu_client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/steampipe-plugin-kaytu/kaytu-sdk/config"
	essdk "github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	steampipesdk "github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"github.com/kaytu-io/pennywise/pkg/cost"
	"github.com/kaytu-io/pennywise/pkg/schema"
	"github.com/kaytu-io/pennywise/pkg/submission"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"net/http"
	"runtime"
	"strings"
	"time"
)

type ResourceCostEstimate struct {
	ResourceID   string  `json:"resource_id"`
	ResourceType string  `json:"resource_type"`
	Cost         float64 `json:"cost"`
}

func ResourceTypeConversion(resourceType string) string {
	//TODO
	switch resourceType {
	case "aws::elasticloadbalancing::loadbalancer":
		return "aws_lb"
	case "aws::ec2::volumesnapshot":
		return "aws_ebs_snapshot"
	}
	return resourceType
}

func GetValues(resource Resource) (map[string]interface{}, error) {
	switch strings.ToLower(resource.ResourceType) {
	// AWS
	case "aws::elasticloadbalancing::loadbalancer":
		return getAwsLoadBalancerValues(resource)
	case "aws::elasticloadbalancingv2::loadbalancer":
		return getAwsLoadBalancer2Values(resource)
	case "aws::ec2::instance":
		return getAwsEc2InstanceValues(resource)
	case "aws::autoscaling::autoscalinggroup":
		return nil, nil
	case "aws::rds::dbinstance":
		return getAwsRdsDbInstanceValues(resource)
	case "aws::ec2::volume":
		return getAwsEbsVolumeValues(resource)
	case "aws::ec2::volumesnapshot":
		return getAwsEbsSnapshotValues(resource)
	case "aws::efs::filesystem":
		return getAwsEfsFileSystemValues(resource)
	case "aws::elasticache::cluster":
		return getAwsElastiCacheClusterValues(resource)
	case "aws::elasticache::replicationgroup":
		return getAwsElastiCacheReplicationGroupValues(resource)
	case "aws::ec2::eip":
		return getAwsEc2EipValues(resource)
	case "aws::eks::cluster":
		return getAwsEksClusterValues(resource)
	case "aws::eks::nodegroup":
		return getAwsEksNodeGroupValues(resource)
	case "aws::fsx::filesystem":
		return getAwsFSXFileSystemValues(resource)
	case "aws::ec2::natgateway":
		return getAwsNatGatewayValues(resource)
	case "aws::ec2::host":
		return getAwsEc2Values(resource)
	case "aws::lambda::function":
		return getAwsLambdaFunctionValues(resource)
	case "aws::elasticsearch::domain":
		return getAwsEsDomainValues(resource)
	case "aws::opensearch::domain":
		return getAwsOpenSearchDomainValues(resource)

	// Azure
	case "microsoft.compute/virtualmachines":
		return nil, nil
	case "microsoft.compute/disks":
		return nil, nil
	case "microsoft.compute/images":
		return nil, nil
	case "microsoft.compute/snapshots":
		return nil, nil
	case "microsoft.compute/virtualmachinescalesets":
		return nil, nil
	case "microsoft.network/loadbalancers":
		return nil, nil
	case "microsoft.network/loadbalancers/loadbalancingeules":
		return nil, nil
	case "microsoft.network/loadbalancers/outboundrules":
		return nil, nil
	case "microsoft.network/applicationgateways":
		return nil, nil
	case "microsoft.network/natgateways":
		return nil, nil
	case "microsoft.network/publicipaddresses":
		return nil, nil
	case "microsoft.network/publicipprefixes":
		return nil, nil
	case "microsoft.containerregistry/registries":
		return nil, nil
	case "microsoft.network/privateendpoints":
		return nil, nil
	case "microsoft.storage/queues":
		return nil, nil
	case "microsoft.storage/fileshares":
		return nil, nil
	case "microsoft.storage/storageaccounts":
		return nil, nil
	case "microsoft.network/virtualnetworkgateways":
		return nil, nil
	case "microsoft.keyvault/vaults/keys":
		return nil, nil
	case "microsoft.keyvault/managedhsms":
		return nil, nil
	case "microsoft.cdn/profiles/endpoints":
		return nil, nil
	case "microsoft.network/dnszones":
		return nil, nil
	case "microsoft.network/privatednszones":
		return nil, nil
	case "microsoft.documentdb/sqldatabases":
		return nil, nil
	case "microsoft.documentdb/mongodatabases":
		return nil, nil
	case "microsoft.documentdb/mongocollection":
		return nil, nil
	case "microsoft.dbformariadb/servers":
		return nil, nil
	case "microsoft.sql/servers/databases":
		return nil, nil
	case "microsoft.sql/managedInstances":
		return nil, nil
	case "microsoft.dbformysql/servers":
		return nil, nil
	case "microsoft.dbforpostgresql/servers":
		return nil, nil
	case "microsoft.dbforpostgresql/flexibleservers":
		return nil, nil
	case "microsoft.dbformysql/flexibleservers":
		return nil, nil
	case "microsoft.containerservice/managedclusters":
		return nil, nil
	case "microsoft.web/hostingenvironments":
		return nil, nil
	case "microsoft.web/plan":
		return nil, nil
	case "microsoft.apimanagement/service":
		return nil, nil
	case "microsoft.web/sites":
		return nil, nil
	case "microsoft.search/searchservices":
		return nil, nil
	case "microsoft.automation/automationaccounts":
		return nil, nil
	}
	return map[string]interface{}{}, nil
}

type LookupQueryResponse struct {
	Hits struct {
		Hits []struct {
			ID      string         `json:"_id"`
			Score   float64        `json:"_score"`
			Index   string         `json:"_index"`
			Type    string         `json:"_type"`
			Version int64          `json:"_version,omitempty"`
			Source  LookupResource `json:"_source"`
			Sort    []any          `json:"sort"`
		}
	}
}

func FetchLookupByResourceIDType(client Client, ctx context.Context, d *plugin.QueryData) (*LookupQueryResponse, error) {
	filters := essdk.BuildFilter(ctx, d.QueryContext, map[string]string{
		"resource_id":   "resource_id",
		"resource_type": "resource_type",
	}, "", nil, nil, nil)
	out, err := json.Marshal(filters)
	if err != nil {
		return nil, err
	}

	var filterMap []map[string]any
	err = json.Unmarshal(out, &filterMap)
	if err != nil {
		return nil, err
	}

	request := make(map[string]any)
	request["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filterMap,
		},
	}

	b, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	plugin.Logger(ctx).Error("ListResourceCostEstimate Query", "query=", string(b), "index=", InventorySummaryIndex)

	var response LookupQueryResponse
	err = client.ES.Search(context.Background(), InventorySummaryIndex, string(b), &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func ListResourceCostEstimate(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (any, error) {
	plugin.Logger(ctx).Warn("ListResourceCostEstimate", d)
	runtime.GC()
	// create service
	cfg := config.GetConfig(d.Connection)

	plugin.Logger(ctx).Trace("ListResourceCostEstimate 2", cfg)
	ke, err := config.NewClientCached(cfg, d.ConnectionCache, ctx)
	if err != nil {
		return nil, err
	}
	k := Client{ES: ke}

	plugin.Logger(ctx).Trace("ListResourceCostEstimate 3", k)
	sc, err := steampipesdk.NewSelfClientCached(ctx, d.ConnectionCache)
	if err != nil {
		plugin.Logger(ctx).Error("ListResourceCostEstimate NewSelfClientCached", "error", err)
		return nil, err
	}
	plugin.Logger(ctx).Trace("ListResourceCostEstimate 4", sc)
	encodedResourceCollectionFilters, err := sc.GetConfigTableValueOrNil(ctx, steampipesdk.KaytuConfigKeyResourceCollectionFilters)
	if err != nil {
		plugin.Logger(ctx).Error("ListResourceCostEstimate GetConfigTableValueOrNil for resource_collection_filters", "error", err)
		return nil, err
	}
	plugin.Logger(ctx).Trace("ListResourceCostEstimate 5", encodedResourceCollectionFilters)
	clientType, err := sc.GetConfigTableValueOrNil(ctx, steampipesdk.KaytuConfigKeyClientType)
	if err != nil {
		plugin.Logger(ctx).Error("ListResourceCostEstimate GetConfigTableValueOrNil for client_type", "error", err)
		return nil, err
	}

	plugin.Logger(ctx).Trace("Columns", d.EqualsQuals)
	var indexes []string
	for column, q := range d.EqualsQuals {
		if column == "resource_type" {
			if s, ok := q.GetValue().(*proto.QualValue_StringValue); ok && s != nil {
				indexes = []string{ResourceTypeToESIndex(s.StringValue)}
			} else if l := q.GetListValue(); l != nil {
				for _, v := range l.GetValues() {
					if v == nil {
						continue
					}
					indexes = append(indexes, v.GetStringValue())
				}
			}
		}
	}

	req := submission.Submission{
		ID:        "submittion-1",
		CreatedAt: time.Now(),
		Resources: []schema.ResourceDef{},
	}

	var resources []Resource

	for _, index := range indexes {
		paginator, err := k.NewResourcePaginator(essdk.BuildFilterWithDefaultFieldName(ctx, d.QueryContext, resourceMapping,
			"", nil, encodedResourceCollectionFilters, clientType, true), d.QueryContext.Limit, index)
		if err != nil {
			plugin.Logger(ctx).Error("ListResourceCostEstimate NewResourcePaginator", "error", err)
			return nil, err
		}

		for paginator.HasNext() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				plugin.Logger(ctx).Error("ListResourceCostEstimate NextPage", "error", err)
				return nil, err
			}
			plugin.Logger(ctx).Trace("ListResourceCostEstimate", "next page")

			for _, hit := range page {
				resources = append(resources, hit)

				var provider schema.ProviderName
				if hit.SourceType == source.CloudAWS.String() {
					provider = schema.AWSProvider
				} else if hit.SourceType == source.CloudAzure.String() {
					provider = schema.AzureProvider
				}
				values, err := GetValues(hit)
				if err != nil {
					plugin.Logger(ctx).Error("GetValues ", "error", err)
					return nil, err
				}
				req.Resources = append(req.Resources, schema.ResourceDef{
					Address:      hit.ID,
					Type:         ResourceTypeConversion(hit.ResourceType),
					Name:         hit.Metadata.Name,
					RegionCode:   hit.Metadata.Region,
					ProviderName: provider,
					Values:       values,
				})
			}
		}
		err = paginator.Close(ctx)
		if err != nil {
			return nil, err
		}
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	plugin.Logger(ctx).Warn("ListResourceCostEstimate: Pennywise")

	var response cost.State
	statusCode, err := httpclient.DoRequest("GET", *cfg.PennywiseBaseURL+"/api/v1/cost/submission", nil, reqBody, &response)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get pennywise cost, status code = %d", statusCode)
	}

	for _, hit := range resources {
		resourceCost, err := response.Cost()
		if err != nil {
			return nil, err
		}

		d.StreamListItem(ctx, ResourceCostEstimate{
			ResourceID:   hit.ID,
			ResourceType: hit.ResourceType,
			Cost:         resourceCost.Decimal.InexactFloat64(),
		})
	}

	plugin.Logger(ctx).Warn("ListResourceCostEstimate: Done", fmt.Sprintf("%v", response.Resources))
	return nil, nil
}
