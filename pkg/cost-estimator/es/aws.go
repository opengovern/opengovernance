package es

import (
	"github.com/kaytu-io/kaytu-util/pkg/es"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"golang.org/x/net/context"
)

func GetEC2Instance(client kaytu.Client, resourceId string) (EC2InstanceResponse, error) {
	index := es.ResourceTypeToESIndex("AWS::EC2::Instance")
	queryBytes, err := GetResourceQuery(resourceId)
	if err != nil {
		return EC2InstanceResponse{}, err
	}
	var resp EC2InstanceResponse
	err = client.Search(context.Background(), index, string(queryBytes), &resp)
	if err != nil {
		return EC2InstanceResponse{}, err
	}
	return resp, nil
}
