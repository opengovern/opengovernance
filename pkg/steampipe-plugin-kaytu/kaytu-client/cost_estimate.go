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
	case "aws::elasticloadbalancingv2::loadbalancer":
		return "aws_alb"
	case "aws::ec2::instance":
		return "aws_instance"
	case "aws::autoscaling::autoscalinggroup":
		return "aws_autoscaling_group"
	case "aws::rds::dbinstance":
		return "aws_db_instance"
	case "aws::ec2::volume":
		return "aws_ebs_volume"
	case "aws::ec2::volumegp3":
		return "aws_ebs_volume"
	case "aws::efs::filesystem":
		return "aws_efs_file_system"
	case "aws::elasticache::cluster":
		return "aws_elasticache_cluster"
	case "aws::elasticache::replicationgroup":
		return "aws_elasticache_replication_group"
	case "aws::ec2::eip":
		return "aws_eip"
	case "aws::eks::cluster":
		return "aws_eks_cluster"
	case "aws::eks::nodegroup":
		return "aws_eks_node_group"
	case "aws::fsx::filesystem":
		return "aws_fsx_ontap_file_system"
	case "aws::ec2::natgateway":
		return "aws_nat_gateway"
	case "aws::ec2::host":
		return "aws_ec2_host"
	case "aws::lambda::function":
		return "aws_lambda_function"
	case "aws::elasticsearch::domain":
		return "aws_elasticsearch_domain"
	case "aws::opensearch::domain":
		return "aws_opensearch_domain"
	case "aws::dynamodb::table":
		return "aws_dynamodb_table"

	// Azure
	case "microsoft.compute/virtualmachines":
		return "azurerm_virtual_machine"
	case "microsoft.compute/disks":
		return "azurerm_managed_disk"
	case "microsoft.compute/images":
		return "azurerm_image"
	case "microsoft.compute/snapshots":
		return "azurerm_snapshot"
	case "microsoft.compute/virtualmachinescalesets":
		return "azurerm_virtual_machine_scale_set"
	case "microsoft.network/loadbalancers":
		return "azurerm_lb"
	case "microsoft.network/loadbalancers/loadbalancingeules":
		return "azurerm_lb_rule"
	case "microsoft.network/loadbalancers/outboundrules":
		return "azurerm_lb_outbound_rule"
	case "microsoft.network/applicationgateways":
		return "azurerm_application_gateway"
	case "microsoft.network/natgateways":
		return "azurerm_nat_gateway"
	case "microsoft.network/publicipaddresses":
		return "azurerm_public_ip"
	case "microsoft.network/publicipprefixes":
		return "azurerm_public_ip_prefix"
	case "microsoft.containerregistry/registries":
		return "azurerm_container_registry"
	case "microsoft.network/privateendpoints":
		return "azurerm_private_endpoint"
	case "microsoft.storage/queues":
		return "azurerm_storage_queue"
	case "microsoft.storage/fileshares":
		return "azurerm_storage_share"
	case "microsoft.storage/storageaccounts":
		return "azurerm_storage_account"
	case "microsoft.network/virtualnetworkgateways":
		return "azurerm_virtual_network_gateway"
	case "microsoft.keyvault/vaults/keys":
		return "azurerm_key_vault_key"
	case "microsoft.keyvault/managedhsms":
		return "azurerm_key_vault_managed_hardware_security_module"
	case "microsoft.cdn/profiles/endpoints":
		return "azurerm_cdn_endpoint"
	case "microsoft.network/dnszones":
		return "azurerm_dns_zone"
	case "microsoft.network/privatednszones":
		return "azurerm_private_dns_zone"
	case "microsoft.documentdb/sqldatabases":
		return "azurerm_cosmosdb_sql_database"
	case "microsoft.documentdb/mongodatabases":
		return "azurerm_cosmosdb_mongo_database"
	case "microsoft.documentdb/mongocollection":
		return "azurerm_cosmosdb_mongo_collection"
	case "microsoft.dbformariadb/servers":
		return "azurerm_mariadb_server"
	case "microsoft.sql/servers/databases":
		return "azurerm_sql_database"
	case "microsoft.sql/managedInstances":
		return "azurerm_sql_managed_instance"
	case "microsoft.dbformysql/servers":
		return "azurerm_mysql_server"
	case "microsoft.dbforpostgresql/servers":
		return "azurerm_postgresql_server"
	case "microsoft.dbforpostgresql/flexibleservers":
		return "azurerm_postgresql_flexible_server"
	case "microsoft.dbformysql/flexibleservers":
		return "azurerm_mysql_flexible_server"
	case "microsoft.containerservice/managedclusters":
		return "azurerm_kubernetes_cluster"
	case "microsoft.web/hostingenvironments":
		return "azurerm_app_service_environment"
	case "microsoft.web/plan":
		return "azurerm_app_service_plan"
	case "microsoft.apimanagement/service":
		return "azurerm_api_management"
	case "microsoft.web/sites":
		return "azurerm_function_app"
	case "microsoft.search/searchservices":
		return "azurerm_search_service"
	case "microsoft.automation/automationaccounts":
		return "azurerm_automation_account"
	}
	return resourceType
}

func GetValues(resource Resource, resourceType string) (map[string]interface{}, error) {
	switch strings.ToLower(resourceType) {
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
	case "aws::ec2::volumegp3":
		return getAwsEbsVolumeGp3Values(resource)
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
		return getAwsEc2HostValues(resource)
	case "aws::lambda::function":
		return getAwsLambdaFunctionValues(resource)
	case "aws::elasticsearch::domain":
		return getAwsEsDomainValues(resource)
	case "aws::opensearch::domain":
		return getAwsOpenSearchDomainValues(resource)
	case "aws::dynamodb::table":
		return getAwsDynamoDbTableValues(resource)

	// Azure
	case "microsoft.compute/virtualmachines":
		return nil, nil
	case "microsoft.compute/disks":
		return getAzureComputeDiskValues(resource)
	case "microsoft.compute/images":
		return nil, nil
	case "microsoft.compute/snapshots":
		return getAzureComputeSnapshotValues(resource)
	case "microsoft.compute/virtualmachinescalesets":
		return nil, nil
	case "microsoft.network/loadbalancers":
		return getAzureLoadBalancerValues(resource)
	case "microsoft.network/loadbalancers/loadbalancingeules":
		return nil, nil
	case "microsoft.network/loadbalancers/outboundrules":
		return nil, nil
	case "microsoft.network/applicationgateways":
		return getAzureApplicationGatewayValues(resource)
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
	err = client.ES.Search(ctx, InventorySummaryIndex, string(b), &response)
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
	var indexes []struct {
		index        string
		resourceType string
	}
	for column, q := range d.EqualsQuals {
		if column == "resource_type" {
			if s, ok := q.GetValue().(*proto.QualValue_StringValue); ok && s != nil {
				indexes = []struct {
					index        string
					resourceType string
				}{{index: ResourceTypeToESIndex(s.StringValue), resourceType: s.StringValue}}
			} else if l := q.GetListValue(); l != nil {
				for _, v := range l.GetValues() {
					if v == nil {
						continue
					}
					indexes = append(indexes, struct {
						index        string
						resourceType string
					}{index: v.GetStringValue(), resourceType: v.GetStringValue()})
				}
			}
		}
	}

	req := schema.Submission{
		ID:        "submittion-1",
		CreatedAt: time.Now(),
		Resources: []schema.ResourceDef{},
	}

	var resources []Resource

	for _, index := range indexes {
		paginator, err := k.NewResourcePaginator(essdk.BuildFilterWithDefaultFieldName(ctx, d.QueryContext, resourceMapping,
			"", nil, encodedResourceCollectionFilters, clientType, true), d.QueryContext.Limit, index.index)
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
				values, err := GetValues(hit, index.resourceType)
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
	statusCode, err := httpclient.DoRequest(ctx, "GET", *cfg.PennywiseBaseURL+"/api/v1/cost/submission", nil, reqBody, &response)
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
