package configs

import "encoding/json"

type IntegrationCredentials struct {
	AdminEmail string          `json:"admin_email"`
	CustomerID string          `json:"customer_id"`
	KeyFile    json.RawMessage `json:"key_file"`
}
