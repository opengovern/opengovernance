package repo

import (
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/connector"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"gorm.io/gorm"
)

type GCPComputeSKURepo interface {
	Create(tableName string, tx *gorm.DB, m *model.GCPComputeSKU) error
	Delete(tableName string, id string) error
	List() ([]model.GCPComputeSKU, error)
}

type GCPComputeSKURepoImpl struct {
	db *connector.Database

	viewName string
}

func NewGCPComputeSKURepo(db *connector.Database) GCPComputeSKURepo {
	stmt := &gorm.Statement{DB: db.Conn()}
	stmt.Parse(&model.GCPComputeSKU{})

	return &GCPComputeSKURepoImpl{
		db: db,

		viewName: stmt.Schema.Table,
	}
}

func (r *GCPComputeSKURepoImpl) Create(tableName string, tx *gorm.DB, m *model.GCPComputeSKU) error {
	if tx == nil {
		tx = r.db.Conn()
	}
	tx = tx.Table(tableName)
	return tx.Create(&m).Error
}

func (r *GCPComputeSKURepoImpl) Delete(tableName string, sku string) error {
	return r.db.Conn().Table(tableName).Where("sku=?", sku).Delete(&model.GCPComputeSKU{}).Error
}

func (r *GCPComputeSKURepoImpl) List() ([]model.GCPComputeSKU, error) {
	var m []model.GCPComputeSKU
	tx := r.db.Conn().Table(r.viewName).Find(&m)
	return m, tx.Error
}
