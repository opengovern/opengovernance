package repo

import (
	"errors"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/connector"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"gorm.io/gorm"
)

type RDSDBStorageRepo interface {
	Create(m *model.RDSDBStorage) error
	Get(id uint) (*model.RDSDBStorage, error)
	Update(id uint, m model.RDSDBStorage) error
	Delete(id uint) error
	List() ([]model.RDSDBStorage, error)
	Truncate() error
}

type RDSDBStorageRepoImpl struct {
	db *connector.Database
}

func NewRDSDBStorageRepo(db *connector.Database) RDSDBStorageRepo {
	return &RDSDBStorageRepoImpl{
		db: db,
	}
}

func (r *RDSDBStorageRepoImpl) Create(m *model.RDSDBStorage) error {
	return r.db.Conn().Create(&m).Error
}

func (r *RDSDBStorageRepoImpl) Get(id uint) (*model.RDSDBStorage, error) {
	var m model.RDSDBStorage
	tx := r.db.Conn().Model(&model.RDSDBStorage{}).Where("id=?", id).First(&m)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &m, nil
}

func (r *RDSDBStorageRepoImpl) Update(id uint, m model.RDSDBStorage) error {
	return r.db.Conn().Model(&model.RDSDBStorage{}).Where("id=?", id).Updates(&m).Error
}

func (r *RDSDBStorageRepoImpl) Delete(id uint) error {
	return r.db.Conn().Unscoped().Delete(&model.RDSDBStorage{}, id).Error
}

func (r *RDSDBStorageRepoImpl) List() ([]model.RDSDBStorage, error) {
	var ms []model.RDSDBStorage
	tx := r.db.Conn().Model(&model.RDSDBStorage{}).Find(&ms)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return ms, nil
}

func (r *RDSDBStorageRepoImpl) Truncate() error {
	tx := r.db.Conn().Unscoped().Where("1 = 1").Delete(&model.RDSDBStorage{})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}
