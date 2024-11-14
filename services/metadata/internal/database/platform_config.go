package database

import (
	"github.com/opengovern/opengovernance/services/metadata/models"
)

func (db Database) ListApp() ([]models.PlatformConfiguration, error) {
	var apps []models.PlatformConfiguration
	err := db.orm.Model(&models.PlatformConfiguration{}).Find(&apps).Error
	if err != nil {
		return nil, err
	}
	return apps, nil
}

func (db Database) CreateApp(app *models.PlatformConfiguration) error {
	return db.orm.Model(&models.PlatformConfiguration{}).Create(app).Error
}

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
