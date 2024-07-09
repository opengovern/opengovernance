package repo

import (
	"context"
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
	GetByAccountID(endpoint, accountId, auth0UserId, id string) ([]uint, error)
	GetLastByAccountID(endpoint, accountId, auth0UserId, groupByType, byType string) ([]uint, error)
	GetCostZero() (*model.UsageV2, error)
	GetRDSInstanceOptimizationsCountForUser(ctx context.Context, userId string) (int64, error)
	GetRDSInstanceOptimizationsCountForOrg(ctx context.Context, orgAddress string) (int64, error)
	GetRDSClusterOptimizationsCountForUser(ctx context.Context, userId string) (int64, error)
	GetRDSClusterOptimizationsCountForOrg(ctx context.Context, orgAddress string) (int64, error)
	GetEC2InstanceOptimizationsCountForUser(ctx context.Context, userId string) (int64, error)
	GetEC2InstanceOptimizationsCountForOrg(ctx context.Context, orgAddress string) (int64, error)
	GetAccountsForUser(ctx context.Context, userId string) ([]string, error)
	GetAccountsForOrg(ctx context.Context, orgAddress string) ([]string, error)
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

func (r *UsageV2RepoImpl) GetByAccountID(endpoint, accountId, auth0UserId, randomID string) ([]uint, error) {
	tx := r.db.Conn().Raw(fmt.Sprintf(`
SELECT 
  id
FROM 
  usage_v2 
WHERE 
  api_endpoint like '%s%%' and 
  (statistics ->> 'auth0UserId') = '%s' and
  (request -> 'identification' ->> 'randomID') = '%s' and
  (statistics ->> 'accountID') = '%s'
`, endpoint, auth0UserId, randomID, accountId))
	rows, err := tx.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uint
	for rows.Next() {
		var id uint
		err = rows.Scan(&id)
		if err != nil {
			return nil, err
		}

		ids = append(ids, id)
	}

	return ids, nil
}

func (r *UsageV2RepoImpl) GetLastByAccountID(endpoint, accountId, auth0UserId, randomID, groupByType string) ([]uint, error) {
	tx := r.db.Conn().Raw(fmt.Sprintf(`
SELECT 
  max(id)
FROM 
  usage_v2 
WHERE 
  api_endpoint like '%s%%' and 
  (statistics ->> 'auth0UserId') = '%s' and
  (request -> 'identification' ->> 'randomID') = '%s' and
  (statistics ->> 'accountID') = '%s'
GROUP BY request -> '%s' ->> 'id'
`, endpoint, auth0UserId, accountId, randomID, groupByType))
	rows, err := tx.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uint
	for rows.Next() {
		var id uint
		err = rows.Scan(&id)
		if err != nil {
			return nil, err
		}

		ids = append(ids, id)
	}

	return ids, nil
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

func (r *UsageV2RepoImpl) GetRDSInstanceOptimizationsCountForUser(ctx context.Context, userId string) (int64, error) {
	var count int64
	err := r.db.Conn().WithContext(ctx).
		Raw(`
			SELECT COUNT(*) 
			FROM usage_v2 
			WHERE api_endpoint = 'aws-rds' 
			AND statistics ->> 'auth0UserId' = ? 
			AND request ->> 'loading' <> 'true'
		`, userId).
		Scan(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *UsageV2RepoImpl) GetRDSInstanceOptimizationsCountForOrg(ctx context.Context, orgAddress string) (int64, error) {
	var count int64
	err := r.db.Conn().WithContext(ctx).
		Raw(`
			SELECT COUNT(*) 
			FROM usage_v2 
			WHERE api_endpoint = 'aws-rds' 
			AND statistics ->> 'orgEmail' LIKE ? 
			AND request ->> 'loading' <> 'true'
		`, fmt.Sprintf("%%@%s", orgAddress)).
		Scan(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *UsageV2RepoImpl) GetRDSClusterOptimizationsCountForUser(ctx context.Context, userId string) (int64, error) {
	var count int64
	err := r.db.Conn().WithContext(ctx).
		Raw(`
			SELECT COUNT(*) 
			FROM usage_v2 
			WHERE api_endpoint = 'aws-rds-cluster' 
			AND statistics ->> 'auth0UserId' = ? 
			AND request ->> 'loading' <> 'true'
		`, userId).
		Scan(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *UsageV2RepoImpl) GetRDSClusterOptimizationsCountForOrg(ctx context.Context, orgAddress string) (int64, error) {
	var count int64
	err := r.db.Conn().WithContext(ctx).
		Raw(`
			SELECT COUNT(*) 
			FROM usage_v2 
			WHERE api_endpoint = 'aws-rds-cluster' 
			AND statistics ->> 'orgEmail' LIKE ? 
			AND request ->> 'loading' <> 'true'
		`, fmt.Sprintf("%%@%s", orgAddress)).
		Scan(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *UsageV2RepoImpl) GetEC2InstanceOptimizationsCountForUser(ctx context.Context, userId string) (int64, error) {
	var count int64
	err := r.db.Conn().WithContext(ctx).
		Raw(`
			SELECT COUNT(*) 
			FROM usage_v2 
			WHERE api_endpoint = 'ec2-instance' 
			AND statistics ->> 'auth0UserId' = ? 
			AND request ->> 'loading' <> 'true'
		`, userId).
		Scan(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *UsageV2RepoImpl) GetEC2InstanceOptimizationsCountForOrg(ctx context.Context, orgAddress string) (int64, error) {
	var count int64
	err := r.db.Conn().WithContext(ctx).
		Raw(`
			SELECT COUNT(*) 
			FROM usage_v2 
			WHERE api_endpoint = 'ec2-instance' 
			AND statistics ->> 'orgEmail' LIKE ? 
			AND request ->> 'loading' <> 'true'
		`, fmt.Sprintf("%%@%s", orgAddress)).
		Scan(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

//func (r *UsageV2RepoImpl) GetEBSVolumeOptimizationsCountForUser(userId string) (int64, error) {
//	var count int64
//	err := r.db.Conn().Model(&model.UsageV2{}).
//		Select("SUM(statistics ->> 'ebsVolumeCount') as total_volumes").
//		Where("api_endpoint = 'ec2-instance'").
//		Where("statistics ->> 'auth0UserId' = ?", userId).
//		Where("request ->> 'loading' <> 'true'").
//		Scan(&count).Error
//	if err != nil {
//		return 0, err
//	}
//	return count, nil
//}

//func (r *UsageV2RepoImpl) GetEBSVolumeOptimizationsCountForOrg(orgAddress string) (int64, error) {
//	var count int64
//	err := r.db.Conn().Model(&model.UsageV2{}).
//		Select("SUM(statistics ->> 'ebsVolumeCount') as total_volumes").
//		Where("api_endpoint = 'ec2-instance'").
//		Where("statistics ->> 'orgEmail' LIKE ?", fmt.Sprintf("%%@%s", orgAddress)).
//		Where("request ->> 'loading' <> 'true'").
//		Scan(&count).Error
//	if err != nil {
//		return 0, err
//	}
//	return count, nil
//}

func (r *UsageV2RepoImpl) GetAccountsForUser(ctx context.Context, userId string) ([]string, error) {
	var accounts []string
	err := r.db.Conn().Model(&model.UsageV2{}).WithContext(ctx).
		Select("distinct(statistics ->> 'accountID') as accounts").
		Where("statistics ->> 'auth0UserId' = ?", userId).
		Where("request ->> 'loading' <> 'true'").
		Scan(&accounts).Error
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

func (r *UsageV2RepoImpl) GetAccountsForOrg(ctx context.Context, orgAddress string) ([]string, error) {
	var accounts []string
	err := r.db.Conn().Model(&model.UsageV2{}).WithContext(ctx).
		Select("distinct(statistics ->> 'accountID') as accounts").
		Where("statistics ->> 'orgEmail' LIKE ?", fmt.Sprintf("%%@%s", orgAddress)).
		Where("request ->> 'loading' <> 'true'").
		Scan(&accounts).Error
	if err != nil {
		return nil, err
	}
	return accounts, nil
}
