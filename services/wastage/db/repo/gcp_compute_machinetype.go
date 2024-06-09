package repo

import (
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/connector"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"gorm.io/gorm"
)

type GCPComputeMachineTypeRepo interface {
	Create(tableName string, tx *gorm.DB, m *model.GCPComputeMachineType) error
	Delete(tableName string, id string) error
	List() ([]model.GCPComputeMachineType, error)
}

type GCPComputeMachineTypeRepoImpl struct {
	db *connector.Database

	viewName string
}

func NewGCPComputeMachineTypeRepo(db *connector.Database) GCPComputeMachineTypeRepo {
	stmt := &gorm.Statement{DB: db.Conn()}
	stmt.Parse(&model.GCPComputeMachineType{})

	return &GCPComputeMachineTypeRepoImpl{
		db: db,

		viewName: stmt.Schema.Table,
	}
}

func (r *GCPComputeMachineTypeRepoImpl) Create(tableName string, tx *gorm.DB, m *model.GCPComputeMachineType) error {
	if tx == nil {
		tx = r.db.Conn()
	}
	tx = tx.Table(tableName)
	return tx.Create(&m).Error
}

func (r *GCPComputeMachineTypeRepoImpl) Delete(tableName string, sku string) error {
	return r.db.Conn().Table(tableName).Where("sku=?", sku).Delete(&model.GCPComputeMachineType{}).Error
}

func (r *GCPComputeMachineTypeRepoImpl) List() ([]model.GCPComputeMachineType, error) {
	var m []model.GCPComputeMachineType
	tx := r.db.Conn().Table(r.viewName).Find(&m)
	return m, tx.Error
}
