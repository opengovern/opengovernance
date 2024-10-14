package db

import (
	"github.com/opengovern/opengovernance/services/integration/model"
)

func (db Database) ListConnectionGroups() ([]model.ConnectionGroup, error) {
	var cgs []model.ConnectionGroup
	tx := db.Orm.Find(&cgs)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return cgs, nil
}

func (db Database) GetConnectionGroupByName(name string) (*model.ConnectionGroup, error) {
	var cg model.ConnectionGroup
	err := db.Orm.First(&cg, "name = ?", name).Error
	if err != nil {
		return nil, err
	}
	return &cg, nil
}

func (db Database) DeleteConnectionGroups() error {
	tx := db.Orm.
		Where("1 = 1").
		Unscoped().
		Delete(&model.ConnectionGroup{})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
