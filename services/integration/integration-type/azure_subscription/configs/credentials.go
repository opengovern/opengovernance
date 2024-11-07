package configs

type IntegrationCredentials struct {
	TenantID        string `json:"tenant_id"`
	ClientID        string `json:"client_id"`
	ClientSecret    string `json:"client_secret"`
	CertificatePath string `json:"certificate_path"`
	CertificatePass string `json:"certificate_pass"`
	Username        string `json:"username"`
	Password        string `json:"password"`
}
