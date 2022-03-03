package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/services/preview/eventgrid/mgmt/2021-06-01-preview/eventgrid"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
)

func EventGridDomainTopic(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	rgs, err := resourceGroup(ctx, authorizer, subscription)
	if err != nil {
		return nil, err
	}

	client := eventgrid.NewDomainTopicsClient(subscription)
	client.Authorizer = authorizer

	var values []Resource
	for _, rg := range rgs {
		domains, err := eventGridDomain(ctx, authorizer, subscription, *rg.Name)
		if err != nil {
			return nil, err
		}

		for _, domain := range domains {
			it, err := client.ListByDomainComplete(ctx, *rg.Name, *domain.Name, "", nil)
			if err != nil {
				return nil, err
			}

			for v := it.Value(); it.NotDone(); v = it.Value() {
				values = append(values, Resource{
					ID:          *v.ID,
					Location:    "global",
					Description: JSONAllFieldsMarshaller{Value: v},
				})

				err := it.NextWithContext(ctx)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return values, nil
}

func eventGridDomain(ctx context.Context, authorizer autorest.Authorizer, subscription string, resourceGroup string) ([]eventgrid.Domain, error) {
	client := eventgrid.NewDomainsClient(subscription)
	client.Authorizer = authorizer

	it, err := client.ListByResourceGroupComplete(ctx, resourceGroup, "", nil)
	if err != nil {
		return nil, err
	}

	var values []eventgrid.Domain
	for v := it.Value(); it.NotDone(); v = it.Value() {
		values = append(values, v)

		err := it.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}

func EventGridDomain(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	insightsClient := insights.NewDiagnosticSettingsClient(subscription)
	insightsClient.Authorizer = authorizer

	client := eventgrid.NewDomainsClient(subscription)
	client.Authorizer = authorizer

	result, err := client.ListBySubscription(ctx, "", nil)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, domain := range result.Values() {
			resourceGroup := strings.Split(*domain.ID, "/")[4]

			id := *domain.ID
			eventgridListOp, err := insightsClient.List(ctx, id)
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ID:       *domain.ID,
				Location: *domain.Location,
				Description: model.EventGridDomainDescription{
					Domain:                      domain,
					DiagnosticSettingsResources: eventgridListOp.Value,
					ResourceGroup:               resourceGroup,
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

func EventGridTopic(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	insightsClient := insights.NewDiagnosticSettingsClient(subscription)
	insightsClient.Authorizer = authorizer

	client := eventgrid.NewTopicsClient(subscription)
	client.Authorizer = authorizer

	result, err := client.ListBySubscription(ctx, "", nil)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, topic := range result.Values() {
			resourceGroup := strings.Split(*topic.ID, "/")[4]

			eventgridListOp, err := insightsClient.List(ctx, *topic.ID)
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ID:       *topic.ID,
				Location: *topic.Location,
				Description: model.EventGridTopicDescription{
					Topic:                       topic,
					DiagnosticSettingsResources: eventgridListOp.Value,
					ResourceGroup:               resourceGroup,
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
