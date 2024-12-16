package configs

type RegistryType string

const (
	RegistryTypeDockerhub RegistryType = "dockerhub"
	RegistryTypeECR       RegistryType = "ecr"
	RegistryTypeGHCR      RegistryType = "ghcr"
	RegistryTypeACR       RegistryType = "acr"
	RegistryTypeGCR       RegistryType = "gcr"
)

type DockerhubCredentials struct {
	Owner string `json:"owner"`

	Username string `json:"username"`
	Password string `json:"password"`
}

type GhcrCredentials struct {
	Owner string `json:"owner"`

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
	LoginServer string `json:"login_server"`

	TenantID     string `json:"tenant_id"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type GcrCredentials struct {
	ProjectID string `json:"project_id"`
	Location  string `json:"location"`
	JSONKey   string `json:"json_key"`
}

type IntegrationCredentials struct {
	DockerhubCredentials *DockerhubCredentials `json:"dockerhub_credentials"`
	EcrCredentials       *EcrCredentials       `json:"ecr_credentials"`
	GhcrCredentials      *GhcrCredentials      `json:"ghcr_credentials"`
	AcrCredentials       *AcrCredentials       `json:"acr_credentials"`
	GcrCredentials       *GcrCredentials       `json:"gcr_credentials"`
}

func (c IntegrationCredentials) GetRegistryType() RegistryType {
	switch {
	case c.DockerhubCredentials != nil:
		return RegistryTypeDockerhub
	case c.EcrCredentials != nil:
		return RegistryTypeECR
	case c.GhcrCredentials != nil:
		return RegistryTypeGHCR
	case c.AcrCredentials != nil:
		return RegistryTypeACR
	case c.GcrCredentials != nil:
		return RegistryTypeGCR
	default:
		return ""
	}
}
