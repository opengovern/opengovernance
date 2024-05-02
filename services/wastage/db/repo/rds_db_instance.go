package repo

import (
	"errors"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/connector"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"gorm.io/gorm"
)

type RDSDBInstanceRepo interface {
	Create(m *model.RDSDBInstance) error
	Get(id uint) (*model.RDSDBInstance, error)
	Update(id uint, m model.RDSDBInstance) error
	Delete(id uint) error
	List() ([]model.RDSDBInstance, error)
	Truncate() error
	ListByInstanceType(region, instanceType, engine, clusterType string) ([]model.RDSDBInstance, error)
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

func (r *RDSDBInstanceRepoImpl) Create(m *model.RDSDBInstance) error {
	return r.db.Conn().Create(&m).Error
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

func (r *RDSDBInstanceRepoImpl) Truncate() error {
	tx := r.db.Conn().Unscoped().Where("1 = 1").Delete(&model.RDSDBInstance{})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (r *RDSDBInstanceRepoImpl) ListByInstanceType(region, instanceType, engine, clusterType string) ([]model.RDSDBInstance, error) {
	var ms []model.RDSDBInstance
	tx := r.db.Conn().Model(&model.RDSDBInstance{}).
		Where("region_code = ?", region).
		Where("instance_type = ?", instanceType).
		Where("database_engine = ?", engine).
		Where("deployment_option = ?", clusterType).
		Find(&ms)
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
