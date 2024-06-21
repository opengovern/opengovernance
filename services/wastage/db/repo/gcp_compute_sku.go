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

type GCPComputeSKURepo interface {
	Create(tableName string, tx *gorm.DB, m *model.GCPComputeSKU) error
	Delete(tableName string, id string) error
	List() ([]model.GCPComputeSKU, error)
	GetCheapestCustomCore(machineFamily string, pref map[string]interface{}) (*model.GCPComputeSKU, error)
	GetCheapestCustomRam(machineFamily string, pref map[string]interface{}) (*model.GCPComputeSKU, error)
	CreateNewTable() (string, error)
	MoveViewTransaction(tableName string) error
	RemoveOldTables(currentTableName string) error
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

func (r *GCPComputeSKURepoImpl) GetCheapestCustomCore(machineFamily string, pref map[string]interface{}) (*model.GCPComputeSKU, error) {
	var m model.GCPComputeSKU
	tx := r.db.Conn().Table(r.viewName).
		Where("resource_family = 'Compute'").
		Where("unit_price != 0").
		Where("machine_family = ?", machineFamily).
		Where("resource_group = 'CPU'")
	for k, v := range pref {
		tx = tx.Where(k, v)
	}
	tx = tx.Order("unit_price ASC").First(&m)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &m, nil
}

func (r *GCPComputeSKURepoImpl) GetCheapestCustomRam(machineFamily string, pref map[string]interface{}) (*model.GCPComputeSKU, error) {
	var m model.GCPComputeSKU
	tx := r.db.Conn().Table(r.viewName).
		Where("resource_family = 'Compute'").
		Where("unit_price != 0").
		Where("machine_family = ?", machineFamily).
		Where("resource_group = 'RAM'")
	for k, v := range pref {
		tx = tx.Where(k, v)
	}
	tx = tx.Order("unit_price ASC").First(&m)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &m, nil
}

func (r *GCPComputeSKURepoImpl) CreateNewTable() (string, error) {
	sf := sonyflake.NewSonyflake(sonyflake.Settings{})
	var gcpComputeSKUTable string
	for {
		id, err := sf.NextID()
		if err != nil {
			return "", err
		}

		gcpComputeSKUTable = fmt.Sprintf("%s_%s_%d",
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
	`, gcpComputeSKUTable)).First(&c)
		if tx.Error != nil {
			return "", err
		}
		if c == 0 {
			break
		}
	}

	err := r.db.Conn().Table(gcpComputeSKUTable).AutoMigrate(&model.GCPComputeSKU{})
	if err != nil {
		return "", err
	}
	return gcpComputeSKUTable, nil
}

func (r *GCPComputeSKURepoImpl) MoveViewTransaction(tableName string) error {
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

func (r *GCPComputeSKURepoImpl) getOldTables(currentTableName string) ([]string, error) {
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

func (r *GCPComputeSKURepoImpl) RemoveOldTables(currentTableName string) error {
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
