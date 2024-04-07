package pg

import (
	"context"
	"errors"

	onboard "github.com/kaytu-io/kaytu-engine/services/integration/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (c Client) ListConnections(ctx context.Context) ([]onboard.Connection, error) {
	var result []onboard.Connection
	err := c.db.Preload(clause.Associations).Find(&result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c Client) GetConnectionByIDs(ctx context.Context, kaytuId string, id string) (*onboard.Connection, error) {
	var result onboard.Connection
	var err error
	tx := c.db.Preload(clause.Associations).Model(&onboard.Connection{})
	switch {
	case kaytuId != "" && id != "":
		err = tx.Where("id = ? AND source_id = ?", kaytuId, id).First(&result).Error
	case kaytuId != "" && id == "":
		err = tx.Where("id = ?", kaytuId).First(&result).Error
	case kaytuId == "" && id != "":
		err = tx.Where("source_id = ?", id).First(&result).Error
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}
