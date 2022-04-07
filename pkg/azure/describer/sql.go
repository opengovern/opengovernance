package describer

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/preview/sql/mgmt/2017-03-01-preview/sql"
	sqlv3 "github.com/Azure/azure-sdk-for-go/services/preview/sql/mgmt/v3.0/sql"
	sqlV5 "github.com/Azure/azure-sdk-for-go/services/preview/sql/mgmt/v5.0/sql"

	"strings"

	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func MssqlManagedInstance(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	managedInstanceClient := sqlV5.NewManagedInstanceVulnerabilityAssessmentsClient(subscription)
	managedInstanceClient.Authorizer = authorizer

	managedServerClient := sqlV5.NewManagedServerSecurityAlertPoliciesClient(subscription)
	managedServerClient.Authorizer = authorizer

	managedInstanceEncClient := sqlV5.NewManagedInstanceEncryptionProtectorsClient(subscription)
	managedInstanceEncClient.Authorizer = authorizer

	client := sqlV5.NewManagedInstancesClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx, "")
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, managedInstance := range result.Values() {
			resourceGroup := strings.Split(string(*managedInstance.ID), "/")[4]
			managedInstanceName := *managedInstance.Name
			iop, err := managedInstanceClient.ListByInstance(ctx, resourceGroup, managedInstanceName)
			if err != nil {
				return nil, err
			}
			viop := iop.Values()
			for iop.NotDone() {
				err := iop.NextWithContext(ctx)
				if err != nil {
					return nil, err
				}

				viop = append(viop, iop.Values()...)
			}

			sop, err := managedServerClient.ListByInstance(ctx, resourceGroup, managedInstanceName)
			if err != nil {
				return nil, err
			}
			vsop := sop.Values()
			for sop.NotDone() {
				err := sop.NextWithContext(ctx)
				if err != nil {
					return nil, err
				}

				vsop = append(vsop, sop.Values()...)
			}

			eop, err := managedInstanceEncClient.ListByInstance(ctx, resourceGroup, managedInstanceName)
			if err != nil {
				return nil, err
			}
			veop := eop.Values()
			for eop.NotDone() {
				err := eop.NextWithContext(ctx)
				if err != nil {
					return nil, err
				}

				veop = append(veop, eop.Values()...)
			}

			values = append(values, Resource{
				ID:       *managedInstance.ID,
				Name:     *managedInstance.Name,
				Location: *managedInstance.Location,
				Description: model.MssqlManagedInstanceDescription{
					ManagedInstance:                         managedInstance,
					ManagedInstanceVulnerabilityAssessments: viop,
					ManagedDatabaseSecurityAlertPolicies:    vsop,
					ManagedInstanceEncryptionProtectors:     veop,
					ResourceGroup:                           resourceGroup,
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

func SqlDatabase(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	parentClient := sqlv3.NewServersClient(subscription)
	parentClient.Authorizer = authorizer

	databaseVulnerabilityScanClient := sqlV5.NewDatabaseVulnerabilityAssessmentScansClient(subscription)
	databaseVulnerabilityScanClient.Authorizer = authorizer

	databaseVulnerabilityClient := sqlV5.NewDatabaseVulnerabilityAssessmentsClient(subscription)
	databaseVulnerabilityClient.Authorizer = authorizer

	transparentDataClient := sql.NewTransparentDataEncryptionsClient(subscription)
	transparentDataClient.Authorizer = authorizer

	longTermClient := sqlV5.NewLongTermRetentionPoliciesClient(subscription)
	longTermClient.Authorizer = authorizer

	databasesClientClient := sql.NewDatabasesClient(subscription)
	databasesClientClient.Authorizer = authorizer

	client := sql.NewDatabasesClient(subscription)
	client.Authorizer = authorizer

	result, err := parentClient.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, server := range result.Values() {
			resourceGroupName := strings.Split(string(*server.ID), "/")[4]
			result, err := client.ListByServer(ctx, resourceGroupName, *server.Name, "", "")
			if err != nil {
				return nil, err
			}
			for _, database := range *result.Value {
				serverName := strings.Split(*database.ID, "/")[8]
				databaseName := *database.Name
				resourceGroupName := strings.Split(string(*database.ID), "/")[4]

				op, err := longTermClient.ListByDatabase(ctx, resourceGroupName, serverName, databaseName)
				if err != nil {
					return nil, err
				}
				longTermRetentionPolicies := op.Values()
				var longTermRetentionPolicy sqlV5.LongTermRetentionPolicy
				if len(longTermRetentionPolicies) > 0 {
					longTermRetentionPolicy = longTermRetentionPolicies[0]
				}

				transparentDataOp, err := transparentDataClient.Get(ctx, resourceGroupName, serverName, databaseName)
				if err != nil {
					return nil, err
				}

				dbVulnerabilityOp, err := databaseVulnerabilityClient.ListByDatabase(ctx, resourceGroupName, serverName, databaseName)
				if err != nil {
					return nil, err
				}
				c := dbVulnerabilityOp.Values()
				for dbVulnerabilityOp.NotDone() {
					err := dbVulnerabilityOp.NextWithContext(ctx)
					if err != nil {
						return nil, err
					}

					c = append(c, dbVulnerabilityOp.Values()...)
				}

				dbVulnerabilityScanOp, err := databaseVulnerabilityScanClient.ListByDatabase(ctx, resourceGroupName, serverName, databaseName)
				if err != nil {
					return nil, err
				}
				v := dbVulnerabilityScanOp.Values()
				for dbVulnerabilityScanOp.NotDone() {
					err := dbVulnerabilityScanOp.NextWithContext(ctx)
					if err != nil {
						return nil, err
					}

					v = append(v, dbVulnerabilityScanOp.Values()...)
				}

				getOp, err := client.Get(ctx, resourceGroupName, serverName, databaseName, "")
				if err != nil {
					return nil, err
				}

				values = append(values, Resource{
					ID:       *server.ID,
					Name:     *server.Name,
					Location: *server.Location,
					Description: model.SqlDatabaseDescription{
						Database:                           getOp,
						LongTermRetentionPolicy:            longTermRetentionPolicy,
						TransparentDataEncryption:          transparentDataOp,
						DatabaseVulnerabilityAssessments:   c,
						VulnerabilityAssessmentScanRecords: v,
						ResourceGroup:                      resourceGroupName,
					},
				})
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
