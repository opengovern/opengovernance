package repo

import (
	"errors"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/connector"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"gorm.io/gorm"
)

type OrganizationRepo interface {
	Create(m *model.Organization) error
	Update(id uint, m model.Organization) error
	List() ([]model.Organization, error)
	Get(id string) (*model.Organization, error)
}

type OrganizationRepoImpl struct {
	db *connector.Database
}

func NewOrganizationRepo(db *connector.Database) OrganizationRepo {
	return &OrganizationRepoImpl{
		db: db,
	}
}

func (r *OrganizationRepoImpl) Create(m *model.Organization) error {
	return r.db.Conn().Create(&m).Error
}

func (r *OrganizationRepoImpl) Update(id uint, m model.Organization) error {
	return r.db.Conn().Model(&model.Organization{}).Where("organization_id=?", id).Updates(&m).Error
}

func (r *OrganizationRepoImpl) List() ([]model.Organization, error) {
	var ms []model.Organization
	tx := r.db.Conn().Model(&model.Organization{}).Find(&ms)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return ms, nil
}

func (r *OrganizationRepoImpl) Get(id string) (*model.Organization, error) {
	var m model.Organization
	tx := r.db.Conn().Model(&model.Organization{}).Where("organization_id=?", id).First(&m)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &m, nil
}
