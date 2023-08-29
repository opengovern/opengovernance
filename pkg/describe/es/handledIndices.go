package es

var awsHandledResourceTypes = []string{"AWS::CertificateManager::Certificate", "AWS::Athena::QueryExecution",
	"AWS::IAM::Policy", "AWS::ECR::Repository", "AWS::ECS::Service", "AWS::CloudFormation::Stack", "AWS::EC2::Instance"}

var azureHandledResourceTypes = []string{"Microsoft.Network/networkSecurityGroups", "Microsoft.Web/sites",
	"Microsoft.Network/virtualNetworks/subnets"}

func IsHandledAWSResourceType(resourceType string) bool {
	for _, r := range awsHandledResourceTypes {
		if resourceType == r {
			return true
		}
	}
	return false
}

func IsHandledAzureResourceType(resourceType string) bool {
	for _, r := range azureHandledResourceTypes {
		if resourceType == r {
			return true
		}
	}
	return false
}
