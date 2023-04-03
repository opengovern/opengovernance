package describer

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-02-01/network"
	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2021-04-01-preview/insights"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func LoadBalancer(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := network.NewLoadBalancersClient(subscription)
	client.Authorizer = authorizer

	insightsClient := insights.NewDiagnosticSettingsClient(subscription)
	insightsClient.Authorizer = authorizer

	result, err := client.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, loadBalancer := range result.Values() {
			resourceGroup := strings.Split(*loadBalancer.ID, "/")[4]

			// Get diagnostic settings
			diagnosticSettings, err := insightsClient.List(ctx, *loadBalancer.ID)
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ID:       *loadBalancer.ID,
				Name:     *loadBalancer.Name,
				Location: *loadBalancer.Location,
				Description: model.LoadBalancerDescription{
					ResourceGroup:     resourceGroup,
					DiagnosticSetting: diagnosticSettings.Value,
					LoadBalancer:      loadBalancer,
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

func LoadBalancerBackendAddressPool(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := network.NewLoadBalancersClient(subscription)
	client.Authorizer = authorizer

	poolClient := network.NewLoadBalancerBackendAddressPoolsClient(subscription)
	poolClient.Authorizer = authorizer

	result, err := client.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, loadBalancer := range result.Values() {
			resourceGroup := strings.Split(*loadBalancer.ID, "/")[4]

			backendAddressPools, err := poolClient.List(ctx, resourceGroup, *loadBalancer.Name)
			if err != nil {
				return nil, err
			}
			for {
				for _, pool := range backendAddressPools.Values() {
					resourceGroup := strings.Split(*pool.ID, "/")[4]
					values = append(values, Resource{
						ID:       *pool.ID,
						Name:     *pool.Name,
						Location: *pool.Location,
						Description: model.LoadBalancerBackendAddressPoolDescription{
							ResourceGroup: resourceGroup,
							LoadBalancer:  loadBalancer,
							Pool:          pool,
						},
					})
				}
				if !backendAddressPools.NotDone() {
					break
				}
				err = backendAddressPools.NextWithContext(ctx)
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

func LoadBalancerNatRule(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := network.NewLoadBalancersClient(subscription)
	client.Authorizer = authorizer

	natRulesClient := network.NewInboundNatRulesClient(subscription)
	natRulesClient.Authorizer = authorizer

	result, err := client.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, loadBalancer := range result.Values() {
			resourceGroup := strings.Split(*loadBalancer.ID, "/")[4]

			natRules, err := natRulesClient.List(ctx, resourceGroup, *loadBalancer.Name)
			if err != nil {
				return nil, err
			}
			for {
				for _, natRule := range natRules.Values() {
					resourceGroup := strings.Split(*natRule.ID, "/")[4]
					values = append(values, Resource{
						ID:       *natRule.ID,
						Name:     *natRule.Name,
						Location: *loadBalancer.Location,
						Description: model.LoadBalancerNatRuleDescription{
							ResourceGroup:    resourceGroup,
							LoadBalancerName: *loadBalancer.Name,
							Rule:             natRule,
						},
					})
				}
				if !natRules.NotDone() {
					break
				}
				err = natRules.NextWithContext(ctx)
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

func LoadBalancerOutboundRule(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := network.NewLoadBalancersClient(subscription)
	client.Authorizer = authorizer

	outboundRulesClient := network.NewLoadBalancerOutboundRulesClient(subscription)
	outboundRulesClient.Authorizer = authorizer

	result, err := client.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, loadBalancer := range result.Values() {
			resourceGroup := strings.Split(*loadBalancer.ID, "/")[4]

			outboundRuleListResultPage, err := outboundRulesClient.List(ctx, resourceGroup, *loadBalancer.Name)
			if err != nil {
				return nil, err
			}
			for {
				for _, outboundRule := range outboundRuleListResultPage.Values() {
					resourceGroup := strings.Split(*outboundRule.ID, "/")[4]
					values = append(values, Resource{
						ID:       *outboundRule.ID,
						Name:     *outboundRule.Name,
						Location: *loadBalancer.Location,
						Description: model.LoadBalancerOutboundRuleDescription{
							ResourceGroup:    resourceGroup,
							LoadBalancerName: *loadBalancer.Name,
							Rule:             outboundRule,
						},
					})
				}
				if !outboundRuleListResultPage.NotDone() {
					break
				}
				err = outboundRuleListResultPage.NextWithContext(ctx)
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

func LoadBalancerProbe(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := network.NewLoadBalancersClient(subscription)
	client.Authorizer = authorizer

	probesClient := network.NewLoadBalancerProbesClient(subscription)
	probesClient.Authorizer = authorizer

	result, err := client.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, loadBalancer := range result.Values() {
			resourceGroup := strings.Split(*loadBalancer.ID, "/")[4]

			probeListResultPage, err := probesClient.List(ctx, resourceGroup, *loadBalancer.Name)
			if err != nil {
				return nil, err
			}
			for {
				for _, probe := range probeListResultPage.Values() {
					resourceGroup := strings.Split(*probe.ID, "/")[4]
					values = append(values, Resource{
						ID:       *probe.ID,
						Name:     *probe.Name,
						Location: *loadBalancer.Location,
						Description: model.LoadBalancerProbeDescription{
							ResourceGroup:    resourceGroup,
							LoadBalancerName: *loadBalancer.Name,
							Probe:            probe,
						},
					})
				}
				if !probeListResultPage.NotDone() {
					break
				}
				err = probeListResultPage.NextWithContext(ctx)
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

func LoadBalancerRule(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := network.NewLoadBalancersClient(subscription)
	client.Authorizer = authorizer

	rulesClient := network.NewLoadBalancerLoadBalancingRulesClient(subscription)
	rulesClient.Authorizer = authorizer

	result, err := client.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, loadBalancer := range result.Values() {
			resourceGroup := strings.Split(*loadBalancer.ID, "/")[4]

			ruleListResultPage, err := rulesClient.List(ctx, resourceGroup, *loadBalancer.Name)
			if err != nil {
				return nil, err
			}
			for {
				for _, rule := range ruleListResultPage.Values() {
					resourceGroup := strings.Split(*rule.ID, "/")[4]
					values = append(values, Resource{
						ID:       *rule.ID,
						Name:     *rule.Name,
						Location: *loadBalancer.Location,
						Description: model.LoadBalancerRuleDescription{
							ResourceGroup:    resourceGroup,
							LoadBalancerName: *loadBalancer.Name,
							Rule:             rule,
						},
					})
				}
				if !ruleListResultPage.NotDone() {
					break
				}
				err = ruleListResultPage.NextWithContext(ctx)
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
