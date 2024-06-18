package repo

import (
	"errors"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/connector"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"gorm.io/gorm"
)

type PreProcessJobRepo interface {
	Create(m *model.PreProcessJob) error
	Get(id uint) (*model.PreProcessJob, error)
	Update(id uint, m model.PreProcessJob) error
	Delete(id uint) error
	List() ([]model.PreProcessJob, error)
}

type PreProcessJobRepoImpl struct {
	db *connector.Database
}

func NewPreProcessJobRepo(db *connector.Database) PreProcessJobRepo {
	return &PreProcessJobRepoImpl{
		db: db,
	}
}

func (r *PreProcessJobRepoImpl) Create(m *model.PreProcessJob) error {
	return r.db.Conn().Create(&m).Error
}

func (r *PreProcessJobRepoImpl) Get(id uint) (*model.PreProcessJob, error) {
	var m model.PreProcessJob
	tx := r.db.Conn().Model(&model.PreProcessJob{}).Where("id=?", id).First(&m)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &m, nil
}

func (r *PreProcessJobRepoImpl) Update(id uint, m model.PreProcessJob) error {
	return r.db.Conn().Model(&model.PreProcessJob{}).Where("id=?", id).Updates(&m).Error
}

func (r *PreProcessJobRepoImpl) Delete(id uint) error {
	return r.db.Conn().Unscoped().Delete(&model.PreProcessJob{
		Model: gorm.Model{
			ID: id,
		},
	}).Error
}

func (r *PreProcessJobRepoImpl) List() ([]model.PreProcessJob, error) {
	var ms []model.PreProcessJob
	tx := r.db.Conn().Model(&model.PreProcessJob{}).Find(&ms)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return ms, nil
}
