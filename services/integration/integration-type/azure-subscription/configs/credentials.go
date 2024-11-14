package configs

type IntegrationCredentials struct {
	TenantID            string `json:"tenant_id"`
	ClientID            string `json:"client_id"`
	ClientPassword      string `json:"client_password"`
	Certificate         string `json:"certificate"`
	CertificatePassword string `json:"certificate_password"`
	SpnObjectId         string `json:"spn_object_id"`
}
