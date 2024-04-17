package repo

import (
	"errors"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/connector"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"gorm.io/gorm"
)

type DataAgeRepo interface {
	Create(m *model.DataAge) error
	Get(dataType string) (*model.DataAge, error)
	Update(dataType string, m model.DataAge) error
	Delete(dataType string) error
	List() ([]model.DataAge, error)
}

type DataAgeRepoImpl struct {
	db *connector.Database
}

func NewDataAgeRepo(db *connector.Database) DataAgeRepo {
	return &DataAgeRepoImpl{
		db: db,
	}
}

func (r *DataAgeRepoImpl) Create(m *model.DataAge) error {
	return r.db.Conn().Create(&m).Error
}

func (r *DataAgeRepoImpl) Get(dataType string) (*model.DataAge, error) {
	var m model.DataAge
	tx := r.db.Conn().Model(&model.DataAge{}).Where("data_type=?", dataType).First(&m)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &m, nil
}

func (r *DataAgeRepoImpl) Update(dataType string, m model.DataAge) error {
	return r.db.Conn().Model(&model.DataAge{}).Where("data_type=?", dataType).Updates(&m).Error
}

func (r *DataAgeRepoImpl) Delete(dataType string) error {
	return r.db.Conn().Delete(&model.DataAge{}, dataType).Error
}

func (r *DataAgeRepoImpl) List() ([]model.DataAge, error) {
	var ms []model.DataAge
	tx := r.db.Conn().Model(&model.DataAge{}).Find(&ms)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return ms, nil
}
