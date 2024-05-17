package repo

import (
	"errors"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/connector"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"gorm.io/gorm"
)

type RDSDBInstanceRepo interface {
	Create(tx *gorm.DB, m *model.RDSDBInstance) error
	Get(id uint) (*model.RDSDBInstance, error)
	Update(id uint, m model.RDSDBInstance) error
	Delete(id uint) error
	List() ([]model.RDSDBInstance, error)
	Truncate(tableName string, tx *gorm.DB) error
	ListByInstanceType(region, instanceType, engine, engineEdition, clusterType string) ([]model.RDSDBInstance, error)
	GetCheapestByPref(pref map[string]any) (*model.RDSDBInstance, error)
}

type RDSDBInstanceRepoImpl struct {
	db *connector.Database
}

func NewRDSDBInstanceRepo(db *connector.Database) RDSDBInstanceRepo {
	return &RDSDBInstanceRepoImpl{
		db: db,
	}
}

func (r *RDSDBInstanceRepoImpl) Create(tx *gorm.DB, m *model.RDSDBInstance) error {
	if tx == nil {
		tx = r.db.Conn()
	}
	return tx.Create(&m).Error
}

func (r *RDSDBInstanceRepoImpl) Get(id uint) (*model.RDSDBInstance, error) {
	var m model.RDSDBInstance
	tx := r.db.Conn().Model(&model.RDSDBInstance{}).Where("id=?", id).First(&m)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &m, nil
}

func (r *RDSDBInstanceRepoImpl) Update(id uint, m model.RDSDBInstance) error {
	return r.db.Conn().Model(&model.RDSDBInstance{}).Where("id=?", id).Updates(&m).Error
}

func (r *RDSDBInstanceRepoImpl) Delete(id uint) error {
	return r.db.Conn().Unscoped().Delete(&model.RDSDBInstance{}, id).Error
}

func (r *RDSDBInstanceRepoImpl) List() ([]model.RDSDBInstance, error) {
	var ms []model.RDSDBInstance
	tx := r.db.Conn().Model(&model.RDSDBInstance{}).Find(&ms)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return ms, nil
}

func (r *RDSDBInstanceRepoImpl) Truncate(tableName string, tx *gorm.DB) error {
	if tx == nil {
		tx = r.db.Conn().Table(tableName)
	}
	tx = tx.Unscoped().Where("1 = 1").Delete(&model.RDSDBInstance{})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (r *RDSDBInstanceRepoImpl) ListByInstanceType(region, instanceType, engine, engineEdition, clusterType string) ([]model.RDSDBInstance, error) {
	var ms []model.RDSDBInstance
	tx := r.db.Conn().Model(&model.RDSDBInstance{}).
		Where("region_code = ?", region).
		Where("instance_type = ?", instanceType).
		Where("database_engine = ?", engine).
		Where("deployment_option = ?", clusterType)
	if engineEdition != "" {
		tx = tx.Where("database_edition = ?", engineEdition)
	}
	tx = tx.Find(&ms)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return ms, nil
}

func (r *RDSDBInstanceRepoImpl) GetCheapestByPref(pref map[string]any) (*model.RDSDBInstance, error) {
	var m model.RDSDBInstance
	tx := r.db.Conn().Model(&model.RDSDBInstance{}).
		Where("price_per_unit != 0")
	for k, v := range pref {
		tx = tx.Where(k, v)
	}
	tx = tx.Order("price_per_unit ASC").First(&m)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &m, nil
}
