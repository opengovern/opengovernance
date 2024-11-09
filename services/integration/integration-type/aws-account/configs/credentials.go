package configs

type IntegrationCredentials struct {
	AwsAccessKeyID            string `json:"aws_access_key_id"`
	AwsSecretAccessKey        string `json:"aws_secret_access_key"`
	CrossAccountRoleName      string `json:"cross_account_role_name"`
	RoleToAssumeInMainAccount string `json:"role_to_assume_in_main_account"`
	ExternalID                string `json:"external_id"`
}
