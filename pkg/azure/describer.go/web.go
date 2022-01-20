package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/web/mgmt/2020-06-01/web"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure"
)

func AppServiceEnvironment(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := web.NewAppServiceEnvironmentsClient(subscription)
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
				Description: JSONAllFieldsMarshaller{
					azure.AppServiceEnvironmentDescription{
						AppServiceEnvironmentResource: v,
					},
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

//TODO-Saleh resource??
func AppServiceFunctionApp(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := web.NewAppsClient(subscription)
	client.Authorizer = authorizer

	webClient := web.NewAppsClient(subscription)
	webClient.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, v := range result.Values() {
			authSettings, err := webClient.GetAuthSettings(ctx, *v.SiteProperties.ResourceGroup, *v.Name)
			if err != nil {
				return nil, err
			}

			configuration, err := webClient.GetConfiguration(ctx, *v.SiteProperties.ResourceGroup, *v.Name)
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ID: *v.ID,
				Description: JSONAllFieldsMarshaller{
					azure.AppServiceFunctionAppDescription{
						Site:               v,
						SiteAuthSettings:   authSettings,
						SiteConfigResource: configuration,
					},
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

//TODO-Saleh resource??
func AppServiceWebApp(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := web.NewAppsClient(subscription)
	client.Authorizer = authorizer

	webClient := web.NewAppsClient(subscription)
	webClient.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, v := range result.Values() {
			op, err := webClient.GetConfiguration(ctx, *v.SiteProperties.ResourceGroup, *v.Name)
			if err != nil {
				return nil, err
			}

			// Return nil, if no virtual network is configured
			var vnetInfo web.VnetInfo
			if *v.SiteConfig.VnetName != "" {
				vnetInfo, err = webClient.GetVnetConnection(ctx, *v.SiteProperties.ResourceGroup, *v.Name, *v.SiteConfig.VnetName)
				if err != nil {
					return nil, err
				}
			}

			authSettings, err := webClient.GetAuthSettings(ctx, *v.SiteProperties.ResourceGroup, *v.Name)
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ID: *v.ID,
				Description: JSONAllFieldsMarshaller{
					azure.AppServiceWebAppDescription{
						Site:               v,
						SiteConfigResource: op,
						SiteAuthSettings:   authSettings,
						VnetInfo:           vnetInfo,
					},
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
