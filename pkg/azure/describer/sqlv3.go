package describer

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/mysql/mgmt/mysqlflexibleservers"
	"github.com/Azure/azure-sdk-for-go/services/preview/sql/mgmt/2017-03-01-preview/sql"
	sqlv3 "github.com/Azure/azure-sdk-for-go/services/preview/sql/mgmt/v3.0/sql"
	"github.com/Azure/azure-sdk-for-go/services/preview/sqlvirtualmachine/mgmt/2017-03-01-preview/sqlvirtualmachine"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func SqlServer(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	virtualNetworkClient := sql.NewVirtualNetworkRulesClient(subscription)
	virtualNetworkClient.Authorizer = authorizer

	privateEndpointClient := sqlv3.NewPrivateEndpointConnectionsClient(subscription)
	privateEndpointClient.Authorizer = authorizer

	encryptionProtectorsClient := sql.NewEncryptionProtectorsClient(subscription)
	encryptionProtectorsClient.Authorizer = authorizer

	firewallRulesClient := sql.NewFirewallRulesClient(subscription)
	firewallRulesClient.Authorizer = authorizer

	serverVulnerabilityClient := sqlv3.NewServerVulnerabilityAssessmentsClient(subscription)
	serverVulnerabilityClient.Authorizer = authorizer

	serverAzureClient := sql.NewServerAzureADAdministratorsClient(subscription)
	serverAzureClient.Authorizer = authorizer

	serverSecurityClient := sql.NewServerSecurityAlertPoliciesClient(subscription)
	serverSecurityClient.Authorizer = authorizer

	serverBlobClient := sql.NewServerBlobAuditingPoliciesClient(subscription)
	serverBlobClient.Authorizer = authorizer

	client := sqlv3.NewServersClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, server := range result.Values() {
			resourceGroupName := strings.Split(string(*server.ID), "/")[4]

			blobOp, err := serverBlobClient.ListByServer(ctx, resourceGroupName, *server.Name)
			if err != nil {
				return nil, err
			}
			bop := blobOp.Values()
			for blobOp.NotDone() {
				err := blobOp.NextWithContext(ctx)
				if err != nil {
					return nil, err
				}

				bop = append(bop, blobOp.Values()...)
			}

			securityOp, err := serverSecurityClient.ListByServer(ctx, resourceGroupName, *server.Name)
			if err != nil {
				return nil, err
			}
			sop := securityOp.Values()
			for securityOp.NotDone() {
				err := securityOp.NextWithContext(ctx)
				if err != nil {
					return nil, err
				}

				sop = append(sop, securityOp.Values()...)
			}

			adminOp, err := serverAzureClient.ListByServer(ctx, resourceGroupName, *server.Name)
			if err != nil {
				if !strings.Contains(err.Error(), "NotFound") {
					return nil, err
				}
			}

			vulnerabilityOp, err := serverVulnerabilityClient.ListByServer(ctx, resourceGroupName, *server.Name)
			if err != nil {
				return nil, err
			}
			vop := vulnerabilityOp.Values()
			for vulnerabilityOp.NotDone() {
				err := vulnerabilityOp.NextWithContext(ctx)
				if err != nil {
					return nil, err
				}

				vop = append(vop, vulnerabilityOp.Values()...)
			}

			firewallOp, err := firewallRulesClient.ListByServer(ctx, resourceGroupName, *server.Name)
			if err != nil {
				return nil, err
			}

			encryptionProtectorOp, err := encryptionProtectorsClient.ListByServer(ctx, resourceGroupName, *server.Name)
			if err != nil {
				return nil, err
			}
			eop := encryptionProtectorOp.Values()
			for encryptionProtectorOp.NotDone() {
				err := encryptionProtectorOp.NextWithContext(ctx)
				if err != nil {
					return nil, err
				}

				eop = append(eop, encryptionProtectorOp.Values()...)
			}

			pvEndpointOp, err := privateEndpointClient.ListByServer(ctx, resourceGroupName, *server.Name)
			if err != nil {
				return nil, err
			}
			pop := pvEndpointOp.Values()
			for pvEndpointOp.NotDone() {
				err := pvEndpointOp.NextWithContext(ctx)
				if err != nil {
					return nil, err
				}

				pop = append(pop, pvEndpointOp.Values()...)
			}

			networkOp, err := virtualNetworkClient.ListByServer(ctx, resourceGroupName, *server.Name)
			if err != nil {
				return nil, err
			}
			nop := networkOp.Values()
			for networkOp.NotDone() {
				err := networkOp.NextWithContext(ctx)
				if err != nil {
					return nil, err
				}

				nop = append(nop, networkOp.Values()...)
			}

			resource := Resource{
				ID:       *server.ID,
				Name:     *server.Name,
				Location: *server.Location,
				Description: model.SqlServerDescription{
					Server:                         server,
					ServerBlobAuditingPolicies:     bop,
					ServerSecurityAlertPolicies:    sop,
					ServerAzureADAdministrators:    adminOp.Value,
					ServerVulnerabilityAssessments: vop,
					FirewallRules:                  firewallOp.Value,
					EncryptionProtectors:           eop,
					PrivateEndpointConnections:     pop,
					VirtualNetworkRules:            nop,
					ResourceGroup:                  resourceGroupName,
				},
			}
			if stream != nil {
				if err := (*stream)(resource); err != nil {
					return nil, err
				}
			} else {
				values = append(values, resource)
			}
		}
		if !result.NotDone() {
			break
		}
		err = result.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}
	return values, nil
}

func SqlServerElasticPool(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := sqlv3.NewServersClient(subscription)
	client.Authorizer = authorizer

	elasticPoolClient := sql.NewElasticPoolsClient(subscription)
	elasticPoolClient.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, server := range result.Values() {
			serverResourceGroup := strings.Split(string(*server.ID), "/")[4]

			elasticPoolResult, err := elasticPoolClient.ListByServer(ctx, serverResourceGroup, *server.Name)
			if err != nil {
				return nil, err
			}

			for {
				for _, elasticPool := range *elasticPoolResult.Value {
					resourceGroup := strings.Split(string(*elasticPool.ID), "/")[4]
					resource := Resource{
						ID:       *elasticPool.ID,
						Name:     *elasticPool.Name,
						Location: *elasticPool.Location,
						Description: model.SqlServerElasticPoolDescription{
							Pool:          elasticPool,
							ServerName:    *server.Name,
							ResourceGroup: resourceGroup,
						},
					}
					if stream != nil {
						if err := (*stream)(resource); err != nil {
							return nil, err
						}
					} else {
						values = append(values, resource)
					}
				}
			}
		}
		if !result.NotDone() {
			break
		}
		err = result.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}
	return values, nil
}

func SqlServerVirtualMachine(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := sqlvirtualmachine.NewSQLVirtualMachinesClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, vm := range result.Values() {
			resourceGroup := strings.Split(string(*vm.ID), "/")[4]
			resource := Resource{
				ID:       *vm.ID,
				Name:     *vm.Name,
				Location: *vm.Location,
				Description: model.SqlServerVirtualMachineDescription{
					VirtualMachine: vm,
					ResourceGroup:  resourceGroup,
				},
			}
			if stream != nil {
				if err := (*stream)(resource); err != nil {
					return nil, err
				}
			} else {
				values = append(values, resource)
			}

		}
		if !result.NotDone() {
			break
		}
		err = result.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}
	return values, nil
}

func SqlServerFlexibleServer(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := mysqlflexibleservers.NewServersClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, fs := range result.Values() {
			resourceGroup := strings.Split(string(*fs.ID), "/")[4]
			resource := Resource{
				ID:       *fs.ID,
				Name:     *fs.Name,
				Location: *fs.Location,
				Description: model.SqlServerFlexibleServerDescription{
					FlexibleServer: fs,
					ResourceGroup:  resourceGroup,
				},
			}
			if stream != nil {
				if err := (*stream)(resource); err != nil {
					return nil, err
				}
			} else {
				values = append(values, resource)
			}

		}
		if !result.NotDone() {
			break
		}
		err = result.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}
	return values, nil
}
