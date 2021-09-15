package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/aws/aws-sdk-go-v2/service/route53resolver"
)

func Route53HealthCheck(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := route53.NewFromConfig(cfg)

	var values []interface{}
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.ListHealthChecks(ctx, &route53.ListHealthChecksInput{Marker: prevToken})
		if err != nil {
			return nil, err
		}

		for _, v := range output.HealthChecks {
			values = append(values, v)
		}

		return output.NextMarker, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}

func Route53HostedZone(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := route53.NewFromConfig(cfg)

	var values []interface{}
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.ListHostedZones(ctx, &route53.ListHostedZonesInput{Marker: prevToken})
		if err != nil {
			return nil, err
		}

		for _, v := range output.HostedZones {
			values = append(values, v)
		}

		return output.NextMarker, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}

func Route53DNSSEC(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	zones, err := Route53HostedZone(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := route53.NewFromConfig(cfg)

	var values []interface{}
	for _, zone := range zones {
		output, err := client.GetDNSSEC(ctx, &route53.GetDNSSECInput{
			HostedZoneId: zone.(types.HostedZone).Id,
		})
		if err != nil {
			return nil, err
		}

		values = append(values, output)
	}

	return values, nil
}

// OMIT: Part of the Route53DNSSEC
// func Route53KeySigningKey(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }

func Route53RecordSet(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	zones, err := Route53HostedZone(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := route53.NewFromConfig(cfg)

	var values []interface{}
	for _, zone := range zones {
		var prevType types.RRType
		err = PaginateRetrieveAll(func(prevName *string) (nextName *string, err error) {
			output, err := client.ListResourceRecordSets(ctx, &route53.ListResourceRecordSetsInput{
				HostedZoneId:    zone.(types.HostedZone).Id,
				StartRecordName: prevName,
				StartRecordType: prevType,
			})
			if err != nil {
				return nil, err
			}

			for _, v := range output.ResourceRecordSets {
				values = append(values, v)
			}

			prevType = output.NextRecordType
			return output.NextRecordName, nil
		})
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}

// OMIT: Already part of Route53RecordSet. Not queriable seperatly.
// func Route53RecordSetGroup(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }

func Route53ResolverFirewallDomainList(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := route53resolver.NewFromConfig(cfg)
	paginator := route53resolver.NewListFirewallDomainListsPaginator(client, &route53resolver.ListFirewallDomainListsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.FirewallDomainLists {
			values = append(values, v)
		}
	}

	return values, nil
}

func Route53ResolverFirewallRuleGroup(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := route53resolver.NewFromConfig(cfg)
	paginator := route53resolver.NewListFirewallRuleGroupsPaginator(client, &route53resolver.ListFirewallRuleGroupsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.FirewallRuleGroups {
			values = append(values, v)
		}
	}

	return values, nil
}

func Route53ResolverFirewallRuleGroupAssociation(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := route53resolver.NewFromConfig(cfg)
	paginator := route53resolver.NewListFirewallRuleGroupAssociationsPaginator(client, &route53resolver.ListFirewallRuleGroupAssociationsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.FirewallRuleGroupAssociations {
			values = append(values, v)
		}
	}

	return values, nil
}

// TODO
func Route53ResolverResolverDNSSECConfig(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	vpcs, err := EC2VPC(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := route53resolver.NewFromConfig(cfg)

	var values []interface{}
	for _, vpc := range vpcs {
		output, err := client.GetResolverDnssecConfig(ctx, &route53resolver.GetResolverDnssecConfigInput{
			ResourceId: vpc.(ec2types.Vpc).VpcId,
		})
		if err != nil {
			return nil, err
		}

		values = append(values, output.ResolverDNSSECConfig)
	}

	return values, nil
}

func Route53ResolverResolverEndpoint(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := route53resolver.NewFromConfig(cfg)
	paginator := route53resolver.NewListResolverEndpointsPaginator(client, &route53resolver.ListResolverEndpointsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ResolverEndpoints {
			values = append(values, v)
		}
	}

	return values, nil
}

func Route53ResolverResolverQueryLoggingConfig(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := route53resolver.NewFromConfig(cfg)
	paginator := route53resolver.NewListResolverQueryLogConfigsPaginator(client, &route53resolver.ListResolverQueryLogConfigsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ResolverQueryLogConfigs {
			values = append(values, v)
		}
	}

	return values, nil
}

func Route53ResolverResolverQueryLoggingConfigAssociation(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := route53resolver.NewFromConfig(cfg)
	paginator := route53resolver.NewListResolverQueryLogConfigAssociationsPaginator(client, &route53resolver.ListResolverQueryLogConfigAssociationsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ResolverQueryLogConfigAssociations {
			values = append(values, v)
		}
	}

	return values, nil
}

func Route53ResolverResolverRule(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := route53resolver.NewFromConfig(cfg)
	paginator := route53resolver.NewListResolverRulesPaginator(client, &route53resolver.ListResolverRulesInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ResolverRules {
			values = append(values, v)
		}
	}

	return values, nil
}

func Route53ResolverResolverRuleAssociation(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := route53resolver.NewFromConfig(cfg)
	paginator := route53resolver.NewListResolverRuleAssociationsPaginator(client, &route53resolver.ListResolverRuleAssociationsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ResolverRuleAssociations {
			values = append(values, v)
		}
	}

	return values, nil
}
