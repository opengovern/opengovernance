package database

import (
	"github.com/opengovern/opengovernance/pkg/metadata/models"
)

func (db Database) AppConfigured(configured bool) error {
	return db.orm.Model(&models.AppConfiguration{}).Update("configured", configured).Error
}

func (db Database) GetAppConfiguration() (*models.AppConfiguration, error) {
	var appConfiguration models.AppConfiguration
	err := db.orm.Model(&models.AppConfiguration{}).First(&configMetadata).Error
	if err != nil {
		return nil, err
	}
	return &appConfiguration, nil
}
