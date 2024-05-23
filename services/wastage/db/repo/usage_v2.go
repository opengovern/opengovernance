package repo

import (
	"errors"
	"fmt"
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
	GetRDSInstanceOptimizationsCountForUser(userId string) (int64, error)
	GetRDSInstanceOptimizationsCountForOrg(orgAddress string) (int64, error)
	GetRDSClusterOptimizationsCountForUser(userId string) (int64, error)
	GetRDSClusterOptimizationsCountForOrg(orgAddress string) (int64, error)
	GetEC2InstanceOptimizationsCountForUser(userId string) (int64, error)
	GetEC2InstanceOptimizationsCountForOrg(orgAddress string) (int64, error)
	GetEBSVolumeOptimizationsCountForUser(userId string) (int64, error)
	GetEBSVolumeOptimizationsCountForOrg(orgAddress string) (int64, error)
	GetAccountsForUser(userId string) ([]string, error)
	GetAccountsForOrg(orgAddress string) ([]string, error)
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

func (r *UsageV2RepoImpl) GetRDSInstanceOptimizationsCountForUser(userId string) (int64, error) {
	var count int64
	err := r.db.Conn().Model(&model.UsageV2{}).
		Where("api_endpoint = 'aws-rds'").
		Where("statistics ->> 'auth0UserId' = ?", userId).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *UsageV2RepoImpl) GetRDSInstanceOptimizationsCountForOrg(orgAddress string) (int64, error) {
	var count int64
	err := r.db.Conn().Model(&model.UsageV2{}).
		Where("api_endpoint = 'aws-rds'").
		Where("statistics ->> 'orgEmail' LIKE ?", fmt.Sprintf("%%@%s", orgAddress)).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *UsageV2RepoImpl) GetRDSClusterOptimizationsCountForUser(userId string) (int64, error) {
	var count int64
	err := r.db.Conn().Model(&model.UsageV2{}).
		Where("api_endpoint = 'aws-rds-cluster'").
		Where("statistics ->> 'auth0UserId' = ?", userId).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *UsageV2RepoImpl) GetRDSClusterOptimizationsCountForOrg(orgAddress string) (int64, error) {
	var count int64
	err := r.db.Conn().Model(&model.UsageV2{}).
		Where("api_endpoint = 'aws-rds-cluster'").
		Where("statistics ->> 'orgEmail' LIKE ?", fmt.Sprintf("%%@%s", orgAddress)).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *UsageV2RepoImpl) GetEC2InstanceOptimizationsCountForUser(userId string) (int64, error) {
	var count int64
	err := r.db.Conn().Model(&model.UsageV2{}).
		Where("api_endpoint = 'ec2-instance'").
		Where("statistics ->> 'auth0UserId' = ?", userId).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *UsageV2RepoImpl) GetEC2InstanceOptimizationsCountForOrg(orgAddress string) (int64, error) {
	var count int64
	err := r.db.Conn().Model(&model.UsageV2{}).
		Where("api_endpoint = 'ec2-instance'").
		Where("statistics ->> 'orgEmail' LIKE ?", fmt.Sprintf("%%@%s", orgAddress)).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *UsageV2RepoImpl) GetEBSVolumeOptimizationsCountForUser(userId string) (int64, error) {
	var count int64
	err := r.db.Conn().Model(&model.UsageV2{}).
		Select("SUM(statistics ->> 'ebsVolumeCount') as total_volumes").
		Where("api_endpoint = 'ec2-instance'").
		Where("statistics ->> 'auth0UserId' = ?", userId).
		Scan(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *UsageV2RepoImpl) GetEBSVolumeOptimizationsCountForOrg(orgAddress string) (int64, error) {
	var count int64
	err := r.db.Conn().Model(&model.UsageV2{}).
		Select("SUM(statistics ->> 'ebsVolumeCount') as total_volumes").
		Where("api_endpoint = 'ec2-instance'").
		Where("statistics ->> 'orgEmail' LIKE ?", fmt.Sprintf("%%@%s", orgAddress)).
		Scan(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *UsageV2RepoImpl) GetAccountsForUser(userId string) ([]string, error) {
	var accounts []string
	err := r.db.Conn().Model(&model.UsageV2{}).
		Select("distinct(statistics ->> 'accountID') as accounts").
		Where("api_endpoint = 'ec2-instance'").
		Where("statistics ->> 'auth0UserId' = ?", userId).
		Scan(&accounts).Error
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

func (r *UsageV2RepoImpl) GetAccountsForOrg(orgAddress string) ([]string, error) {
	var accounts []string
	err := r.db.Conn().Model(&model.UsageV2{}).
		Select("distinct(statistics ->> 'accountID') as accounts").
		Where("api_endpoint = 'ec2-instance'").
		Where("statistics ->> 'orgEmail' LIKE ?", fmt.Sprintf("%%@%s", orgAddress)).
		Scan(&accounts).Error
	if err != nil {
		return nil, err
	}
	return accounts, nil
}
