package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/web/mgmt/2020-06-01/web"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
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
			resourceGroup := strings.Split(*v.ID, "/")[4]

			values = append(values, Resource{
				ID: *v.ID,
				Description: model.AppServiceEnvironmentDescription{
					AppServiceEnvironmentResource: v,
					ResourceGroup:                 resourceGroup,
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
			resourceGroup := strings.Split(*v.ID, "/")[4]

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
				Description: model.AppServiceFunctionAppDescription{
					Site:               v,
					SiteAuthSettings:   authSettings,
					SiteConfigResource: configuration,
					ResourceGroup:      resourceGroup,
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
			resourceGroup := strings.Split(*v.ID, "/")[4]

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
				Description: model.AppServiceWebAppDescription{
					Site:               v,
					SiteConfigResource: op,
					SiteAuthSettings:   authSettings,
					VnetInfo:           vnetInfo,
					ResourceGroup:      resourceGroup,
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
