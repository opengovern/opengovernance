package configs

type RegistryType string

const (
	RegistryTypeDockerhub RegistryType = "dockerhub"
	RegistryTypeECR       RegistryType = "ecr"
	RegistryTypeGHCR      RegistryType = "ghcr"
	RegistryTypeACR       RegistryType = "acr"
)

type DockerhubCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type GhcrCredentials struct {
	Username string `json:"username"`
	Token    string `json:"token"`
}

type EcrCredentials struct {
	AccountID string `json:"account_id"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Region    string `json:"region"`
}

type AcrCredentials struct {
	LoginServer string

	TenantID     string `json:"tenant_id"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type IntegrationCredentials struct {
	RegistryType RegistryType `json:"registry_type"`

	DockerhubCredentials *DockerhubCredentials `json:"dockerhub_credentials"`
	EcrCredentials       *EcrCredentials       `json:"ecr_credentials"`
	GhcrCredentials      *GhcrCredentials      `json:"gcr_credentials"`
	AcrCredentials       *AcrCredentials       `json:"acr_credentials"`
}
