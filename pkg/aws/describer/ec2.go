package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	acmtypes "github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func EC2CapacityReservation(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeCapacityReservationsPaginator(client, &ec2.DescribeCapacityReservationsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.CapacityReservations {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2CarrierGateway(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeCarrierGatewaysPaginator(client, &ec2.DescribeCarrierGatewaysInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.CarrierGateways {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2ClientVpnAuthorizationRule(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeClientVpnAuthorizationRulesPaginator(client, &ec2.DescribeClientVpnAuthorizationRulesInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.AuthorizationRules {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2ClientVpnEndpoint(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeClientVpnEndpointsPaginator(client, &ec2.DescribeClientVpnEndpointsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ClientVpnEndpoints {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2ClientVpnRoute(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeClientVpnRoutesPaginator(client, &ec2.DescribeClientVpnRoutesInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Routes {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2ClientVpnTargetNetworkAssociation(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeClientVpnTargetNetworksPaginator(client, &ec2.DescribeClientVpnTargetNetworksInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ClientVpnTargetNetworks {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2CustomerGateway(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	output, err := client.DescribeCustomerGateways(ctx, &ec2.DescribeCustomerGatewaysInput{})
	if err != nil {
		return nil, err
	}

	var values []interface{}
	for _, v := range output.CustomerGateways {
		values = append(values, v)
	}

	return values, nil
}

func EC2DHCPOptions(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeDhcpOptionsPaginator(client, &ec2.DescribeDhcpOptionsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.DhcpOptions {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2Fleet(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeFleetsPaginator(client, &ec2.DescribeFleetsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Fleets {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2EgressOnlyInternetGateway(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeEgressOnlyInternetGatewaysPaginator(client, &ec2.DescribeEgressOnlyInternetGatewaysInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.EgressOnlyInternetGateways {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2EIP(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	output, err := client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{})
	if err != nil {
		return nil, err
	}

	var values []interface{}
	for _, v := range output.Addresses {
		values = append(values, v)
	}

	return values, nil
}

// OMIT: Association is just an Id. It is part of the EC2EIP!
// func EC2EIPAssociation(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }

func EC2EnclaveCertificateIamRoleAssociation(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	certs, err := CertificateManagerCertificate(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := ec2.NewFromConfig(cfg)

	var values []interface{}
	for _, c := range certs {
		cert := c.(acmtypes.CertificateSummary)

		output, err := client.GetAssociatedEnclaveCertificateIamRoles(ctx, &ec2.GetAssociatedEnclaveCertificateIamRolesInput{
			CertificateArn: cert.CertificateArn,
		})
		if err != nil {
			return nil, err
		}

		for _, v := range output.AssociatedRoles {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2FlowLog(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeFlowLogsPaginator(client, &ec2.DescribeFlowLogsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.FlowLogs {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2Host(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeHostsPaginator(client, &ec2.DescribeHostsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Hosts {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2Instance(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeInstancesPaginator(client, &ec2.DescribeInstancesInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, r := range page.Reservations {
			for _, v := range r.Instances {
				values = append(values, v)
			}
		}
	}

	return values, nil
}

func EC2InternetGateway(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeInternetGatewaysPaginator(client, &ec2.DescribeInternetGatewaysInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.InternetGateways {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2LaunchTemplate(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeLaunchTemplatesPaginator(client, &ec2.DescribeLaunchTemplatesInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.LaunchTemplates {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2NatGateway(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeNatGatewaysPaginator(client, &ec2.DescribeNatGatewaysInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.NatGateways {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2NetworkAcl(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeNetworkAclsPaginator(client, &ec2.DescribeNetworkAclsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.NetworkAcls {
			values = append(values, v)
		}
	}

	return values, nil
}

// OMIT: Part of EC2NetworkAcl
// func EC2NetworkAclEntry(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }

// OMIT: Part of EC2NetworkAcl
// func EC2SubnetNetworkAclAssociation(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }

func EC2NetworkInsightsAnalysis(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeNetworkInsightsAnalysesPaginator(client, &ec2.DescribeNetworkInsightsAnalysesInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.NetworkInsightsAnalyses {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2NetworkInsightsPath(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeNetworkInsightsPathsPaginator(client, &ec2.DescribeNetworkInsightsPathsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.NetworkInsightsPaths {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2NetworkInterface(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeNetworkInterfacesPaginator(client, &ec2.DescribeNetworkInterfacesInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.NetworkInterfaces {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2NetworkInterfaceAttachment(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeNetworkInterfacesPaginator(client, &ec2.DescribeNetworkInterfacesInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, ni := range page.NetworkInterfaces {
			values = append(values, ni.Attachment)
		}
	}

	return values, nil
}

func EC2NetworkInterfacePermission(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeNetworkInterfacePermissionsPaginator(client, &ec2.DescribeNetworkInterfacePermissionsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.NetworkInterfacePermissions {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2PlacementGroup(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	output, err := client.DescribePlacementGroups(ctx, &ec2.DescribePlacementGroupsInput{})
	if err != nil {
		return nil, err
	}

	var values []interface{}
	for _, v := range output.PlacementGroups {
		values = append(values, v)
	}

	return values, nil
}

func EC2PrefixList(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribePrefixListsPaginator(client, &ec2.DescribePrefixListsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.PrefixLists {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2RouteTable(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeRouteTablesPaginator(client, &ec2.DescribeRouteTablesInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.RouteTables {
			values = append(values, v)
		}
	}

	return values, nil
}

// OMIT: Already part of EC2RouteTable
// func EC2Route(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }

// OMIT: Already part of EC2RouteTable
// func EC2GatewayRouteTableAssociation(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }

// OMIT: Already part of EC2RouteTable
// func EC2SubnetRouteTableAssociation(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }

// OMIT: Already part of EC2RouteTable
// func EC2VPNGatewayRoutePropagation(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }

func EC2LocalGatewayRouteTable(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeLocalGatewayRouteTablesPaginator(client, &ec2.DescribeLocalGatewayRouteTablesInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.LocalGatewayRouteTables {
			values = append(values, v)
		}
	}

	return values, nil
}

// OMIT: Part of EC2LocalGatewayRouteTable
// func EC2LocalGatewayRoute(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }

func EC2LocalGatewayRouteTableVPCAssociation(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeLocalGatewayRouteTableVpcAssociationsPaginator(client, &ec2.DescribeLocalGatewayRouteTableVpcAssociationsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.LocalGatewayRouteTableVpcAssociations {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2TransitGatewayRouteTable(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeTransitGatewayRouteTablesPaginator(client, &ec2.DescribeTransitGatewayRouteTablesInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.TransitGatewayRouteTables {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2TransitGatewayRouteTableAssociation(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewGetTransitGatewayRouteTableAssociationsPaginator(client, &ec2.GetTransitGatewayRouteTableAssociationsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Associations {
			values = append(values, v)
		}
	}

	return values, nil
}

// OMIT: Part of EC2TransitGatewayRouteTableAssociation
// func EC2TransitGatewayRoute(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }

func EC2TransitGatewayRouteTablePropagation(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewGetTransitGatewayRouteTablePropagationsPaginator(client, &ec2.GetTransitGatewayRouteTablePropagationsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.TransitGatewayRouteTablePropagations {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2SecurityGroup(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeSecurityGroupsPaginator(client, &ec2.DescribeSecurityGroupsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.SecurityGroups {
			values = append(values, v)
		}
	}

	return values, nil
}

// OMIT: Part of EC2SecurityGroup
// func EC2SecurityGroupEgress(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }

// OMIT: Part of EC2SecurityGroup
// func EC2SecurityGroupIngress(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }

func EC2SpotFleet(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeSpotFleetRequestsPaginator(client, &ec2.DescribeSpotFleetRequestsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.SpotFleetRequestConfigs {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2Subnet(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeSubnetsPaginator(client, &ec2.DescribeSubnetsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Subnets {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2SubnetCidrBlock(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeSubnetsPaginator(client, &ec2.DescribeSubnetsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Subnets {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2TrafficMirrorFilter(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeTrafficMirrorFiltersPaginator(client, &ec2.DescribeTrafficMirrorFiltersInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.TrafficMirrorFilters {
			values = append(values, v)
		}
	}

	return values, nil
}

// OMIT: Part of EC2TrafficMirrorFilter
// func EC2TrafficMirrorFilterRule(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }

func EC2TrafficMirrorSession(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeTrafficMirrorSessionsPaginator(client, &ec2.DescribeTrafficMirrorSessionsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.TrafficMirrorSessions {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2TrafficMirrorTarget(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeTrafficMirrorTargetsPaginator(client, &ec2.DescribeTrafficMirrorTargetsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.TrafficMirrorTargets {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2TransitGateway(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeTransitGatewaysPaginator(client, &ec2.DescribeTransitGatewaysInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.TransitGateways {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2TransitGatewayAttachment(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeTransitGatewayAttachmentsPaginator(client, &ec2.DescribeTransitGatewayAttachmentsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.TransitGatewayAttachments {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2TransitGatewayConnect(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeTransitGatewayConnectsPaginator(client, &ec2.DescribeTransitGatewayConnectsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.TransitGatewayConnects {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2TransitGatewayMulticastDomain(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeTransitGatewayMulticastDomainsPaginator(client, &ec2.DescribeTransitGatewayMulticastDomainsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.TransitGatewayMulticastDomains {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2TransitGatewayMulticastDomainAssociation(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	domains, err := EC2TransitGatewayMulticastDomain(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := ec2.NewFromConfig(cfg)
	var values []interface{}
	for _, domain := range domains {
		paginator := ec2.NewGetTransitGatewayMulticastDomainAssociationsPaginator(client, &ec2.GetTransitGatewayMulticastDomainAssociationsInput{
			TransitGatewayMulticastDomainId: domain.(types.TransitGatewayMulticastDomain).TransitGatewayMulticastDomainId,
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.MulticastDomainAssociations {
				values = append(values, v)
			}
		}
	}

	return values, nil
}

func EC2TransitGatewayMulticastGroupMember(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	domains, err := EC2TransitGatewayMulticastDomain(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := ec2.NewFromConfig(cfg)
	var values []interface{}
	for _, domain := range domains {
		paginator := ec2.NewSearchTransitGatewayMulticastGroupsPaginator(client, &ec2.SearchTransitGatewayMulticastGroupsInput{
			TransitGatewayMulticastDomainId: domain.(types.TransitGatewayMulticastDomain).TransitGatewayMulticastDomainId,
			Filters: []types.Filter{
				{
					Name:   aws.String("is-group-member"),
					Values: []string{"true"},
				},
			},
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.MulticastGroups {
				values = append(values, v)
			}
		}
	}

	return values, nil
}

func EC2TransitGatewayMulticastGroupSource(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	domains, err := EC2TransitGatewayMulticastDomain(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := ec2.NewFromConfig(cfg)
	var values []interface{}
	for _, domain := range domains {
		paginator := ec2.NewSearchTransitGatewayMulticastGroupsPaginator(client, &ec2.SearchTransitGatewayMulticastGroupsInput{
			TransitGatewayMulticastDomainId: domain.(types.TransitGatewayMulticastDomain).TransitGatewayMulticastDomainId,
			Filters: []types.Filter{
				{
					Name:   aws.String("is-group-source"),
					Values: []string{"true"},
				},
			},
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.MulticastGroups {
				values = append(values, v)
			}
		}
	}

	return values, nil
}

func EC2TransitGatewayPeeringAttachment(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeTransitGatewayPeeringAttachmentsPaginator(client, &ec2.DescribeTransitGatewayPeeringAttachmentsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.TransitGatewayPeeringAttachments {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2Volume(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeVolumesPaginator(client, &ec2.DescribeVolumesInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Volumes {
			values = append(values, v)
		}
	}

	return values, nil
}

// OMIT: Part of EC2Volume
// func EC2VolumeAttachment(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }

func EC2VPC(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeVpcsPaginator(client, &ec2.DescribeVpcsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Vpcs {
			values = append(values, v)
		}
	}

	return values, nil
}

// OMIT: Part of EC2Vpc
// func EC2VPCCidrBlock(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }

// OMIT: Not really type but used to make an association. DHCPOptionId is part of EC2Vpc and EC2DHCPOptions
// func EC2VPCDHCPOptionsAssociation(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }

func EC2VPCEndpoint(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeVpcEndpointsPaginator(client, &ec2.DescribeVpcEndpointsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.VpcEndpoints {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2VPCEndpointConnectionNotification(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeVpcEndpointConnectionNotificationsPaginator(client, &ec2.DescribeVpcEndpointConnectionNotificationsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ConnectionNotificationSet {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2VPCEndpointService(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)

	var values []interface{}
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.DescribeVpcEndpointServices(ctx, &ec2.DescribeVpcEndpointServicesInput{
			NextToken: prevToken,
		})
		if err != nil {
			return nil, err
		}

		for _, v := range output.ServiceDetails {
			values = append(values, v)
		}

		return output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}

func EC2VPCEndpointServicePermissions(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeVpcEndpointServicePermissionsPaginator(client, &ec2.DescribeVpcEndpointServicePermissionsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.AllowedPrincipals {
			values = append(values, v)
		}
	}

	return values, nil
}

// OMIT: Attaches an internet gateway, or a virtual private gateway to a VPC, enabling connectivity between the internet and the VPC.
// The attachment is part of the EC2InternetGateway or EC2VPNGateway!
// func EC2VPCGatewayAttachment(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }

func EC2VPCPeeringConnection(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeVpcPeeringConnectionsPaginator(client, &ec2.DescribeVpcPeeringConnectionsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.VpcPeeringConnections {
			values = append(values, v)
		}
	}

	return values, nil
}

func EC2VPNConnection(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	output, err := client.DescribeVpnConnections(ctx, &ec2.DescribeVpnConnectionsInput{})
	if err != nil {
		return nil, err
	}

	var values []interface{}
	for _, v := range output.VpnConnections {
		values = append(values, v)
	}

	return values, nil
}

// OMIT: Part of EC2VPNConnection
// func EC2VPNConnectionRoute(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }

func EC2VPNGateway(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ec2.NewFromConfig(cfg)
	output, err := client.DescribeVpnGateways(ctx, &ec2.DescribeVpnGatewaysInput{})
	if err != nil {
		return nil, err
	}

	var values []interface{}
	for _, v := range output.VpnGateways {
		values = append(values, v)
	}

	return values, nil
}
