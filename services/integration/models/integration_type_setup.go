package models

import "github.com/opengovern/og-util/pkg/integration"

type IntegrationTypeSetup struct {
	IntegrationType integration.Type `gorm:"primaryKey"`
	Enabled         bool
}
