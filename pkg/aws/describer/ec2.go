package describer

import (
	"context"
	"errors"

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
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeDhcpOptionsPaginator(client, &ec2.DescribeDhcpOptionsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.DhcpOptions {
			values = append(values, Resource{
				ID:          *v.DhcpOptionsId,
				Name:        *v.DhcpOptionsId,
				Description: v,
			})
		}
	}

	return values, nil
}

func EC2Fleet(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeFleetsPaginator(client, &ec2.DescribeFleetsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Fleets {
			values = append(values, Resource{
				ID:          *v.FleetId,
				Name:        *v.FleetId,
				Description: v,
			})
		}
	}

	return values, nil
}

func EC2EgressOnlyInternetGateway(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeEgressOnlyInternetGatewaysPaginator(client, &ec2.DescribeEgressOnlyInternetGatewaysInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.EgressOnlyInternetGateways {
			values = append(values, Resource{
				ID:          *v.EgressOnlyInternetGatewayId,
				Name:        *v.EgressOnlyInternetGatewayId,
				Description: v,
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
		return nil, err
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
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeHostsPaginator(client, &ec2.DescribeHostsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Hosts {
			values = append(values, Resource{
				ID:          *v.HostId,
				Name:        *v.HostId,
				Description: v,
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
	client := ec2.NewFromConfig(cfg)
	output, err := client.DescribePlacementGroups(ctx, &ec2.DescribePlacementGroupsInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range output.PlacementGroups {
		values = append(values, Resource{
			ID:          *v.GroupId,
			Name:        *v.GroupName,
			Description: v,
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
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeTransitGatewayRouteTablesPaginator(client, &ec2.DescribeTransitGatewayRouteTablesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.TransitGatewayRouteTables {
			values = append(values, Resource{
				ID:          *v.TransitGatewayRouteTableId,
				Name:        *v.TransitGatewayRouteTableId,
				Description: v,
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
			return nil, err
		}

		for _, v := range page.TransitGateways {
			values = append(values, Resource{
				ARN:         *v.TransitGatewayArn,
				Name:        *v.TransitGatewayArn,
				Description: v,
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
			values = append(values, Resource{
				ID:          *v.ServiceId,
				Name:        *v.ServiceName,
				Description: v,
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
		service := s.Description.(types.ServiceDetail)

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
	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeVpcPeeringConnectionsPaginator(client, &ec2.DescribeVpcPeeringConnectionsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.VpcPeeringConnections {
			values = append(values, Resource{
				ID:          *v.VpcPeeringConnectionId,
				Name:        *v.VpcPeeringConnectionId,
				Description: v,
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
