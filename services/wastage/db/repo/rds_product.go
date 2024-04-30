package repo

import (
	"errors"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/connector"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"gorm.io/gorm"
)

type RDSProductRepo interface {
	Create(m *model.RDSProduct) error
	Get(id uint) (*model.RDSProduct, error)
	Update(id uint, m model.RDSProduct) error
	Delete(id uint) error
	List() ([]model.RDSProduct, error)
	Truncate() error
}

type RDSProductRepoImpl struct {
	db *connector.Database
}

func NewRDSProductRepo(db *connector.Database) RDSProductRepo {
	return &RDSProductRepoImpl{
		db: db,
	}
}

func (r *RDSProductRepoImpl) Create(m *model.RDSProduct) error {
	return r.db.Conn().Create(&m).Error
}

func (r *RDSProductRepoImpl) Get(id uint) (*model.RDSProduct, error) {
	var m model.RDSProduct
	tx := r.db.Conn().Model(&model.RDSProduct{}).Where("id=?", id).First(&m)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &m, nil
}

func (r *RDSProductRepoImpl) Update(id uint, m model.RDSProduct) error {
	return r.db.Conn().Model(&model.RDSProduct{}).Where("id=?", id).Updates(&m).Error
}

func (r *RDSProductRepoImpl) Delete(id uint) error {
	return r.db.Conn().Unscoped().Delete(&model.RDSProduct{}, id).Error
}

func (r *RDSProductRepoImpl) List() ([]model.RDSProduct, error) {
	var ms []model.RDSProduct
	tx := r.db.Conn().Model(&model.RDSProduct{}).Find(&ms)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return ms, nil
}

func (r *RDSProductRepoImpl) Truncate() error {
	tx := r.db.Conn().Unscoped().Where("1 = 1").Delete(&model.RDSProduct{})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}
