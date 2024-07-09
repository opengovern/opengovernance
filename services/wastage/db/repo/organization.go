package repo

import (
	"context"
	"errors"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/connector"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"gorm.io/gorm"
)

type OrganizationRepo interface {
	Create(m *model.Organization) error
	Update(id string, m *model.Organization) error
	Delete(id string) error
	List() ([]model.Organization, error)
	Get(ctx context.Context, id string) (*model.Organization, error)
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

func (r *OrganizationRepoImpl) Update(id string, m *model.Organization) error {
	return r.db.Conn().Model(&model.Organization{}).Where("organization_id=?", id).Updates(&m).Error
}

func (r *OrganizationRepoImpl) Delete(id string) error {
	return r.db.Conn().Model(&model.Organization{}).Where("organization_id=?", id).Delete(&model.Organization{}).Error
}

func (r *OrganizationRepoImpl) List() ([]model.Organization, error) {
	var ms []model.Organization
	tx := r.db.Conn().Model(&model.Organization{}).Find(&ms)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return ms, nil
}

func (r *OrganizationRepoImpl) Get(ctx context.Context, id string) (*model.Organization, error) {
	var m model.Organization
	tx := r.db.Conn().WithContext(ctx).Model(&model.Organization{}).Where("organization_id=?", id).First(&m)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &m, nil
}
