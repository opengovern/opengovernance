package aws

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/smithy-go"
)

const (
	pagingSize         = 1000
	routeTablePageSize = 100
)

type ResourceDescriber func(context.Context, aws.Config) ([]interface{}, error)

var ResourceTypeToDescriber = map[string]ResourceDescriber{
	"AWS::EC2::Instance":                 getEC2Instances,
	"AWS::EC2::Route":                    getEC2Routes, // Doesn't really make sense by itself (Already exists in RouteTable)
	"AWS::EC2::NatGateway":               getEC2NatGateways,
	"AWS::EC2::RouteTable":               getEC2RouteTables,
	"AWS::EC2::SecurityGroup":            getEC2SecurityGroups,
	"AWS::EC2::Subnet":                   getEC2Subnets,
	"AWS::EC2::TransitGateway":           getEC2TransitGateways,
	"AWS::EC2::TransitGatewayAttachment": getEC2TransitGatewayAttachments,
	"AWS::EC2::TransitGatewayConnect":    getEC2TransitGatewayConnets,
	"AWS::EC2::Volume":                   getEC2Volumes,
	"AWS::EC2::VolumeAttachment":         getEC2VolumeAttachments, // Doesn't really make sense by itself (Already exists in Volume)
	"AWS::EC2::VPC":                      getEC2Vpcs,
	"AWS::EC2::VPCPeeringConnection":     getEC2VpcPeeringConnections,
}

type RegionalResponse struct {
	Resources map[string][]interface{}
	Errors    map[string]string
}

func GetResources(
	ctx context.Context,
	cfg aws.Config,
	regions []string,
	resourceType string) (*RegionalResponse, error) {

	type result struct {
		region    string
		resources []interface{}
		err       error
	}

	describe, ok := ResourceTypeToDescriber[resourceType]
	if !ok {
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	input := make(chan result, len(regions))
	for _, region := range regions {
		go func(r string) {
			// Make a shallow copy and override the default region
			rCfg := cfg.Copy()
			rCfg.Region = r

			resources, err := describe(ctx, rCfg)
			input <- result{region: r, resources: resources, err: err}
		}(region)
	}

	response := RegionalResponse{
		Resources: make(map[string][]interface{}, len(regions)),
		Errors:    make(map[string]string, len(regions)),
	}
	for range regions {
		resp := <-input
		if resp.err != nil {
			// If an action is not supported in a region, we will get InvalidAction error code. In that case,
			// just send empty list as the response. Since we are using the AWS SDK, if we hit an InvalidAction
			// we can be certain that the API operation is not supported in that particular region.
			var ae smithy.APIError
			if errors.As(resp.err, &ae) && ae.ErrorCode() == "InvalidAction" {
				resp.resources, resp.err = []interface{}{}, nil
			} else {
				response.Errors[resp.region] = resp.err.Error()
				continue
			}
		}

		if resp.resources == nil {
			resp.resources = []interface{}{}
		}
		response.Resources[resp.region] = resp.resources
	}

	return &response, nil
}
