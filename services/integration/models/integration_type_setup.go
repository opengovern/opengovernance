package models

type IntegrationTypeSetup struct {
	IntegrationType string `gorm:"primaryKey"`
	Enabled         bool
}
