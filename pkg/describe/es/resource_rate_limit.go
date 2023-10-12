package es

var ResourceRateLimit = map[string]int{
	"Microsoft.Management/groups":                 1,
	"Microsoft.CostManagement/CostByResourceType": 1,
	"AWS::Organizations::Account":                 1,
	"AWS::Shield::ProtectionGroup":                1,
	"AWS::IAM::Policy":                            1,
	"AWS::IAM::Role":                              1,
	"AWS::SES::ConfigurationSet":                  1,
	"AWS::IAM::CredentialReport":                  1,
}
