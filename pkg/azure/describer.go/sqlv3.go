package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/preview/sql/mgmt/2017-03-01-preview/sql"
	sqlv3 "github.com/Azure/azure-sdk-for-go/services/preview/sql/mgmt/v3.0/sql"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
)

func SqlServer(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
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

			securityOp, err := serverSecurityClient.ListByServer(ctx, resourceGroupName, *server.Name)
			if err != nil {
				return nil, err
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

			firewallOp, err := firewallRulesClient.ListByServer(ctx, resourceGroupName, *server.Name)
			if err != nil {
				return nil, err
			}

			encryptionProtectorOp, err := encryptionProtectorsClient.ListByServer(ctx, resourceGroupName, *server.Name)
			if err != nil {
				return nil, err
			}

			pvEndpointOp, err := privateEndpointClient.ListByServer(ctx, resourceGroupName, *server.Name)
			if err != nil {
				return nil, err
			}
			vop := pvEndpointOp.Values()
			for pvEndpointOp.NotDone() {
				err := pvEndpointOp.NextWithContext(ctx)
				if err != nil {
					return nil, err
				}

				vop = append(vop, pvEndpointOp.Values()...)
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

			values = append(values, Resource{
				ID: *server.ID,
				Description: model.SqlServerDescription{
					server,
					blobOp,
					securityOp,
					adminOp,
					vulnerabilityOp,
					firewallOp,
					encryptionProtectorOp,
					vop,
					nop,
				},
			})
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
