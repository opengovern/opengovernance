package repo

import (
	"errors"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/connector"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"gorm.io/gorm"
)

type EC2InstanceTypeRepo interface {
	Create(tableName string, tx *gorm.DB, m *model.EC2InstanceType) error
	Get(id uint) (*model.EC2InstanceType, error)
	Update(tableName string, id uint, m model.EC2InstanceType) error
	UpdateExtrasByRegionAndType(tableName string, tx *gorm.DB, region, instanceType string, extras map[string]any) error
	UpdateNullExtrasByType(tableName string, tx *gorm.DB, instanceType string, extras map[string]any) error
	Delete(tableName string, id uint) error
	List() ([]model.EC2InstanceType, error)
	GetCheapestByCoreAndNetwork(bandwidth float64, pref map[string]interface{}) (*model.EC2InstanceType, error)
	Truncate(tableName string, tx *gorm.DB) error
	ListByInstanceType(instanceType, operation, region string) ([]model.EC2InstanceType, error)
	MoveViewTransaction(tableName string) error
	RemoveOldTables(tableName string) error
}

type EC2InstanceTypeRepoImpl struct {
	db *connector.Database
}

func NewEC2InstanceTypeRepo(db *connector.Database) EC2InstanceTypeRepo {
	return &EC2InstanceTypeRepoImpl{
		db: db,
	}
}

func (r *EC2InstanceTypeRepoImpl) Create(tableName string, tx *gorm.DB, m *model.EC2InstanceType) error {
	if tx == nil {
		tx = r.db.Conn().Table(tableName)
	}
	return tx.Create(&m).Error
}

func (r *EC2InstanceTypeRepoImpl) Get(id uint) (*model.EC2InstanceType, error) {
	var m model.EC2InstanceType
	tx := r.db.Conn().Table("ec2_instance_types").Where("id=?", id).First(&m)
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
	tx := r.db.Conn().Table("ec2_instance_types").
		Where("network_max_bandwidth >= ?", bandwidth).
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

func (r *EC2InstanceTypeRepoImpl) Update(tableName string, id uint, m model.EC2InstanceType) error {
	return r.db.Conn().Table(tableName).Where("id=?", id).Updates(&m).Error
}

func (r *EC2InstanceTypeRepoImpl) Delete(tableName string, id uint) error {
	return r.db.Conn().Unscoped().Table(tableName).Delete(&model.EC2InstanceType{}, id).Error
}

func (r *EC2InstanceTypeRepoImpl) List() ([]model.EC2InstanceType, error) {
	var ms []model.EC2InstanceType
	tx := r.db.Conn().Table("ec2_instance_types").Find(&ms)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return ms, nil
}

func (r *EC2InstanceTypeRepoImpl) UpdateExtrasByRegionAndType(tableName string, tx *gorm.DB, region, instanceType string, extras map[string]any) error {
	if tx == nil {
		tx = r.db.Conn()
	}
	tx = tx.Table(tableName).
		Where("region_code = ?", region).
		Where("instance_type = ?", instanceType).
		Updates(extras)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (r *EC2InstanceTypeRepoImpl) UpdateNullExtrasByType(tableName string, tx *gorm.DB, instanceType string, extras map[string]any) error {
	if tx == nil {
		tx = r.db.Conn()
	}
	for k, v := range extras {
		tx = tx.Table(tableName).
			Where("instance_type = ?", instanceType).
			Where(k+" IS NULL").
			Update(k, v)
		if tx.Error != nil {
			return tx.Error
		}
	}
	return nil
}

func (r *EC2InstanceTypeRepoImpl) ListByInstanceType(instanceType, operation, region string) ([]model.EC2InstanceType, error) {
	var ms []model.EC2InstanceType
	tx := r.db.Conn().Table("ec2_instance_types").
		Where("instance_type = ? AND capacity_status = 'Used'", instanceType).
		Where("region_code = ?", region).
		Where("operation = ?", operation).
		Find(&ms)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return ms, nil
}

func (r *EC2InstanceTypeRepoImpl) Truncate(tableName string, tx *gorm.DB) error {
	if tx == nil {
		tx = r.db.Conn().Table(tableName)
	}
	tx = tx.Unscoped().Where("1 = 1").Delete(&model.EC2InstanceType{})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (r *EC2InstanceTypeRepoImpl) MoveViewTransaction(tableName string) error {
	tx := r.db.Conn().Begin()
	var err error
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	dropViewQuery := fmt.Sprintf("DROP VIEW IF EXISTS ec2_instance_types")
	tx = tx.Exec(dropViewQuery)
	err = tx.Error
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	createViewQuery := fmt.Sprintf(`
  CREATE OR REPLACE VIEW ec2_instance_types AS
  SELECT *
  FROM %s;
`, tableName)

	tx = tx.Exec(createViewQuery)
	err = tx.Error
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	tx = tx.Commit()
	err = tx.Error
	if err != nil {
		return err
	}
	return nil
}

func (r *EC2InstanceTypeRepoImpl) getOldTables(currentTableName string) ([]string, error) {
	query := fmt.Sprintf(`
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = current_schema
		AND table_name LIKE 'ec2_instance_types_%%' AND table_name <> '%s';
	`, currentTableName)

	var tableNames []string
	tx := r.db.Conn().Raw(query).Find(&tableNames)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return tableNames, nil
}

func (r *EC2InstanceTypeRepoImpl) RemoveOldTables(currentTableName string) error {
	tableNames, err := r.getOldTables(currentTableName)
	if err != nil {
		return err
	}
	for _, tn := range tableNames {
		err = r.db.Conn().Migrator().DropTable(tn)
		if err != nil {
			return err
		}
	}
	return nil
}
