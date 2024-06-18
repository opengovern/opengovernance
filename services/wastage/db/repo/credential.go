package repo

import (
	"errors"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/connector"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"gorm.io/gorm"
)

type CredentialRepo interface {
	Create(m *model.Credential) error
	Get(id uint) (*model.Credential, error)
	Update(id uint, m model.Credential) error
	Delete(id uint) error
	List() ([]model.Credential, error)
}

type CredentialRepoImpl struct {
	db *connector.Database
}

func NewCredentialRepo(db *connector.Database) CredentialRepo {
	return &CredentialRepoImpl{
		db: db,
	}
}

func (r *CredentialRepoImpl) Create(m *model.Credential) error {
	return r.db.Conn().Create(&m).Error
}

func (r *CredentialRepoImpl) Get(id uint) (*model.Credential, error) {
	var m model.Credential
	tx := r.db.Conn().Model(&model.Credential{}).Where("id=?", id).First(&m)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &m, nil
}

func (r *CredentialRepoImpl) Update(id uint, m model.Credential) error {
	return r.db.Conn().Model(&model.Credential{}).Where("id=?", id).Updates(&m).Error
}

func (r *CredentialRepoImpl) Delete(id uint) error {
	return r.db.Conn().Unscoped().Delete(&model.Credential{
		Model: gorm.Model{
			ID: id,
		},
	}).Error
}

func (r *CredentialRepoImpl) List() ([]model.Credential, error) {
	var ms []model.Credential
	tx := r.db.Conn().Model(&model.Credential{}).Find(&ms)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return ms, nil
}
