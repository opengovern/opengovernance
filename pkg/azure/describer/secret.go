package describer

import (
	"context"
	"strings"

	secret "github.com/Azure/azure-sdk-for-go/services/keyvault/v7.1/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2019-09-01/keyvault"
)

func KeyVaultSecret(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	keyVaultClient := keyvault.NewVaultsClient(subscription)
	keyVaultClient.Authorizer = authorizer

	vaultsClient := keyvault.NewVaultsClient(subscription)
	vaultsClient.Authorizer = authorizer

	client := secret.New()
	client.Authorizer = authorizer

	maxResults := int32(100)
	result, err := keyVaultClient.List(ctx, &maxResults)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, vault := range result.Values() {
			vaultURI := "https://" + *vault.Name + ".vault.azure.net/"
			maxResults := int32(25)
			res, err := client.GetSecrets(ctx, vaultURI, &maxResults)
			if err != nil {
				return nil, err
			}

			for {
				for _, sc := range res.Values() {
					splitID := strings.Split(*sc.ID, "/")
					resourceGroup := splitID[4]

					if !*sc.Attributes.Enabled {
						continue
					}
					op, err := client.GetSecret(ctx, vaultURI, resourceGroup, "")
					if err != nil {
						return nil, err
					}

					maxResults := int32(100)
					vaultsOp, err := vaultsClient.List(ctx, &maxResults)
					if err != nil {
						return nil, err
					}

					var vaultID, location string
					for _, i := range vaultsOp.Values() {
						if *i.Name == *vault.Name {
							vaultID = *i.ID
							location = *i.Location
						}
					}
					splitVaultID := strings.Split(vaultID, "/")
					akas := []string{"azure:///subscriptions/" + subscription + "/resourceGroups/" + splitVaultID[4] +
						"/providers/Microsoft.KeyVault/vaults/" + *vault.Name + "/secrets/" + splitID[4],
						"azure:///subscriptions/" + subscription + "/resourcegroups/" + splitVaultID[4] +
							"/providers/microsoft.keyvault/vaults/" + *vault.Name + "/secrets/" + splitID[4]}

					turbotData := map[string]interface{}{
						"SubscriptionId": subscription,
						"ResourceGroup":  splitVaultID[4],
						"Location":       location,
						"Akas":           akas,
					}

					resource := Resource{
						ID:       *sc.ID,
						Name:     *sc.ID,
						Location: "global",
						Description: model.KeyVaultSecretDescription{
							SecretItem:    sc,
							SecretBundle:  op,
							TurboData:     turbotData,
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
				if !res.NotDone() {
					break
				}
				err = res.NextWithContext(ctx)
				if err != nil {
					return nil, err
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
