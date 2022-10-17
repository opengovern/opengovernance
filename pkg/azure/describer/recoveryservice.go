package describer

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/recoveryservices/mgmt/recoveryservices"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func RecoveryServicesVault(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := recoveryservices.NewVaultsClient(subscription)
	client.Authorizer = authorizer

	result, err := client.ListBySubscriptionID(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, vault := range result.Values() {
			resourceGroup := strings.Split(*vault.ID, "/")[4]

			values = append(values, Resource{
				ID:       *vault.ID,
				Name:     *vault.Name,
				Location: *vault.Location,
				Description: model.RecoveryServicesVaultDescription{
					Vault:         vault,
					ResourceGroup: resourceGroup,
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
