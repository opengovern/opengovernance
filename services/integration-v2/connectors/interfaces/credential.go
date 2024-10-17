package interfaces

import "github.com/opengovern/opengovernance/services/integration-v2/models"

type CredentialType interface {
	HealthCheck() error
	GetIntegrations() ([]models.Integration, error)
	ToJSON() ([]byte, error) // Method to store the credentials as JSON in the database
	ParseJSON([]byte) error
}
