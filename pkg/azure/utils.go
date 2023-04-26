package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/authorization/mgmt/authorization"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/subscription/mgmt/subscription"
)

const (
	DefaultReaderRoleDefinitionIDTemplate = "/subscriptions/%s/providers/Microsoft.Authorization/roleDefinitions/acdd72a7-3385-48ef-bd42-f606fba81ae7"
)

func CheckSPNAccessPermission(authConf AuthConfig) error {
	authorizer, err := NewAuthorizerFromConfig(authConf)
	if err != nil {
		return err
	}
	// list subscriptions
	client := subscription.NewSubscriptionsClient()
	client.Authorizer = authorizer
	authorizer.WithAuthorization()

	_, err = client.ListComplete(context.TODO())
	if err != nil {
		return err
	}
	return nil
}

func CheckRole(authConf AuthConfig, subscriptionID string, roleDefinitionIDTemplate string) (bool, error) {
	if roleDefinitionIDTemplate == "" {
		return false, fmt.Errorf("roleDefinitionIDTemplate is empty")
	}
	roleDefinitionID := fmt.Sprintf(DefaultReaderRoleDefinitionIDTemplate, subscriptionID)

	authorizer, err := NewAuthorizerFromConfig(authConf)
	if err != nil {
		return false, err
	}

	client := authorization.NewRoleAssignmentsClient(subscriptionID)
	client.Authorizer = authorizer
	authorizer.WithAuthorization()

	it, err := client.ListComplete(context.TODO(), "")
	if err != nil {
		return false, err
	}

	for it.NotDone() {
		v := it.Value()

		if v.Properties.RoleDefinitionID != nil && *v.Properties.RoleDefinitionID == roleDefinitionID {
			return true, nil
		}

		if it.NotDone() {
			err := it.NextWithContext(context.TODO())
			if err != nil {
				return false, err
			}
		} else {
			break
		}
	}

	return false, nil
}
