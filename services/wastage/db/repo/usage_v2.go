package repo

import (
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/connector"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
)

type UsageV2Repo interface {
	Create(m *model.UsageV2) error
	Update(id uint, m model.UsageV2) error
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
