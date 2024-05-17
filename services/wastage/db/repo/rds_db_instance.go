package repo

import (
	"errors"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/connector"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/sony/sonyflake"
	"gorm.io/gorm"
	"time"
)

type RDSDBInstanceRepo interface {
	Create(tableName string, tx *gorm.DB, m *model.RDSDBInstance) error
	Get(id uint) (*model.RDSDBInstance, error)
	Update(id uint, m model.RDSDBInstance) error
	Delete(id uint) error
	List() ([]model.RDSDBInstance, error)
	Truncate(tx *gorm.DB) error
	ListByInstanceType(region, instanceType, engine, engineEdition, clusterType string) ([]model.RDSDBInstance, error)
	GetCheapestByPref(pref map[string]any) (*model.RDSDBInstance, error)
	MoveViewTransaction(tableName string) error
	RemoveOldTables(tableName string) error
	CreateNewTable() (string, error)
}

type RDSDBInstanceRepoImpl struct {
	db *connector.Database

	viewName string
}

func NewRDSDBInstanceRepo(db *connector.Database) RDSDBInstanceRepo {
	stmt := &gorm.Statement{DB: db.Conn()}
	stmt.Parse(&model.RDSDBInstance{})

	return &RDSDBInstanceRepoImpl{
		db: db,

		viewName: stmt.Schema.Table,
	}
}

func (r *RDSDBInstanceRepoImpl) Create(tableName string, tx *gorm.DB, m *model.RDSDBInstance) error {
	if tx == nil {
		tx = r.db.Conn()
	}
	tx = tx.Table(tableName)
	return tx.Create(&m).Error
}

func (r *RDSDBInstanceRepoImpl) Get(id uint) (*model.RDSDBInstance, error) {
	var m model.RDSDBInstance
	tx := r.db.Conn().Table(r.viewName).Where("id=?", id).First(&m)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &m, nil
}

func (r *RDSDBInstanceRepoImpl) Update(id uint, m model.RDSDBInstance) error {
	return r.db.Conn().Table(r.viewName).Where("id=?", id).Updates(&m).Error
}

func (r *RDSDBInstanceRepoImpl) Delete(id uint) error {
	return r.db.Conn().Unscoped().Delete(&model.RDSDBInstance{}, id).Error
}

func (r *RDSDBInstanceRepoImpl) List() ([]model.RDSDBInstance, error) {
	var ms []model.RDSDBInstance
	tx := r.db.Conn().Table(r.viewName).Find(&ms)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return ms, nil
}

func (r *RDSDBInstanceRepoImpl) Truncate(tx *gorm.DB) error {
	if tx == nil {
		tx = r.db.Conn()
	}
	tx = tx.Unscoped().Where("1 = 1").Delete(&model.RDSDBInstance{})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (r *RDSDBInstanceRepoImpl) ListByInstanceType(region, instanceType, engine, engineEdition, clusterType string) ([]model.RDSDBInstance, error) {
	var ms []model.RDSDBInstance
	tx := r.db.Conn().Table(r.viewName).
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
	tx := r.db.Conn().Table(r.viewName).
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

func (r *RDSDBInstanceRepoImpl) CreateNewTable() (string, error) {
	sf := sonyflake.NewSonyflake(sonyflake.Settings{})
	var ec2InstanceTypeTable string
	for {
		id, err := sf.NextID()
		if err != nil {
			return "", err
		}

		ec2InstanceTypeTable = fmt.Sprintf("%s_%s_%d",
			r.viewName,
			time.Now().Format("2006_01_02"),
			id,
		)
		var c int32
		tx := r.db.Conn().Raw(fmt.Sprintf(`
		SELECT count(*)
		FROM information_schema.tables
		WHERE table_schema = current_schema
		AND table_name = '%s';
	`, ec2InstanceTypeTable)).First(&c)
		if tx.Error != nil {
			return "", err
		}
		if c == 0 {
			break
		}
	}

	err := r.db.Conn().Table(ec2InstanceTypeTable).AutoMigrate(&model.RDSDBInstance{})
	if err != nil {
		return "", err
	}
	return ec2InstanceTypeTable, nil
}

func (r *RDSDBInstanceRepoImpl) MoveViewTransaction(tableName string) error {
	tx := r.db.Conn().Begin()
	var err error
	defer func() {
		_ = tx.Rollback()
	}()

	dropViewQuery := fmt.Sprintf("DROP VIEW IF EXISTS %s", r.viewName)
	tx = tx.Exec(dropViewQuery)
	err = tx.Error
	if err != nil {
		return err
	}

	createViewQuery := fmt.Sprintf(`
  CREATE OR REPLACE VIEW %s AS
  SELECT *
  FROM %s;
`, r.viewName, tableName)

	tx = tx.Exec(createViewQuery)
	err = tx.Error
	if err != nil {
		return err
	}

	tx = tx.Commit()
	err = tx.Error
	if err != nil {
		return err
	}
	return nil
}

func (r *RDSDBInstanceRepoImpl) getOldTables(currentTableName string) ([]string, error) {
	query := fmt.Sprintf(`
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = current_schema
		AND table_name LIKE '%s_%%' AND table_name <> '%s';
	`, r.viewName, currentTableName)

	var tableNames []string
	tx := r.db.Conn().Raw(query).Find(&tableNames)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return tableNames, nil
}

func (r *RDSDBInstanceRepoImpl) RemoveOldTables(currentTableName string) error {
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
