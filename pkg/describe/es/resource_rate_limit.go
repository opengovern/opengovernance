package es

var ResourceRateLimit = map[string]int{
	"Microsoft.Management/groups":                 1,
	"Microsoft.CostManagement/CostByResourceType": 1,
	"AWS::Organizations::Account":                 3,
	"AWS::Shield::ProtectionGroup":                3,
	"AWS::IAM::Policy":                            3,
	"AWS::IAM::Role":                              3,
	"AWS::SES::ConfigurationSet":                  3,
	"AWS::IAM::CredentialReport":                  3,
}
