package repo

import (
	"errors"
	"github.com/opengovern/opengovernance/services/wastage/db/connector"
	"github.com/opengovern/opengovernance/services/wastage/db/model"
	"gorm.io/gorm"
)

type UsageRepo interface {
	Create(m *model.Usage) error
	Update(id uint, m model.Usage) error
	List() ([]model.Usage, error)
	GetRandomNotMoved() (*model.Usage, error)
}

type UsageRepoImpl struct {
	db *connector.Database
}

func NewUsageRepo(db *connector.Database) UsageRepo {
	return &UsageRepoImpl{
		db: db,
	}
}

func (r *UsageRepoImpl) Create(m *model.Usage) error {
	return r.db.Conn().Create(&m).Error
}

func (r *UsageRepoImpl) Update(id uint, m model.Usage) error {
	return r.db.Conn().Model(&model.Usage{}).Where("id=?", id).Updates(&m).Error
}

func (r *UsageRepoImpl) List() ([]model.Usage, error) {
	var ms []model.Usage
	tx := r.db.Conn().Model(&model.Usage{}).Find(&ms)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return ms, nil
}

func (r *UsageRepoImpl) GetRandomNotMoved() (*model.Usage, error) {
	var m model.Usage
	tx := r.db.Conn().Model(&model.Usage{}).Where("moved=? OR moved IS NULL", false).First(&m)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &m, nil
}
