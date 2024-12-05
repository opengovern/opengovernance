package configs

type IntegrationCredentials struct {
	AdminEmail string `json:"admin_email"`
	CustomerID string `json:"customer_id"`
	KeyFile    string `json:"key_file"`
}
