package repo

import (
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/connector"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
)

type UsageRepo interface {
	Create(m *model.Usage) error
	Update(id uint, m model.Usage) error
	List() ([]model.Usage, error)
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
