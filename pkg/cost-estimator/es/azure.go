package es

import (
	"github.com/kaytu-io/kaytu-util/pkg/es"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"golang.org/x/net/context"
)

func GetMicrosoftVirtualMachine(client kaytu.Client, resourceId string) (MicrosoftVirtualMachineResponse, error) {
	index := es.ResourceTypeToESIndex("Microsoft.Compute/virtualMachines")
	queryBytes, err := GetResourceQuery(resourceId)
	if err != nil {
		return MicrosoftVirtualMachineResponse{}, err
	}
	var resp MicrosoftVirtualMachineResponse
	err = client.Search(context.Background(), index, string(queryBytes), &resp)
	if err != nil {
		return MicrosoftVirtualMachineResponse{}, err
	}
	return resp, nil
}
