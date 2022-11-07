package cloudservice

import (
	"strings"
)

type AWSARN struct {
	Partition    string
	Service      string
	Region       string
	AccountID    string
	ResourceType string
	ResourceID   string
}

func ParseARN(arn string) AWSARN {
	arn = strings.ToLower(arn)
	arr := strings.Split(arn, ":")
	// aws::service::resourceType
	if len(arr) == 5 && arr[0] == "aws" {
		return AWSARN{
			Partition:    "",
			Service:      arr[2],
			Region:       "",
			AccountID:    "",
			ResourceType: arr[4],
			ResourceID:   "",
		}
	}

	if len(arr) == 6 {
		//arn:partition:service:region:account-id:resource-id
		//arn:partition:service:region:account-id:resource-type/resource-id
		resourceType := ""
		resourceId := arr[5]
		if ar := strings.Split(arr[5], "/"); len(ar) == 2 {
			resourceType = ar[0]
			resourceId = ar[1]
		}
		return AWSARN{
			Partition:    arr[1],
			Service:      arr[2],
			Region:       arr[3],
			AccountID:    arr[4],
			ResourceType: resourceType,
			ResourceID:   resourceId,
		}
	} else if len(arr) == 7 {
		//arn:partition:service:region:account-id:resource-type:resource-id
		return AWSARN{
			Partition:    arr[1],
			Service:      arr[2],
			Region:       arr[3],
			AccountID:    arr[4],
			ResourceType: arr[5],
			ResourceID:   arr[6],
		}
	} else {
		return AWSARN{}
	}
}

func (a AWSARN) Type() string {
	//service-provider::service-name::data-type-name
	return strings.ToLower("aws::" + a.Service + "::" + a.ResourceType)
}
