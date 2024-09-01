package repo

import (
	"context"
	"errors"
	"github.com/kaytu-io/kaytu-engine/services/information/db/model"
	"gorm.io/gorm"
)

type CspmUsageRepo interface {
	Create(ctx context.Context, m *model.CspmUsage) error
	Update(ctx context.Context, id string, m *model.CspmUsage) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]model.CspmUsage, error)
	Get(ctx context.Context, id string) (*model.CspmUsage, error)
	GetLatestByWorkspaceIdAndHostname(ctx context.Context, workspaceId, hostname string) (*model.CspmUsage, error)
}

type CspmUsageRepoImpl struct {
	db *gorm.DB
}

func NewCspmUsageRepo(db *gorm.DB) CspmUsageRepo {
	return &CspmUsageRepoImpl{
		db: db,
	}
}

func (r *CspmUsageRepoImpl) Create(ctx context.Context, m *model.CspmUsage) error {
	return r.db.WithContext(ctx).Create(&m).Error
}

func (r *CspmUsageRepoImpl) Update(ctx context.Context, id string, m *model.CspmUsage) error {
	return r.db.WithContext(ctx).Model(&model.CspmUsage{}).Where("id=?", id).Updates(&m).Error
}

func (r *CspmUsageRepoImpl) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&model.CspmUsage{}).Where("id=?", id).Delete(&model.CspmUsage{}).Error
}

func (r *CspmUsageRepoImpl) List(ctx context.Context) ([]model.CspmUsage, error) {
	var ms []model.CspmUsage
	tx := r.db.WithContext(ctx).Model(&model.CspmUsage{}).Find(&ms)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return ms, nil
}

func (r *CspmUsageRepoImpl) Get(ctx context.Context, id string) (*model.CspmUsage, error) {
	var m model.CspmUsage
	tx := r.db.WithContext(ctx).Model(&model.CspmUsage{}).Where("id=?", id).First(&m)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &m, nil
}

func (r *CspmUsageRepoImpl) GetLatestByWorkspaceIdAndHostname(ctx context.Context, workspaceId, hostname string) (*model.CspmUsage, error) {
	var m model.CspmUsage
	tx := r.db.WithContext(ctx).Model(&model.CspmUsage{}).Where("workspace_id=? AND hostname=?", workspaceId, hostname).Order("gather_timestamp DESC").First(&m)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &m, nil
}
