package db

import "github.com/kaytu-io/kaytu-engine/services/subscription/db/model"

func (db Database) CreateMeter(meter *model.Meter) error {
	return db.Orm.Model(&model.Meter{}).Create(meter).Error
}

func (db Database) GetMeter(workspaceId, dateHour string, meterType model.MeterType) (*model.Meter, error) {
	var meter *model.Meter
	err := db.Orm.Model(&model.Meter{}).
		Where("workspace_id = ?", workspaceId).
		Where("date_hour = ?", dateHour).
		Where("meter_type = ?", meterType).
		Find(&meter).Error
	return meter, err
}

func (db Database) UpdateMeterPublished(workspaceId, dateHour string, meterType model.MeterType) error {
	return db.Orm.Model(&model.Meter{}).
		Where("workspace_id = ?", workspaceId).
		Where("date_hour = ?", dateHour).
		Where("meter_type = ?", meterType).
		Update("published", true).Error
}
