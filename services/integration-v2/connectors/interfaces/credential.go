package interfaces

import "github.com/opengovern/opengovernance/services/integration-v2/models"

type Credential interface {
	HealthCheck() error
	GetIntegrations() ([]models.Integration, error)
	toJSON() ([]byte, error) // Method to store the credentials as JSON in the database
}

type BaseCredential struct {
	Credential Credential
}
