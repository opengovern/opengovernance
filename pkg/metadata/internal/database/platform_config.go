package database

import (
	"github.com/opengovern/opengovernance/pkg/metadata/models"
)

func (db Database) AppConfigured(configured bool) error {
	return db.orm.Model(&models.PlatformConfiguration{}).Update("configured", configured).Error
}

func (db Database) GetAppConfiguration() (*models.PlatformConfiguration, error) {
	var appConfiguration models.PlatformConfiguration
	err := db.orm.Model(&models.PlatformConfiguration{}).First(&appConfiguration).Error
	if err != nil {
		return nil, err
	}
	return &appConfiguration, nil
}
