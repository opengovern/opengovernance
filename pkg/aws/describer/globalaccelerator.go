package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/globalaccelerator"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func GlobalAcceleratorAccelerator(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := globalaccelerator.NewFromConfig(cfg)
	paginator := globalaccelerator.NewListAcceleratorsPaginator(client, &globalaccelerator.ListAcceleratorsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, accelerator := range page.Accelerators {
			attribute, err := client.DescribeAcceleratorAttributes(ctx, &globalaccelerator.DescribeAcceleratorAttributesInput{
				AcceleratorArn: accelerator.AcceleratorArn,
			})
			if err != nil {
				return nil, err
			}

			tags, err := client.ListTagsForResource(ctx, &globalaccelerator.ListTagsForResourceInput{
				ResourceArn: accelerator.AcceleratorArn,
			})
			if err != nil {
				return nil, err
			}

			resource := Resource{
				ARN:  *accelerator.AcceleratorArn,
				Name: *accelerator.Name,
				Description: model.GlobalAcceleratorAcceleratorDescription{
					Accelerator:           accelerator,
					AcceleratorAttributes: attribute.AcceleratorAttributes,
					Tags:                  tags.Tags,
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

	return values, nil
}

func GlobalAcceleratorListener(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := globalaccelerator.NewFromConfig(cfg)
	paginator := globalaccelerator.NewListAcceleratorsPaginator(client, &globalaccelerator.ListAcceleratorsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, accelerator := range page.Accelerators {
			listenerPaginator := globalaccelerator.NewListListenersPaginator(client, &globalaccelerator.ListListenersInput{
				AcceleratorArn: accelerator.AcceleratorArn,
			})
			for listenerPaginator.HasMorePages() {
				listenerPage, err := listenerPaginator.NextPage(ctx)
				if err != nil {
					return nil, err
				}
				for _, listener := range listenerPage.Listeners {
					resource := Resource{
						ARN:  *listener.ListenerArn,
						Name: *listener.ListenerArn,
						Description: model.GlobalAcceleratorListenerDescription{
							Listener:       listener,
							AcceleratorArn: *accelerator.AcceleratorArn,
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
	}

	return values, nil
}

func GlobalAcceleratorEndpointGroup(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := globalaccelerator.NewFromConfig(cfg)
	paginator := globalaccelerator.NewListAcceleratorsPaginator(client, &globalaccelerator.ListAcceleratorsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, accelerator := range page.Accelerators {
			listenerPaginator := globalaccelerator.NewListListenersPaginator(client, &globalaccelerator.ListListenersInput{
				AcceleratorArn: accelerator.AcceleratorArn,
			})
			for listenerPaginator.HasMorePages() {
				listenerPage, err := listenerPaginator.NextPage(ctx)
				if err != nil {
					return nil, err
				}
				for _, listener := range listenerPage.Listeners {
					endpointGroupPaginator := globalaccelerator.NewListEndpointGroupsPaginator(client, &globalaccelerator.ListEndpointGroupsInput{
						ListenerArn: listener.ListenerArn,
					})
					for endpointGroupPaginator.HasMorePages() {
						endpointGroupPage, err := endpointGroupPaginator.NextPage(ctx)
						if err != nil {
							return nil, err
						}
						for _, endpointGroup := range endpointGroupPage.EndpointGroups {
							resource := Resource{
								ARN:  *endpointGroup.EndpointGroupArn,
								Name: *endpointGroup.EndpointGroupArn,
								Description: model.GlobalAcceleratorEndpointGroupDescription{
									EndpointGroup:  endpointGroup,
									ListenerArn:    *listener.ListenerArn,
									AcceleratorArn: *accelerator.AcceleratorArn,
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
			}
		}
	}

	return values, nil
}
