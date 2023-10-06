package es

var awsHandledResourceTypes = []string{"AWS::CertificateManager::Certificate", "AWS::Athena::QueryExecution",
	"AWS::IAM::Policy", "AWS::ECR::Repository", "AWS::ECS::Service", "AWS::CloudFormation::Stack", "AWS::EC2::Instance",
	"AWS::AccessAnalyzer::Analyzer", "AWS::Glue::CatalogTable", "AWS::RDS::DBClusterParameterGroup", "AWS::WAFv2::IPSet",
	"AWS::EC2::RouteTable", "AWS::WAFv2::RuleGroup", "AWS::WAFv2::WebACL", "AWS::S3::Bucket",
	"AWS::RDS::DBParameterGroup", "AWS::ECS::TaskDefinition"}

var azureHandledResourceTypes = []string{"Microsoft.Network/networkSecurityGroups", "Microsoft.Web/sites",
	"Microsoft.Network/virtualNetworks/subnets", "Microsoft.Network/frontDoors", "Microsoft.Network/loadBalancers",
	"Microsoft.Network/virtualNetworks", "Microsoft.Network/routeTables", "Microsoft.DocumentDB/SqlDatabases",
	"Microsoft.Network/applicationGateways", "Microsoft.LoadBalancer/backendAddressPools", "Microsoft.KeyVault/vaults",
	"Microsoft.DataFactory/factoriesDatasets", "Microsoft.Authorization/roleDefinitions", "Microsoft.Logic/workflows",
	"Microsoft.Compute/virtualMachines", "Microsoft.DBforMySQL/servers", "Microsoft.DBforPostgreSQL/servers",
	"Microsoft.Compute/virtualMachineScaleSets", "Microsoft.Sql/servers", "Microsoft.Web/plan",
	"Microsoft.DataFactory/factoriesPipelines", "Microsoft.Compute/virtualMachineScaleSets/networkInterfaces"}

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
