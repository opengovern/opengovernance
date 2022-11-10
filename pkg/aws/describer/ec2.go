package describer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func EC2VolumeSnapshot(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	var values []Resource
	client := ec2.NewFromConfig(cfg)

	var ownerId = "owner-id"
	ownerFilter := types.Filter{
		Name: &ownerId,
		Values: []string{
			"self",
		},
	}

	describeCtx := GetDescribeContext(ctx)

	paginator := ec2.NewDescribeSnapshotsPaginator(client, &ec2.DescribeSnapshotsInput{
		Filters: []types.Filter{
			ownerFilter,
		},
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, snapshot := range page.Snapshots {
			// This prevents Implicit memory aliasing in for loop
			snapshot := snapshot
			attrs, err := client.DescribeSnapshotAttribute(ctx, &ec2.DescribeSnapshotAttributeInput{
				Attribute:  types.SnapshotAttributeNameCreateVolumePermission,
				SnapshotId: snapshot.SnapshotId,
			})
			if err != nil {
				return nil, err
			}

			arn := "arn:" + describeCtx.Partition + ":ec2:" + describeCtx.Region + ":" + describeCtx.AccountID + ":snapshot/" + *snapshot.SnapshotId
			values = append(values, Resource{
				ARN:  arn,
				Name: *snapshot.SnapshotId,
				Description: model.EC2VolumeSnapshotDescription{
					Snapshot:                &snapshot,
					CreateVolumePermissions: attrs.CreateVolumePermissions,
				},
			})
		}
	}

	return values, nil
}

func EC2Volume(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	var values []Resource
	client := ec2.NewFromConfig(cfg)

	describeCtx := GetDescribeContext(ctx)

	paginator := ec2.NewDescribeVolumesPaginator(client, &ec2.DescribeVolumesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, volume := range page.Volumes {
			volume := volume
			var description model.EC2VolumeDescription
			description.Volume = &volume

			attrs := []types.VolumeAttributeName{
				types.VolumeAttributeNameAutoEnableIO,
				types.VolumeAttributeNameProductCodes,
			}

			for _, attr := range attrs {
				attrs, err := client.DescribeVolumeAttribute(ctx, &ec2.DescribeVolumeAttributeInput{
					Attribute: attr,
					VolumeId:  volume.VolumeId,
				})
				if err != nil {
					return nil, err
				}

				switch attr {
				case types.VolumeAttributeNameAutoEnableIO:
					description.Attributes.AutoEnableIO = *attrs.AutoEnableIO.Value
				case types.VolumeAttributeNameProductCodes:
					description.Attributes.ProductCodes = attrs.ProductCodes
				}
			}

			arn := "arn:" + describeCtx.Partition + ":ec2:" + describeCtx.Region + ":" + describeCtx.AccountID + ":volume/" + *volume.VolumeId
			values = append(values, Resource{
				ARN:         arn,
				Name:        *volume.VolumeId,
				Description: description,
			})
		}
	}

	return values, nil
}

func EC2CapacityReservation(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeCapacityReservationsPaginator(client, &ec2.DescribeCapacityReservationsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if isErr(err, "InvalidCapacityReservationId.NotFound") || isErr(err, "InvalidCapacityReservationId.Unavailable") || isErr(err, "InvalidCapacityReservationId.Malformed") {
				continue
			}
			return nil, err
		}

		for _, v := range page.CapacityReservations {
			values = append(values, Resource{
				ARN:  *v.CapacityReservationArn,
				Name: *v.CapacityReservationId,
				Description: model.EC2CapacityReservationDescription{
					CapacityReservation: v,
				},
			})
		}
	}

	return values, nil
}

func EC2CapacityReservationFleet(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeCapacityReservationFleetsPaginator(client, &ec2.DescribeCapacityReservationFleetsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.CapacityReservationFleets {
			values = append(values, Resource{
				ARN:  *v.CapacityReservationFleetArn,
				Name: *v.CapacityReservationFleetId,
				Description: model.EC2CapacityReservationFleetDescription{
					CapacityReservationFleet: v,
				},
			})
		}
	}

	return values, nil
}

func EC2CarrierGateway(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeCarrierGatewaysPaginator(client, &ec2.DescribeCarrierGatewaysInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.CarrierGateways {
			values = append(values, Resource{
				ID:          *v.CarrierGatewayId,
				Name:        *v.CarrierGatewayId,
				Description: v,
			})
		}
	}

	return values, nil
}

func EC2ClientVpnAuthorizationRule(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	endpoints, err := EC2ClientVpnEndpoint(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := ec2.NewFromConfig(cfg)

	var values []Resource
	for _, e := range endpoints {
		endpoint := e.Description.(types.ClientVpnEndpoint)
		paginator := ec2.NewDescribeClientVpnAuthorizationRulesPaginator(client, &ec2.DescribeClientVpnAuthorizationRulesInput{
			ClientVpnEndpointId: endpoint.ClientVpnEndpointId,
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.AuthorizationRules {
				values = append(values, Resource{
					ID:          CompositeID(*v.ClientVpnEndpointId, *v.DestinationCidr, *v.GroupId),
					Name:        *v.ClientVpnEndpointId,
					Description: v,
				})
			}
		}
	}

	return values, nil
}

func EC2ClientVpnEndpoint(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeClientVpnEndpointsPaginator(client, &ec2.DescribeClientVpnEndpointsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ClientVpnEndpoints {
			values = append(values, Resource{
				ID:          *v.ClientVpnEndpointId,
				Name:        *v.ClientVpnEndpointId,
				Description: v,
			})
		}
	}

	return values, nil
}

func EC2ClientVpnRoute(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	endpoints, err := EC2ClientVpnEndpoint(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := ec2.NewFromConfig(cfg)

	var values []Resource
	for _, e := range endpoints {
		endpoint := e.Description.(types.ClientVpnEndpoint)
		paginator := ec2.NewDescribeClientVpnRoutesPaginator(client, &ec2.DescribeClientVpnRoutesInput{
			ClientVpnEndpointId: endpoint.ClientVpnEndpointId,
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.Routes {
				values = append(values, Resource{
					ID:          CompositeID(*v.ClientVpnEndpointId, *v.DestinationCidr, *v.TargetSubnet),
					Name:        *v.ClientVpnEndpointId,
					Description: v,
				})
			}
		}
	}

	return values, nil
}

func EC2ClientVpnTargetNetworkAssociation(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	endpoints, err := EC2ClientVpnEndpoint(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := ec2.NewFromConfig(cfg)

	var values []Resource
	for _, e := range endpoints {
		endpoint := e.Description.(types.ClientVpnEndpoint)
		paginator := ec2.NewDescribeClientVpnTargetNetworksPaginator(client, &ec2.DescribeClientVpnTargetNetworksInput{
			ClientVpnEndpointId: endpoint.ClientVpnEndpointId,
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.ClientVpnTargetNetworks {
				values = append(values, Resource{
					ID:          *v.AssociationId,
					Name:        *v.AssociationId,
					Description: v,
				})
			}
		}
	}

	return values, nil
}

func EC2CustomerGateway(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	output, err := client.DescribeCustomerGateways(ctx, &ec2.DescribeCustomerGatewaysInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range output.CustomerGateways {
		values = append(values, Resource{
			ID:          *v.CustomerGatewayId,
			Name:        *v.DeviceName,
			Description: v,
		})
	}

	return values, nil
}

func EC2DHCPOptions(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeDhcpOptionsPaginator(client, &ec2.DescribeDhcpOptionsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if !isErr(err, "InvalidDhcpOptionID.NotFound") {
				return nil, err
			}
			continue
		}

		for _, v := range page.DhcpOptions {
			arn := fmt.Sprintf("arn:%s:ec2:%s:%s:dhcp-options/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, *v.DhcpOptionsId)

			values = append(values, Resource{
				ARN:  arn,
				Name: *v.DhcpOptionsId,
				Description: model.EC2DhcpOptionsDescription{
					DhcpOptions: v,
				},
			})
		}
	}

	return values, nil
}

func EC2Fleet(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeFleetsPaginator(client, &ec2.DescribeFleetsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Fleets {
			arn := fmt.Sprintf("arn:%s:ec2:%s:%s:fleet/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, *v.FleetId)
			values = append(values, Resource{
				ID:   arn,
				Name: *v.FleetId,
				Description: model.EC2FleetDescription{
					Fleet: v,
				},
			})
		}
	}

	return values, nil
}

func EC2EgressOnlyInternetGateway(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeEgressOnlyInternetGatewaysPaginator(client, &ec2.DescribeEgressOnlyInternetGatewaysInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if !isErr(err, "InvalidEgressOnlyInternetGatewayId.NotFound") && !isErr(err, "InvalidEgressOnlyInternetGatewayId.Malformed") {
				return nil, err
			}
			continue
		}

		for _, v := range page.EgressOnlyInternetGateways {
			arn := fmt.Sprintf("arn:%s:ec2:%s:%s:egress-only-internet-gateway/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, *v.EgressOnlyInternetGatewayId)
			values = append(values, Resource{
				ID:   arn,
				Name: *v.EgressOnlyInternetGatewayId,
				Description: model.EC2EgressOnlyInternetGatewayDescription{
					EgressOnlyInternetGateway: v,
				},
			})
		}
	}

	return values, nil
}

func EC2EIP(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	client := ec2.NewFromConfig(cfg)
	output, err := client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{})
	if err != nil {
		if !isErr(err, "InvalidAllocationID.NotFound") && !isErr(err, "InvalidAllocationID.Malformed") {
			return nil, err
		}
		return nil, nil
	}

	var values []Resource
	for _, v := range output.Addresses {
		arn := "arn:" + describeCtx.Partition + ":ec2:" + describeCtx.Region + ":" + describeCtx.AccountID + ":eip/" + *v.AllocationId
		values = append(values, Resource{
			ARN:  arn,
			Name: *v.AllocationId,
			Description: model.EC2EIPDescription{
				Address: v,
			},
		})
	}

	return values, nil
}

func EC2EnclaveCertificateIamRoleAssociation(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	certs, err := CertificateManagerCertificate(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := ec2.NewFromConfig(cfg)

	var values []Resource
	for _, c := range certs {
		cert := c.Description.(model.CertificateManagerCertificateDescription)

		output, err := client.GetAssociatedEnclaveCertificateIamRoles(ctx, &ec2.GetAssociatedEnclaveCertificateIamRolesInput{
			CertificateArn: cert.Certificate.CertificateArn,
		})
		if err != nil {
			return nil, err
		}

		for _, v := range output.AssociatedRoles {
			values = append(values, Resource{
				ID:          *v.AssociatedRoleArn, // Don't set to ARN since that will be the same for the role itself and this association
				Name:        *v.AssociatedRoleArn,
				Description: v,
			})
		}
	}

	return values, nil
}

func EC2FlowLog(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeFlowLogsPaginator(client, &ec2.DescribeFlowLogsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.FlowLogs {
			arn := "arn:" + describeCtx.Partition + ":ec2:" + describeCtx.Region + ":" + describeCtx.AccountID + ":vpc-flow-log/" + *v.FlowLogId
			values = append(values, Resource{
				ARN:  arn,
				Name: *v.FlowLogId,
				Description: model.EC2FlowLogDescription{
					FlowLog: v,
				},
			})
		}
	}

	return values, nil
}

func EC2Host(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeHostsPaginator(client, &ec2.DescribeHostsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Hosts {
			arn := fmt.Sprintf("arn:%s:ec2:%s:%s:dedicated-host/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, *v.HostId)
			values = append(values, Resource{
				ID:   arn,
				Name: *v.HostId,
				Description: model.EC2HostDescription{
					Host: v,
				},
			})
		}
	}

	return values, nil
}

func EC2Instance(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeInstancesPaginator(client, &ec2.DescribeInstancesInput{})

	describeCtx := GetDescribeContext(ctx)

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, r := range page.Reservations {
			for _, v := range r.Instances {
				var desc model.EC2InstanceDescription

				in := v // Do this to avoid the pointer being replaced by the for loop
				desc.Instance = &in

				statusOutput, err := client.DescribeInstanceStatus(ctx, &ec2.DescribeInstanceStatusInput{
					InstanceIds:         []string{*v.InstanceId},
					IncludeAllInstances: aws.Bool(true),
				})
				if err != nil {
					return nil, err
				}
				if len(statusOutput.InstanceStatuses) > 0 {
					desc.InstanceStatus = &statusOutput.InstanceStatuses[0]
				}

				attrs := []types.InstanceAttributeName{
					types.InstanceAttributeNameUserData,
					types.InstanceAttributeNameInstanceInitiatedShutdownBehavior,
					types.InstanceAttributeNameDisableApiTermination,
				}

				for _, attr := range attrs {
					output, err := client.DescribeInstanceAttribute(ctx, &ec2.DescribeInstanceAttributeInput{
						InstanceId: v.InstanceId,
						Attribute:  attr,
					})
					if err != nil {
						return nil, err
					}

					switch attr {
					case types.InstanceAttributeNameUserData:
						desc.Attributes.UserData = aws.ToString(output.UserData.Value)
					case types.InstanceAttributeNameInstanceInitiatedShutdownBehavior:
						desc.Attributes.InstanceInitiatedShutdownBehavior = aws.ToString(output.InstanceInitiatedShutdownBehavior.Value)
					case types.InstanceAttributeNameDisableApiTermination:
						desc.Attributes.DisableApiTermination = aws.ToBool(output.DisableApiTermination.Value)
					}
				}
				arn := "arn:" + describeCtx.Partition + ":ec2:" + describeCtx.Region + ":" + describeCtx.AccountID + ":instance/" + *v.InstanceId
				values = append(values, Resource{
					ARN:         arn,
					Name:        *v.InstanceId,
					Description: desc,
				})
			}
		}
	}

	return values, nil
}

func EC2InternetGateway(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeInternetGatewaysPaginator(client, &ec2.DescribeInternetGatewaysInput{})

	describeCtx := GetDescribeContext(ctx)

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.InternetGateways {
			arn := "arn:" + describeCtx.Partition + ":ec2:" + describeCtx.Region + ":" + describeCtx.AccountID + ":internet-gateway/" + *v.InternetGatewayId
			values = append(values, Resource{
				ARN:  arn,
				Name: *v.InternetGatewayId,
				Description: model.EC2InternetGatewayDescription{
					InternetGateway: v,
				},
			})
		}
	}

	return values, nil
}

func EC2LaunchTemplate(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeLaunchTemplatesPaginator(client, &ec2.DescribeLaunchTemplatesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.LaunchTemplates {
			values = append(values, Resource{
				ID:          *v.LaunchTemplateId,
				Name:        *v.LaunchTemplateName,
				Description: v,
			})
		}
	}

	return values, nil
}

func EC2NatGateway(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeNatGatewaysPaginator(client, &ec2.DescribeNatGatewaysInput{})

	describeCtx := GetDescribeContext(ctx)

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.NatGateways {
			arn := "arn:" + describeCtx.Partition + ":ec2:" + describeCtx.Region + ":" + describeCtx.AccountID + ":natgateway/" + *v.NatGatewayId
			values = append(values, Resource{
				ARN:  arn,
				Name: *v.NatGatewayId,
				Description: model.EC2NatGatewayDescription{
					NatGateway: v,
				},
			})
		}
	}

	return values, nil
}

func EC2NetworkAcl(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeNetworkAclsPaginator(client, &ec2.DescribeNetworkAclsInput{})

	describeCtx := GetDescribeContext(ctx)

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.NetworkAcls {
			arn := "arn:" + describeCtx.Partition + ":ec2:" + describeCtx.Region + ":" + describeCtx.AccountID + ":network-acl/" + *v.NetworkAclId
			values = append(values, Resource{
				ARN:  arn,
				Name: *v.NetworkAclId,
				Description: model.EC2NetworkAclDescription{
					NetworkAcl: v,
				},
			})
		}
	}

	return values, nil
}

func EC2NetworkInsightsAnalysis(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeNetworkInsightsAnalysesPaginator(client, &ec2.DescribeNetworkInsightsAnalysesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.NetworkInsightsAnalyses {
			values = append(values, Resource{
				ARN:         *v.NetworkInsightsAnalysisArn,
				Name:        *v.NetworkInsightsAnalysisArn,
				Description: v,
			})
		}
	}

	return values, nil
}

func EC2NetworkInsightsPath(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeNetworkInsightsPathsPaginator(client, &ec2.DescribeNetworkInsightsPathsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.NetworkInsightsPaths {
			values = append(values, Resource{
				ARN:         *v.NetworkInsightsPathArn,
				Name:        *v.NetworkInsightsPathArn,
				Description: v,
			})
		}
	}

	return values, nil
}

func EC2NetworkInterface(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeNetworkInterfacesPaginator(client, &ec2.DescribeNetworkInterfacesInput{})

	describeCtx := GetDescribeContext(ctx)

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.NetworkInterfaces {
			arn := "arn:" + describeCtx.Partition + ":ec2:" + describeCtx.Region + ":" + describeCtx.AccountID + ":network-interface/" + *v.NetworkInterfaceId
			values = append(values, Resource{
				ARN:  arn,
				Name: *v.NetworkInterfaceId,
				Description: model.EC2NetworkInterfaceDescription{
					NetworkInterface: v,
				},
			})
		}
	}

	return values, nil
}

func EC2NetworkInterfacePermission(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeNetworkInterfacePermissionsPaginator(client, &ec2.DescribeNetworkInterfacePermissionsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.NetworkInterfacePermissions {
			values = append(values, Resource{
				ID:          *v.NetworkInterfacePermissionId,
				Name:        *v.NetworkInterfacePermissionId,
				Description: v,
			})
		}
	}

	return values, nil
}

func EC2PlacementGroup(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	client := ec2.NewFromConfig(cfg)
	output, err := client.DescribePlacementGroups(ctx, &ec2.DescribePlacementGroupsInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range output.PlacementGroups {
		arn := fmt.Sprintf("arn:%s:ec2:%s:%s:placement-group/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, *v.GroupName)
		values = append(values, Resource{
			ID:   arn,
			Name: *v.GroupName,
			Description: model.EC2PlacementGroupDescription{
				PlacementGroup: v,
			},
		})
	}

	return values, nil
}

func EC2PrefixList(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribePrefixListsPaginator(client, &ec2.DescribePrefixListsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.PrefixLists {
			values = append(values, Resource{
				ID:          *v.PrefixListId,
				Name:        *v.PrefixListName,
				Description: v,
			})
		}
	}

	return values, nil
}

func EC2RegionalSettings(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	out, err := client.GetEbsEncryptionByDefault(ctx, &ec2.GetEbsEncryptionByDefaultInput{})
	if err != nil {
		return nil, err
	}

	outkey, err := client.GetEbsDefaultKmsKeyId(ctx, &ec2.GetEbsDefaultKmsKeyIdInput{})
	if err != nil {
		return nil, err
	}

	return []Resource{
		{
			// No ID or ARN. Per Account Configuration
			Name: cfg.Region + " EC2 Settings", // Based on Steampipe
			Description: model.EC2RegionalSettingsDescription{
				EbsEncryptionByDefault: out.EbsEncryptionByDefault,
				KmsKeyId:               outkey.KmsKeyId,
			},
		},
	}, nil
}

func EC2RouteTable(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeRouteTablesPaginator(client, &ec2.DescribeRouteTablesInput{})

	describeCtx := GetDescribeContext(ctx)

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.RouteTables {
			arn := "arn:" + describeCtx.Partition + ":ec2:" + describeCtx.Region + ":" + describeCtx.AccountID + ":route-table/" + *v.RouteTableId

			values = append(values, Resource{
				ARN:  arn,
				Name: *v.RouteTableId,
				Description: model.EC2RouteTableDescription{
					RouteTable: v,
				},
			})
		}
	}

	return values, nil
}

func EC2LocalGatewayRouteTable(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeLocalGatewayRouteTablesPaginator(client, &ec2.DescribeLocalGatewayRouteTablesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.LocalGatewayRouteTables {
			values = append(values, Resource{
				ARN:         *v.LocalGatewayRouteTableArn,
				Name:        *v.LocalGatewayRouteTableId,
				Description: v,
			})
		}
	}

	return values, nil
}

func EC2LocalGatewayRouteTableVPCAssociation(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeLocalGatewayRouteTableVpcAssociationsPaginator(client, &ec2.DescribeLocalGatewayRouteTableVpcAssociationsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.LocalGatewayRouteTableVpcAssociations {
			values = append(values, Resource{
				ID:          *v.LocalGatewayRouteTableVpcAssociationId,
				Name:        *v.LocalGatewayRouteTableVpcAssociationId,
				Description: v,
			})
		}
	}

	return values, nil
}

func EC2TransitGatewayRouteTable(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeTransitGatewayRouteTablesPaginator(client, &ec2.DescribeTransitGatewayRouteTablesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if !isErr(err, "InvalidRouteTableID.NotFound") && !isErr(err, "InvalidRouteTableId.Unavailable") && !isErr(err, "InvalidRouteTableId.Malformed") {
				return nil, err
			}
			continue
		}

		for _, v := range page.TransitGatewayRouteTables {
			arn := fmt.Sprintf("arn:%s:ec2:%s:%s:transit-gateway-route-table/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, *v.TransitGatewayRouteTableId)

			values = append(values, Resource{
				ARN:  arn,
				Name: *v.TransitGatewayRouteTableId,
				Description: model.EC2TransitGatewayRouteTableDescription{
					TransitGatewayRouteTable: v,
				},
			})
		}
	}

	return values, nil
}

func EC2TransitGatewayRouteTableAssociation(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	rts, err := EC2TransitGatewayRouteTable(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := ec2.NewFromConfig(cfg)

	var values []Resource
	for _, r := range rts {
		routeTable := r.Description.(types.TransitGatewayRouteTable)
		paginator := ec2.NewGetTransitGatewayRouteTableAssociationsPaginator(client, &ec2.GetTransitGatewayRouteTableAssociationsInput{
			TransitGatewayRouteTableId: routeTable.TransitGatewayRouteTableId,
		})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.Associations {
				values = append(values, Resource{
					ID:          *v.TransitGatewayAttachmentId,
					Name:        *v.TransitGatewayAttachmentId,
					Description: v,
				})
			}
		}
	}

	return values, nil
}

func EC2TransitGatewayRouteTablePropagation(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	rts, err := EC2TransitGatewayRouteTable(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := ec2.NewFromConfig(cfg)

	var values []Resource
	for _, r := range rts {
		routeTable := r.Description.(types.TransitGatewayRouteTable)
		paginator := ec2.NewGetTransitGatewayRouteTablePropagationsPaginator(client, &ec2.GetTransitGatewayRouteTablePropagationsInput{
			TransitGatewayRouteTableId: routeTable.TransitGatewayRouteTableId,
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.TransitGatewayRouteTablePropagations {
				values = append(values, Resource{
					ID:          CompositeID(*routeTable.TransitGatewayRouteTableId, *v.TransitGatewayAttachmentId),
					Name:        *routeTable.TransitGatewayRouteTableId,
					Description: v,
				})
			}
		}
	}

	return values, nil
}

func EC2SecurityGroup(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeSecurityGroupsPaginator(client, &ec2.DescribeSecurityGroupsInput{})

	describeCtx := GetDescribeContext(ctx)

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.SecurityGroups {
			arn := "arn:" + describeCtx.Partition + ":ec2:" + describeCtx.Region + ":" + describeCtx.AccountID + ":security-group/" + *v.GroupId
			values = append(values, Resource{
				ARN:  arn,
				Name: *v.GroupName,
				Description: model.EC2SecurityGroupDescription{
					SecurityGroup: v,
				},
			})
		}
	}

	return values, nil
}

func getEC2SecurityGroupRuleDescriptionFromIPPermission(group types.SecurityGroup, permission types.IpPermission, groupType string) []model.EC2SecurityGroupRuleDescription {
	var descArr []model.EC2SecurityGroupRuleDescription

	// create 1 row per ip-range
	if permission.IpRanges != nil {
		for _, r := range permission.IpRanges {
			descArr = append(descArr, model.EC2SecurityGroupRuleDescription{
				Group:           group,
				Permission:      permission,
				IPRange:         &r,
				Ipv6Range:       nil,
				UserIDGroupPair: nil,
				PrefixListId:    nil,
				Type:            groupType,
			})
		}
	}

	// create 1 row per prefix-list Id
	if permission.PrefixListIds != nil {
		for _, r := range permission.PrefixListIds {
			descArr = append(descArr, model.EC2SecurityGroupRuleDescription{
				Group:           group,
				Permission:      permission,
				IPRange:         nil,
				Ipv6Range:       nil,
				UserIDGroupPair: nil,
				PrefixListId:    &r,
				Type:            groupType,
			})
		}
	}

	// create 1 row per ipv6-range
	if permission.Ipv6Ranges != nil {
		for _, r := range permission.Ipv6Ranges {
			descArr = append(descArr, model.EC2SecurityGroupRuleDescription{
				Group:           group,
				Permission:      permission,
				IPRange:         nil,
				Ipv6Range:       &r,
				UserIDGroupPair: nil,
				PrefixListId:    nil,
				Type:            groupType,
			})
		}
	}

	// create 1 row per user id group pair
	if permission.UserIdGroupPairs != nil {
		for _, r := range permission.UserIdGroupPairs {
			descArr = append(descArr, model.EC2SecurityGroupRuleDescription{
				Group:           group,
				Permission:      permission,
				IPRange:         nil,
				Ipv6Range:       nil,
				UserIDGroupPair: &r,
				PrefixListId:    nil,
				Type:            groupType,
			})
		}
	}

	return descArr
}

func EC2SecurityGroupRule(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	groups, err := EC2SecurityGroup(ctx, cfg)
	if err != nil {
		return nil, err
	}

	var values []Resource
	descArr := make([]model.EC2SecurityGroupRuleDescription, 0, 128)
	for _, groupWrapper := range groups {
		group := groupWrapper.Description.(model.EC2SecurityGroupDescription).SecurityGroup
		if group.IpPermissions != nil {
			for _, permission := range group.IpPermissions {
				descArr = append(descArr, getEC2SecurityGroupRuleDescriptionFromIPPermission(group, permission, "ingress")...)
			}
		}
		if group.IpPermissionsEgress != nil {
			for _, permission := range group.IpPermissionsEgress {
				descArr = append(descArr, getEC2SecurityGroupRuleDescriptionFromIPPermission(group, permission, "egress")...)
			}
		}
	}
	for _, desc := range descArr {
		hashCode := desc.Type + "_" + *desc.Permission.IpProtocol
		if desc.Permission.FromPort != nil {
			hashCode = hashCode + "_" + fmt.Sprint(desc.Permission.FromPort) + "_" + fmt.Sprint(desc.Permission.ToPort)
		}

		if desc.IPRange != nil && desc.IPRange.CidrIp != nil {
			hashCode = hashCode + "_" + *desc.IPRange.CidrIp
		} else if desc.Ipv6Range != nil && desc.Ipv6Range.CidrIpv6 != nil {
			hashCode = hashCode + "_" + *desc.Ipv6Range.CidrIpv6
		} else if desc.UserIDGroupPair != nil && *desc.UserIDGroupPair.GroupId == *desc.Group.GroupId {
			hashCode = hashCode + "_" + *desc.Group.GroupId
		} else if desc.PrefixListId != nil && desc.PrefixListId.PrefixListId != nil {
			hashCode = hashCode + "_" + *desc.PrefixListId.PrefixListId
		}

		arn := fmt.Sprintf("arn:%s:ec2:%s:%s:security-group/%s:%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, *desc.Group.GroupId, hashCode)
		values = append(values, Resource{
			ARN:         arn,
			Name:        fmt.Sprintf("%s_%s", *desc.Group.GroupId, hashCode),
			Description: desc,
		})
	}

	return values, nil
}

func EC2SpotFleet(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeSpotFleetRequestsPaginator(client, &ec2.DescribeSpotFleetRequestsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.SpotFleetRequestConfigs {
			values = append(values, Resource{
				ID:          *v.SpotFleetRequestId,
				Name:        *v.SpotFleetRequestId,
				Description: v,
			})
		}
	}

	return values, nil
}

func EC2Subnet(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeSubnetsPaginator(client, &ec2.DescribeSubnetsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Subnets {
			values = append(values, Resource{
				ARN:  *v.SubnetArn,
				Name: *v.SubnetId,
				Description: model.EC2SubnetDescription{
					Subnet: v,
				},
			})
		}
	}

	return values, nil
}

func EC2TrafficMirrorFilter(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeTrafficMirrorFiltersPaginator(client, &ec2.DescribeTrafficMirrorFiltersInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.TrafficMirrorFilters {
			values = append(values, Resource{
				ID:          *v.TrafficMirrorFilterId,
				Name:        *v.TrafficMirrorFilterId,
				Description: v,
			})
		}
	}

	return values, nil
}

func EC2TrafficMirrorSession(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeTrafficMirrorSessionsPaginator(client, &ec2.DescribeTrafficMirrorSessionsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.TrafficMirrorSessions {
			values = append(values, Resource{
				ID:          *v.TrafficMirrorSessionId,
				Name:        *v.TrafficMirrorFilterId,
				Description: v,
			})
		}
	}

	return values, nil
}

func EC2TrafficMirrorTarget(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeTrafficMirrorTargetsPaginator(client, &ec2.DescribeTrafficMirrorTargetsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.TrafficMirrorTargets {
			values = append(values, Resource{
				ID:          *v.TrafficMirrorTargetId,
				Name:        *v.TrafficMirrorTargetId,
				Description: v,
			})
		}
	}

	return values, nil
}

func EC2TransitGateway(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeTransitGatewaysPaginator(client, &ec2.DescribeTransitGatewaysInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if !isErr(err, "InvalidTransitGatewayID.NotFound") && !isErr(err, "InvalidTransitGatewayID.Unavailable") && !isErr(err, "InvalidTransitGatewayID.Malformed") {
				return nil, err
			}
			continue
		}

		for _, v := range page.TransitGateways {
			values = append(values, Resource{
				ARN:  *v.TransitGatewayArn,
				Name: *v.TransitGatewayId,
				Description: model.EC2TransitGatewayDescription{
					TransitGateway: v,
				},
			})
		}
	}

	return values, nil
}

func EC2TransitGatewayAttachment(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeTransitGatewayAttachmentsPaginator(client, &ec2.DescribeTransitGatewayAttachmentsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.TransitGatewayAttachments {
			values = append(values, Resource{
				ID:          *v.TransitGatewayAttachmentId,
				Name:        *v.TransitGatewayAttachmentId,
				Description: v,
			})
		}
	}

	return values, nil
}

func EC2TransitGatewayConnect(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeTransitGatewayConnectsPaginator(client, &ec2.DescribeTransitGatewayConnectsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.TransitGatewayConnects {
			values = append(values, Resource{
				ID:          *v.TransitGatewayAttachmentId,
				Name:        *v.TransitGatewayAttachmentId,
				Description: v,
			})
		}
	}

	return values, nil
}

func EC2TransitGatewayMulticastDomain(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeTransitGatewayMulticastDomainsPaginator(client, &ec2.DescribeTransitGatewayMulticastDomainsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.TransitGatewayMulticastDomains {
			values = append(values, Resource{
				ARN:         *v.TransitGatewayMulticastDomainArn,
				Name:        *v.TransitGatewayMulticastDomainArn,
				Description: v,
			})
		}
	}

	return values, nil
}

func EC2TransitGatewayMulticastDomainAssociation(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	domains, err := EC2TransitGatewayMulticastDomain(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := ec2.NewFromConfig(cfg)
	var values []Resource
	for _, domain := range domains {
		paginator := ec2.NewGetTransitGatewayMulticastDomainAssociationsPaginator(client, &ec2.GetTransitGatewayMulticastDomainAssociationsInput{
			TransitGatewayMulticastDomainId: domain.Description.(types.TransitGatewayMulticastDomain).TransitGatewayMulticastDomainId,
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.MulticastDomainAssociations {
				values = append(values, Resource{
					ID:          *v.TransitGatewayAttachmentId,
					Name:        *v.TransitGatewayAttachmentId,
					Description: v,
				})
			}
		}
	}

	return values, nil
}

func EC2TransitGatewayMulticastGroupMember(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	domains, err := EC2TransitGatewayMulticastDomain(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := ec2.NewFromConfig(cfg)
	var values []Resource
	for _, domain := range domains {
		tgmdID := domain.Description.(types.TransitGatewayMulticastDomain).TransitGatewayMulticastDomainId
		paginator := ec2.NewSearchTransitGatewayMulticastGroupsPaginator(client, &ec2.SearchTransitGatewayMulticastGroupsInput{
			TransitGatewayMulticastDomainId: tgmdID,
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
				values = append(values, Resource{
					ID:          CompositeID(*tgmdID, *v.GroupIpAddress),
					Name:        *v.GroupIpAddress,
					Description: v,
				})
			}
		}
	}

	return values, nil
}

func EC2TransitGatewayMulticastGroupSource(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	domains, err := EC2TransitGatewayMulticastDomain(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := ec2.NewFromConfig(cfg)
	var values []Resource
	for _, domain := range domains {
		tgmdID := domain.Description.(types.TransitGatewayMulticastDomain).TransitGatewayMulticastDomainId
		paginator := ec2.NewSearchTransitGatewayMulticastGroupsPaginator(client, &ec2.SearchTransitGatewayMulticastGroupsInput{
			TransitGatewayMulticastDomainId: tgmdID,
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
				values = append(values, Resource{
					ID:          CompositeID(*tgmdID, *v.GroupIpAddress),
					Name:        *v.GroupIpAddress,
					Description: v,
				})
			}
		}
	}

	return values, nil
}

func EC2TransitGatewayPeeringAttachment(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeTransitGatewayPeeringAttachmentsPaginator(client, &ec2.DescribeTransitGatewayPeeringAttachmentsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.TransitGatewayPeeringAttachments {
			values = append(values, Resource{
				ID:          *v.TransitGatewayAttachmentId,
				Name:        *v.TransitGatewayAttachmentId,
				Description: v,
			})
		}
	}

	return values, nil
}

func EC2VPC(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeVpcsPaginator(client, &ec2.DescribeVpcsInput{})

	describeCtx := GetDescribeContext(ctx)

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Vpcs {
			arn := "arn:" + describeCtx.Partition + ":ec2:" + describeCtx.Region + ":" + describeCtx.AccountID + ":vpc/" + *v.VpcId
			values = append(values, Resource{
				ARN:  arn,
				Name: *v.VpcId,
				Description: model.EC2VpcDescription{
					Vpc: v,
				},
			})
		}
	}

	return values, nil
}

func EC2VPCEndpoint(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeVpcEndpointsPaginator(client, &ec2.DescribeVpcEndpointsInput{})

	describeCtx := GetDescribeContext(ctx)

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.VpcEndpoints {
			arn := "arn:" + describeCtx.Partition + ":ec2:" + describeCtx.Region + ":" + describeCtx.AccountID + ":vpc-endpoint/" + *v.VpcEndpointId
			values = append(values, Resource{
				ARN:  arn,
				ID:   *v.VpcEndpointId,
				Name: *v.VpcEndpointId,
				Description: model.EC2VPCEndpointDescription{
					VpcEndpoint: v,
				},
			})
		}
	}

	return values, nil
}

func EC2VPCEndpointConnectionNotification(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeVpcEndpointConnectionNotificationsPaginator(client, &ec2.DescribeVpcEndpointConnectionNotificationsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ConnectionNotificationSet {
			values = append(values, Resource{
				ARN:         *v.ConnectionNotificationArn,
				Name:        *v.ConnectionNotificationArn,
				Description: v,
			})
		}
	}

	return values, nil
}

func EC2VPCEndpointService(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	client := ec2.NewFromConfig(cfg)

	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.DescribeVpcEndpointServices(ctx, &ec2.DescribeVpcEndpointServicesInput{
			NextToken: prevToken,
		})
		if err != nil {
			return nil, err
		}

		for _, v := range output.ServiceDetails {
			splitServiceName := strings.Split(*v.ServiceName, ".")
			arn := fmt.Sprintf("arn:%s:ec2:%s:%s:vpc-endpoint-service/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, splitServiceName[len(splitServiceName)-1])

			values = append(values, Resource{
				ARN:  arn,
				Name: *v.ServiceName,
				Description: model.EC2VPCEndpointServiceDescription{
					VpcEndpointService: v,
				},
			})
		}

		return output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}

func EC2VPCEndpointServicePermissions(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	services, err := EC2VPCEndpointService(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := ec2.NewFromConfig(cfg)

	var values []Resource
	for _, s := range services {
		service := s.Description.(model.EC2VPCEndpointServiceDescription).VpcEndpointService

		paginator := ec2.NewDescribeVpcEndpointServicePermissionsPaginator(client, &ec2.DescribeVpcEndpointServicePermissionsInput{
			ServiceId: service.ServiceId,
		})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				var ae smithy.APIError
				if errors.As(err, &ae) && ae.ErrorCode() == "InvalidVpcEndpointServiceId.NotFound" {
					// VpcEndpoint doesn't have permissions set. Move on!
					break
				}
				return nil, err
			}

			for _, v := range page.AllowedPrincipals {
				values = append(values, Resource{
					ARN:         *v.Principal,
					Name:        *v.Principal,
					Description: v,
				})
			}
		}
	}

	return values, nil
}

func EC2VPCPeeringConnection(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeVpcPeeringConnectionsPaginator(client, &ec2.DescribeVpcPeeringConnectionsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.VpcPeeringConnections {
			arn := fmt.Sprintf("arn:%s:ec2:%s:%s:vpc-peering-connection/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, *v.VpcPeeringConnectionId)
			values = append(values, Resource{
				ARN:  arn,
				Name: *v.VpcPeeringConnectionId,
				Description: model.EC2VpcPeeringConnectionDescription{
					VpcPeeringConnection: v,
				},
			})
		}
	}

	return values, nil
}

func EC2VPNConnection(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	output, err := client.DescribeVpnConnections(ctx, &ec2.DescribeVpnConnectionsInput{})
	if err != nil {
		return nil, err
	}

	describeCtx := GetDescribeContext(ctx)

	var values []Resource
	for _, v := range output.VpnConnections {
		arn := "arn:" + describeCtx.Partition + ":ec2:" + describeCtx.Region + ":" + describeCtx.AccountID + ":vpn-connection/" + *v.VpnConnectionId
		values = append(values, Resource{
			ARN:  arn,
			Name: *v.VpnConnectionId,
			Description: model.EC2VPNConnectionDescription{
				VpnConnection: v,
			},
		})
	}

	return values, nil
}

func EC2VPNGateway(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	output, err := client.DescribeVpnGateways(ctx, &ec2.DescribeVpnGatewaysInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range output.VpnGateways {
		values = append(values, Resource{
			ID:          *v.VpnGatewayId,
			Name:        *v.VpnGatewayId,
			Description: v,
		})
	}

	return values, nil
}

func EC2Region(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	output, err := client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{
		AllRegions: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	describeCtx := GetDescribeContext(ctx)

	var values []Resource
	for _, v := range output.Regions {
		arn := "arn:" + describeCtx.Partition + "::" + *v.RegionName + ":" + describeCtx.AccountID
		values = append(values, Resource{
			ARN:  arn,
			Name: *v.RegionName,
			Description: model.EC2RegionDescription{
				Region: v,
			},
		})
	}

	return values, nil
}

func EC2KeyPair(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	client := ec2.NewFromConfig(cfg)
	output, err := client.DescribeKeyPairs(ctx, &ec2.DescribeKeyPairsInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range output.KeyPairs {
		arn := "arn:" + describeCtx.Partition + ":ec2:" + describeCtx.Region + ":" + describeCtx.AccountID + ":key-pair/" + *v.KeyName
		values = append(values, Resource{
			ARN:  arn,
			Name: *v.KeyName,
			Description: model.EC2KeyPairDescription{
				KeyPair: v,
			},
		})
	}

	return values, nil
}

func EC2AMI(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	client := ec2.NewFromConfig(cfg)
	output, err := client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Owners: []string{"self"},
	})
	if err != nil {
		if isErr(err, "InvalidAMIID.NotFound") || isErr(err, "InvalidAMIID.Unavailable") || isErr(err, "InvalidAMIID.Malformed") {
			return nil, nil
		}
		return nil, err
	}

	var values []Resource

	for _, v := range output.Images {
		imageAttribute, err := client.DescribeImageAttribute(ctx, &ec2.DescribeImageAttributeInput{
			Attribute: types.ImageAttributeNameLaunchPermission,
			ImageId:   v.ImageId,
		})
		if err != nil {
			if isErr(err, "InvalidAMIID.NotFound") || isErr(err, "InvalidAMIID.Unavailable") || isErr(err, "InvalidAMIID.Malformed") {
				continue
			}
			return nil, err
		}

		arn := "arn:" + describeCtx.Partition + ":ec2:" + describeCtx.Region + ":" + describeCtx.AccountID + ":image/" + *v.ImageId
		values = append(values, Resource{
			ARN:  arn,
			Name: *v.ImageId,
			Description: model.EC2AMIDescription{
				AMI:               v,
				LaunchPermissions: *imageAttribute,
			},
		})
	}

	return values, nil
}

func EC2ReservedInstances(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	client := ec2.NewFromConfig(cfg)
	output, err := client.DescribeReservedInstances(ctx, &ec2.DescribeReservedInstancesInput{})
	if err != nil {
		if isErr(err, "InvalidParameterValue") || isErr(err, "InvalidInstanceID.Unavailable") || isErr(err, "InvalidInstanceID.Malformed") {
			return nil, nil
		}
		return nil, err
	}

	var values []Resource

	filterName := "reserved-instances-id"
	for _, v := range output.ReservedInstances {
		var modifications []types.ReservedInstancesModification
		modificationPaginator := ec2.NewDescribeReservedInstancesModificationsPaginator(client, &ec2.DescribeReservedInstancesModificationsInput{
			Filters: []types.Filter{
				{
					Name:   &filterName,
					Values: []string{*v.ReservedInstancesId},
				},
			},
		})
		for modificationPaginator.HasMorePages() {
			page, err := modificationPaginator.NextPage(ctx)
			if err != nil {
				if isErr(err, "InvalidParameterValue") || isErr(err, "InvalidInstanceID.Unavailable") || isErr(err, "InvalidInstanceID.Malformed") {
					continue
				}
				return nil, err
			}

			modifications = append(modifications, page.ReservedInstancesModifications...)
		}

		arn := "arn:" + describeCtx.Partition + ":ec2:" + describeCtx.Region + ":" + describeCtx.AccountID + ":reserved-instances/" + *v.ReservedInstancesId
		values = append(values, Resource{
			ARN:  arn,
			Name: *v.ReservedInstancesId,
			Description: model.EC2ReservedInstancesDescription{
				ReservedInstances:   v,
				ModificationDetails: modifications,
			},
		})
	}

	return values, nil
}

func EC2IpamPool(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeIpamPoolsPaginator(client, &ec2.DescribeIpamPoolsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.IpamPools {
			values = append(values, Resource{
				ARN:  *v.IpamPoolArn,
				Name: *v.IpamPoolId,
				Description: model.EC2IpamPoolDescription{
					IpamPool: v,
				},
			})
		}
	}

	return values, nil
}

func EC2Ipam(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeIpamsPaginator(client, &ec2.DescribeIpamsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Ipams {
			values = append(values, Resource{
				ARN:  *v.IpamArn,
				Name: *v.IpamId,
				Description: model.EC2IpamDescription{
					Ipam: v,
				},
			})
		}
	}

	return values, nil
}
