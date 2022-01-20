package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/databoxedge/mgmt/2019-07-01/databoxedge"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure"
)

func DataboxEdgeDevice(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := databoxedge.NewDevicesClient(subscription)
	client.Authorizer = authorizer

	result, err := client.ListBySubscription(ctx, "")
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, v := range result.Values() {
			values = append(values, Resource{
				ID: *v.ID,
				Description: JSONAllFieldsMarshaller{
					azure.DataboxEdgeDeviceDescription{
						Device: v,
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
