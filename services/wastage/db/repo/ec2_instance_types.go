package repo

import (
	"errors"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/connector"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"gorm.io/gorm"
)

type EC2InstanceTypeRepo interface {
	Create(m *model.EC2InstanceType) error
	Get(id uint) (*model.EC2InstanceType, error)
	Update(id uint, m model.EC2InstanceType) error
	Delete(id uint) error
	List() ([]model.EC2InstanceType, error)
	GetCheapestByCoreAndNetwork(bandwidth float64, pref map[string]interface{}) (*model.EC2InstanceType, error)
	Truncate() error
	ListByInstanceType(instanceType string) ([]model.EC2InstanceType, error)
	GetCurrentInstanceType(instanceType, tenancy, os string) (*model.EC2InstanceType, error)
}

type EC2InstanceTypeRepoImpl struct {
	db *connector.Database
}

func NewEC2InstanceTypeRepo(db *connector.Database) EC2InstanceTypeRepo {
	return &EC2InstanceTypeRepoImpl{
		db: db,
	}
}

func (r *EC2InstanceTypeRepoImpl) Create(m *model.EC2InstanceType) error {
	return r.db.Conn().Create(&m).Error
}

func (r *EC2InstanceTypeRepoImpl) Get(id uint) (*model.EC2InstanceType, error) {
	var m model.EC2InstanceType
	tx := r.db.Conn().Model(&model.EC2InstanceType{}).Where("id=?", id).First(&m)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &m, nil
}

func (r *EC2InstanceTypeRepoImpl) GetCheapestByCoreAndNetwork(bandwidth float64, pref map[string]interface{}) (*model.EC2InstanceType, error) {
	var m model.EC2InstanceType
	tx := r.db.Conn().Model(&model.EC2InstanceType{}).
		Where("network_max_bandwidth >= ?", bandwidth).
		Where("pre_installed_sw = 'NA'").
		Where("capacity_status = 'Used'").
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

func (r *EC2InstanceTypeRepoImpl) Update(id uint, m model.EC2InstanceType) error {
	return r.db.Conn().Model(&model.EC2InstanceType{}).Where("id=?", id).Updates(&m).Error
}

func (r *EC2InstanceTypeRepoImpl) Delete(id uint) error {
	return r.db.Conn().Unscoped().Delete(&model.EC2InstanceType{}, id).Error
}

func (r *EC2InstanceTypeRepoImpl) List() ([]model.EC2InstanceType, error) {
	var ms []model.EC2InstanceType
	tx := r.db.Conn().Model(&model.EC2InstanceType{}).Find(&ms)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return ms, nil
}

func (r *EC2InstanceTypeRepoImpl) ListByInstanceType(instanceType string) ([]model.EC2InstanceType, error) {
	var ms []model.EC2InstanceType
	tx := r.db.Conn().Model(&model.EC2InstanceType{}).Where("instance_type = ?", instanceType).Find(&ms)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return ms, nil
}

func (r *EC2InstanceTypeRepoImpl) Truncate() error {
	tx := r.db.Conn().Unscoped().Where("1 = 1").Delete(&model.EC2InstanceType{})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (r *EC2InstanceTypeRepoImpl) GetCurrentInstanceType(instanceType, tenancy, os string) (*model.EC2InstanceType, error) {
	var m model.EC2InstanceType
	tx := r.db.Conn().Model(&model.EC2InstanceType{}).
		Where("tenancy = ?", tenancy).
		Where("instance_type = ?", instanceType).
		Where("pre_installed_sw = 'NA'").
		Where("operating_system = ?", os).
		Where("license_model = 'No License required'").
		Where("capacity_status = 'Used'")

	tx = tx.Order("price_per_unit ASC").First(&m)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &m, nil
}