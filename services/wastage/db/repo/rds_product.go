package repo

import (
	"errors"
	"fmt"
	"github.com/opengovern/opengovernance/services/wastage/db/connector"
	"github.com/opengovern/opengovernance/services/wastage/db/model"
	"github.com/sony/sonyflake"
	"gorm.io/gorm"
	"time"
)

type RDSProductRepo interface {
	Create(tableName string, tx *gorm.DB, m *model.RDSProduct) error
	Get(id uint) (*model.RDSProduct, error)
	Update(id uint, m model.RDSProduct) error
	Delete(id uint) error
	List() ([]model.RDSProduct, error)
	Truncate(tx *gorm.DB) error
	MoveViewTransaction(tableName string) error
	RemoveOldTables(tableName string) error
	CreateNewTable() (string, error)
}

type RDSProductRepoImpl struct {
	db *connector.Database

	viewName string
}

func NewRDSProductRepo(db *connector.Database) RDSProductRepo {
	stmt := &gorm.Statement{DB: db.Conn()}
	stmt.Parse(&model.RDSProduct{})

	return &RDSProductRepoImpl{
		db: db,

		viewName: stmt.Schema.Table,
	}
}

func (r *RDSProductRepoImpl) Create(tableName string, tx *gorm.DB, m *model.RDSProduct) error {
	if tx == nil {
		tx = r.db.Conn()
	}
	tx = tx.Table(tableName)
	return tx.Create(&m).Error
}

func (r *RDSProductRepoImpl) Get(id uint) (*model.RDSProduct, error) {
	var m model.RDSProduct
	tx := r.db.Conn().Table(r.viewName).Where("id=?", id).First(&m)
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
	tx := r.db.Conn().Table(r.viewName).Find(&ms)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return ms, nil
}

func (r *RDSProductRepoImpl) Truncate(tx *gorm.DB) error {
	if tx == nil {
		tx = r.db.Conn()
	}
	tx = tx.Unscoped().Where("1 = 1").Delete(&model.RDSProduct{})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (r *RDSProductRepoImpl) CreateNewTable() (string, error) {
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

	err := r.db.Conn().Table(ec2InstanceTypeTable).AutoMigrate(&model.RDSProduct{})
	if err != nil {
		return "", err
	}
	return ec2InstanceTypeTable, nil
}

func (r *RDSProductRepoImpl) MoveViewTransaction(tableName string) error {
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

func (r *RDSProductRepoImpl) getOldTables(currentTableName string) ([]string, error) {
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

func (r *RDSProductRepoImpl) RemoveOldTables(currentTableName string) error {
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
