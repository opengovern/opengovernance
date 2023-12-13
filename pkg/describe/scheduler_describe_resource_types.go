package describe

import (
	"errors"
	"github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-azure-describer/azure"
	analyticsDb "github.com/kaytu-io/kaytu-engine/pkg/analytics/db"
	apiAuth "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"strings"
)

func (s *Scheduler) ListDiscoveryResourceTypes() (api.ListDiscoveryResourceTypes, error) {
	var result api.ListDiscoveryResourceTypes

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
	controls, err := s.complianceClient.ListControl(&httpclient.Context{UserRole: apiAuth.InternalRole})
	if err != nil {
		return result, err
	}
	for _, control := range controls {
		if !control.ManualVerification && control.QueryID != nil {
			for _, query := range queries {
				if *control.QueryID == query.ID {
					rts := extractResourceTypes(query.QueryToExecute, source.Type(query.Connector))
					resourceTypes = append(resourceTypes, rts...)
					break
				}
			}
		}
	}
	//benchmarks, err := s.complianceClient.ListBenchmarks(httpclient.FromEchoContext(ctx))
	//if err != nil {
	//	return err
	//}
	//var benchmarksApi1Took, benchmarksApi2Took, benchmarksApi3Took int64
	//for _, bench := range benchmarks {
	//	rts, benchmarksApi1time, benchmarksApi2time, benchmarksApi3time, err := h.extractBenchmarkResourceTypes(httpclient.FromEchoContext(ctx), bench.ID)
	//	if err != nil {
	//		return err
	//	}
	//	benchmarksApi1Took += benchmarksApi1time
	//	benchmarksApi2Took += benchmarksApi2time
	//	benchmarksApi3Took += benchmarksApi3time
	//
	//	rts = UniqueArray(rts)
	//	resourceTypes = append(resourceTypes, rts...)
	//}
	//result.BenchmarksApi1Took = benchmarksApi1Took
	//result.BenchmarksApi2Took = benchmarksApi2Took
	//result.BenchmarksApi3Took = benchmarksApi3Took

	awsResourceTypes, azureResourceTypes := aws.ListResourceTypes(), azure.ListResourceTypes()
	for _, resourceType := range resourceTypes {
		found := false
		resourceType = strings.ToLower(resourceType)
		if strings.HasPrefix(resourceType, "aws") {
			for _, awsResourceType := range awsResourceTypes {
				if strings.ToLower(awsResourceType) == resourceType {
					found = true
					resourceType = awsResourceType
					break
				}
			}
			result.AWSResourceTypes = append(result.AWSResourceTypes, resourceType)
		} else if strings.HasPrefix(resourceType, "microsoft") {
			for _, azureResourceType := range azureResourceTypes {
				if strings.ToLower(azureResourceType) == resourceType {
					found = true
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

		if !found {
			s.logger.Error("resource type " + resourceType + " not found!")
		}
	}
	result.AzureResourceTypes = append(result.AzureResourceTypes, "Microsoft.CostManagement/CostByResourceType")
	result.AWSResourceTypes = append(result.AWSResourceTypes, "AWS::CostExplorer::ByServiceDaily")

	result.AWSResourceTypes = UniqueArray(result.AWSResourceTypes)
	result.AzureResourceTypes = UniqueArray(result.AzureResourceTypes)

	return result, nil
}
