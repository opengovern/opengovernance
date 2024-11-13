package configs

type IntegrationCredentials struct {
	Token          string `json:"token"`
	BaseURL        string `json:"base_url"`
	AppId          string `json:"app_id"`
	InstallationId string `json:"installation_id"`
	PrivateKeyPath string `json:"private_key_path"`
}
