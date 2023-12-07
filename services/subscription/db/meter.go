package db

import (
	"github.com/go-errors/errors"
	"github.com/kaytu-io/kaytu-engine/services/subscription/db/model"
	"gorm.io/gorm"
)

func (db Database) CreateMeter(meter *model.Meter) error {
	return db.Orm.Model(&model.Meter{}).Create(meter).Error
}

func (db Database) GetMeter(workspaceId, dateHour string, meterType model.MeterType) (*model.Meter, error) {
	var meter model.Meter
	err := db.Orm.Model(&model.Meter{}).
		Where("workspace_id = ?", workspaceId).
		Where("date_hour = ?", dateHour).
		Where("meter_type = ?", meterType).
		Find(&meter).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &meter, nil
}

func (db Database) UpdateMeterPublished(workspaceId, dateHour string, meterType model.MeterType) error {
	return db.Orm.Model(&model.Meter{}).
		Where("workspace_id = ?", workspaceId).
		Where("date_hour = ?", dateHour).
		Where("meter_type = ?", meterType).
		Update("published", true).Error
}
