package describe

import (
	"errors"
	"strings"

	"github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-azure-describer/azure"
	analyticsDb "github.com/kaytu-io/kaytu-engine/pkg/analytics/db"
	apiAuth "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/models"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

func (s *Scheduler) ListDiscoveryResourceTypes() (api.ListDiscoveryResourceTypes, error) {
	var result api.ListDiscoveryResourceTypes

	assetDiscoveryEnabledMetadata, err := s.metadataClient.GetConfigMetadata(&httpclient.Context{UserRole: apiAuth.InternalRole}, models.MetadataKeyAssetDiscoveryEnabled)
	if err != nil {
		return result, err
	}
	assetDiscoveryEnabled := assetDiscoveryEnabledMetadata.GetValue().(bool)

	azureDiscoveryType, err := s.metadataClient.GetConfigMetadata(&httpclient.Context{UserRole: apiAuth.InternalRole}, models.MetadataKeyAzureDiscoveryRequiredOnly)
	if err != nil {
		return result, err
	}

	awsDiscoveryType, err := s.metadataClient.GetConfigMetadata(&httpclient.Context{UserRole: apiAuth.InternalRole}, models.MetadataKeyAWSDiscoveryRequiredOnly)
	if err != nil {
		return result, err
	}

	azureRequiredOnly := azureDiscoveryType.GetValue().(bool)
	awsRequiredOnly := awsDiscoveryType.GetValue().(bool)

	awsResourceTypes, azureResourceTypes := aws.ListResourceTypes(), azure.ListResourceTypes()
	if !assetDiscoveryEnabled {
		var rts []string

		for _, rt := range awsResourceTypes {
			if !strings.Contains(rt, "Cost") {
				continue
			}
			rts = append(rts, rt)
		}
		awsResourceTypes = rts

		rts = nil
		for _, rt := range azureResourceTypes {
			if !strings.Contains(rt, "Cost") {
				continue
			}
			rts = append(rts, rt)
		}
		azureResourceTypes = rts
	}

	if !azureRequiredOnly && !awsRequiredOnly {
		result.AzureResourceTypes = azureResourceTypes
		result.AWSResourceTypes = awsResourceTypes
		return result, nil
	}

	var resourceTypes []string
	assetMetrics, err := s.inventoryClient.ListAnalyticsMetrics(&httpclient.Context{UserRole: apiAuth.InternalRole}, analyticsDb.MetricTypeAssets)
	if err != nil {
		return result, err
	}
	spendMetrics, err := s.inventoryClient.ListAnalyticsMetrics(&httpclient.Context{UserRole: apiAuth.InternalRole}, analyticsDb.MetricTypeSpend)
	if err != nil {
		return result, err
	}
	for _, metric := range append(assetMetrics, spendMetrics...) {
		for _, connector := range metric.Connectors {
			rts := extractResourceTypes(metric.Query, connector)
			resourceTypes = append(resourceTypes, rts...)
		}
	}
	result.AzureResourceTypes = append(result.AzureResourceTypes, "Microsoft.CostManagement/CostByResourceType")
	result.AWSResourceTypes = append(result.AWSResourceTypes, "AWS::CostExplorer::ByServiceDaily")

	insights, err := s.complianceClient.ListInsights(&httpclient.Context{UserRole: apiAuth.InternalRole})
	if err != nil {
		return result, err
	}
	for _, ins := range insights {
		rts := extractResourceTypes(ins.Query.QueryToExecute, ins.Connector)
		resourceTypes = append(resourceTypes, rts...)
	}

	queries, err := s.complianceClient.ListQueries(&httpclient.Context{UserRole: apiAuth.InternalRole})
	if err != nil {
		return result, err
	}
	controls, err := s.complianceClient.ListControl(&httpclient.Context{UserRole: apiAuth.InternalRole}, nil)
	if err != nil {
		return result, err
	}
	for _, control := range controls {
		if !control.ManualVerification && control.Query != nil {
			for _, query := range queries {
				if control.Query.ID == query.ID {
					rts := extractResourceTypes(query.QueryToExecute, source.Type(query.Connector))
					resourceTypes = append(resourceTypes, rts...)
					break
				}
			}
		}
	}

	for _, resourceType := range resourceTypes {
		resourceType = strings.ToLower(resourceType)
		if strings.HasPrefix(resourceType, "aws") {
			for _, awsResourceType := range awsResourceTypes {
				if strings.ToLower(awsResourceType) == resourceType {
					resourceType = awsResourceType
					break
				}
			}
			result.AWSResourceTypes = append(result.AWSResourceTypes, resourceType)
		} else if strings.HasPrefix(resourceType, "microsoft") {
			for _, azureResourceType := range azureResourceTypes {
				if strings.ToLower(azureResourceType) == resourceType {
					resourceType = azureResourceType
					break
				}
			}
			result.AzureResourceTypes = append(result.AzureResourceTypes, resourceType)
		} else if strings.HasPrefix(resourceType, "azure") {
			result.AzureResourceTypes = append(result.AzureResourceTypes, resourceType)
		} else {
			return result, errors.New("invalid resource type:" + resourceType)
		}
	}

	result.AWSResourceTypes = UniqueArray(result.AWSResourceTypes)
	result.AzureResourceTypes = UniqueArray(result.AzureResourceTypes)

	if !azureRequiredOnly {
		result.AzureResourceTypes = azureResourceTypes
	}
	if !awsRequiredOnly {
		result.AWSResourceTypes = awsResourceTypes
	}

	return result, nil
}
