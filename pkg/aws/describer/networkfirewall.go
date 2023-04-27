package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func NetworkFirewallFirewall(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := networkfirewall.NewFromConfig(cfg)
	paginator := networkfirewall.NewListFirewallsPaginator(client, &networkfirewall.ListFirewallsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Firewalls {
			firewall, err := client.DescribeFirewall(ctx, &networkfirewall.DescribeFirewallInput{
				FirewallArn: v.FirewallArn,
			})
			if err != nil {
				return nil, err
			}

			resource := Resource{
				ARN:  *v.FirewallArn,
				Name: *v.FirewallName,
				Description: model.NetworkFirewallFirewallDescription{
					Firewall: *firewall.Firewall,
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
