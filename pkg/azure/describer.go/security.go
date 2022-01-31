package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/preview/security/mgmt/v1.0/security"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func SecurityCenterAutoProvisioning(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := security.NewAutoProvisioningSettingsClient(subscription, "")
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, v := range result.Values() {
			values = append(values, Resource{
				ID: *v.ID,
				Description: model.SecurityCenterAutoProvisioningDescription{
					AutoProvisioningSetting: v,
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

func SecurityCenterContact(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := security.NewContactsClient(subscription, "")
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, v := range result.Values() {
			values = append(values, Resource{
				ID: *v.ID,
				Description: model.SecurityCenterContactDescription{
					Contact: v,
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

func SecurityCenterJitNetworkAccessPolicy(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := security.NewJitNetworkAccessPoliciesClient(subscription, "")
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, v := range result.Values() {
			values = append(values, Resource{
				ID: *v.ID,
				Description: model.SecurityCenterJitNetworkAccessPolicyDescription{
					JitNetworkAccessPolicy: v,
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

func SecurityCenterSetting(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := security.NewSettingsClient(subscription, "")
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, v := range result.Values() {
			values = append(values, Resource{
				ID: *v.ID,
				Description: model.SecurityCenterSettingDescription{
					Setting: v,
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

func SecurityCenterSubscriptionPricing(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := security.NewPricingsClient(subscription, "")
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, v := range result.Values() {
			values = append(values, Resource{
				ID: *v.ID,
				Description: model.SecurityCenterSubscriptionPricingDescription{
					Pricing: v,
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
