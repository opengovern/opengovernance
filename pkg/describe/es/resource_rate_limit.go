package es

var ResourceRateLimit = map[string]int{
	"Microsoft.Management/groups":                 5,
	"Microsoft.CostManagement/CostByResourceType": 3,
	"Microsoft.Storage/tables":                    5,
	"AWS::Organizations::Account":                 5,
	"AWS::Shield::ProtectionGroup":                5,
	"AWS::IAM::Policy":                            5,
	"AWS::IAM::Role":                              5,
	"AWS::SES::ConfigurationSet":                  5,
	"AWS::IAM::CredentialReport":                  5,
}
