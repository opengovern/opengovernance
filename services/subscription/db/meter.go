package db

import (
	"github.com/go-errors/errors"
	"github.com/kaytu-io/kaytu-engine/services/subscription/api/entities"
	"github.com/kaytu-io/kaytu-engine/services/subscription/db/model"
	"gorm.io/gorm"
	"time"
)

func (db Database) CreateMeter(meter *model.Meter) error {
	return db.Orm.Model(&model.Meter{}).Create(meter).Error
}

func (db Database) SumOfMeter(workspaceId []string, meterType entities.MeterType, startTime, endTime time.Time) (int64, error) {
	var sum int64
	tx := db.Orm.Model(&model.Meter{}).
		Where("workspace_id IN ?", workspaceId).
		Where("meter_type = ?", meterType).
		Where("usage_date >= ?", startTime).
		Where("usage_date <= ?", endTime).
		Select("coalesce(SUM(value),0)").Find(&sum)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, tx.Error
	} else if tx.RowsAffected == 0 {
		return 0, nil
	}
	return sum, nil
}

func (db Database) AvgOfMeter(workspaceId []string, meterType entities.MeterType, startTime, endTime time.Time) (float64, error) {
	var sum float64
	tx := db.Orm.Model(&model.Meter{}).
		Where("workspace_id IN ?", workspaceId).
		Where("meter_type = ?", meterType).
		Where("usage_date >= ?", startTime).
		Where("usage_date <= ?", endTime).
		Select("coalesce(AVG(value),0)").Find(&sum)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, tx.Error
	} else if tx.RowsAffected == 0 {
		return 0, nil
	}
	return sum, nil
}

func (db Database) GetMeter(workspaceId string, usageDate time.Time, meterType entities.MeterType) (*model.Meter, error) {
	var meter model.Meter
	tx := db.Orm.Model(&model.Meter{}).
		Where("workspace_id = ?", workspaceId).
		Where("usage_date = ?", usageDate).
		Where("meter_type = ?", meterType).
		Find(&meter)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	} else if tx.RowsAffected == 0 {
		return nil, nil
	}
	return &meter, nil
}

func (db Database) GetUnpublishedMetersOlderThan(olderThanTime time.Time) ([]*model.Meter, error) {
	var meters []*model.Meter
	tx := db.Orm.Model(&model.Meter{}).
		Where("published = ?", false).
		Where("created_at <= ?", olderThanTime).
		Find(&meters)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	} else if tx.RowsAffected == 0 {
		return nil, nil
	}
	return meters, nil
}

func (db Database) UpdateMeterPublished(workspaceId string, usageDate time.Time, meterType entities.MeterType) error {
	return db.Orm.Model(&model.Meter{}).
		Where("workspace_id = ?", workspaceId).
		Where("usage_date = ?", usageDate).
		Where("meter_type = ?", meterType).
		Update("published", true).Error
}

func (db Database) UpdateMetersPublished(meters []*model.Meter) error {
	err := db.Orm.Transaction(func(tx *gorm.DB) error {
		for _, meter := range meters {
			err := tx.Model(&model.Meter{}).
				Where("workspace_id = ?", meter.WorkspaceID).
				Where("usage_date = ?", meter.UsageDate).
				Where("meter_type = ?", meter.MeterType).
				Update("published", true).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
