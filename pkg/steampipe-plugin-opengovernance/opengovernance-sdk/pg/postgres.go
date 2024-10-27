package pg

import (
	"context"
	"errors"

	onboard "github.com/opengovern/opengovernance/services/integration/model"
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

func (c Client) GetConnectionByIDs(ctx context.Context, opengovernanceId string, id string) (*onboard.Connection, error) {
	var result onboard.Connection
	var err error
	tx := c.db.Preload(clause.Associations).Model(&onboard.Connection{})
	switch {
	case opengovernanceId != "" && id != "":
		err = tx.Where("id = ? AND source_id = ?", opengovernanceId, id).First(&result).Error
	case opengovernanceId != "" && id == "":
		err = tx.Where("id = ?", opengovernanceId).First(&result).Error
	case opengovernanceId == "" && id != "":
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
