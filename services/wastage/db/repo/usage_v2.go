package repo

import (
	"errors"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/connector"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"gorm.io/gorm"
)

type UsageV2Repo interface {
	Create(m *model.UsageV2) error
	Update(id uint, m model.UsageV2) error
	GetRandomNullStatistics() (*model.UsageV2, error)
	Get(id uint) (*model.UsageV2, error)
	GetCostZero() (*model.UsageV2, error)
}

type UsageV2RepoImpl struct {
	db *connector.Database
}

func NewUsageV2Repo(db *connector.Database) UsageV2Repo {
	return &UsageV2RepoImpl{
		db: db,
	}
}

func (r *UsageV2RepoImpl) Create(m *model.UsageV2) error {
	return r.db.Conn().Create(&m).Error
}

func (r *UsageV2RepoImpl) Update(id uint, m model.UsageV2) error {
	return r.db.Conn().Model(&model.UsageV2{}).Where("id=?", id).Updates(&m).Error
}

func (r *UsageV2RepoImpl) Get(id uint) (*model.UsageV2, error) {
	var m model.UsageV2
	tx := r.db.Conn().Model(&model.UsageV2{}).Where("id=?", id).First(&m)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &m, nil
}

func (r *UsageV2RepoImpl) GetRandomNullStatistics() (*model.UsageV2, error) {
	var m model.UsageV2
	tx := r.db.Conn().Model(&model.UsageV2{}).Where("statistics IS NULL").First(&m)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &m, nil
}

func (r *UsageV2RepoImpl) GetCostZero() (*model.UsageV2, error) {
	var m model.UsageV2
	tx := r.db.Conn().Model(&model.UsageV2{}).Where("api_endpoint = 'aws-rds'").
		Where("(response -> 'rightSizing' -> 'current' ->> 'cost')::float = 0 and (response -> 'rightSizing' -> 'recommended' ->> 'cost')::float = 0").
		Where("((response -> 'rightSizing' -> 'current' ->> 'computeCost')::float <> 0 or (response -> 'rightSizing' -> 'current' ->> 'storageCost')::float <> 0)").
		First(&m)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &m, nil
}
