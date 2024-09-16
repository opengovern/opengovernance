package repository

import (
	"context"
	"errors"

	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/open-governance/services/integration/db"
	"github.com/kaytu-io/open-governance/services/integration/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrDuplicateCredential = errors.New("didn't create credential due to id conflict")
	ErrCredentialNotFound  = errors.New("cannot find the given credential")
)

type Credential interface {
	Get(context.Context, string) (*model.Credential, error)
	Create(context.Context, *model.Credential) error
	Update(context.Context, *model.Credential) error
	ListByFilters(
		context.Context,
		source.Type,
		source.HealthStatus,
		[]model.CredentialType,
	) ([]model.Credential, error)
}

type CredentialSQL struct {
	db db.Database
}

func NewCredentialSQL(db db.Database) Credential {
	return CredentialSQL{
		db: db,
	}
}

func (c CredentialSQL) Get(ctx context.Context, id string) (*model.Credential, error) {
	cred := new(model.Credential)

	if err := c.db.DB.WithContext(ctx).Find(cred, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCredentialNotFound
		}

		return nil, err
	}

	return cred, nil
}

func (c CredentialSQL) Create(ctx context.Context, cred *model.Credential) error {
	tx := c.db.DB.
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(cred)

	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected == 0 {
		return ErrDuplicateCredential
	}

	return nil
}

func (c CredentialSQL) Update(ctx context.Context, creds *model.Credential) error {
	tx := c.db.DB.WithContext(ctx).
		Model(&model.Credential{}).
		Where("id = ?", creds.ID.String()).Updates(creds)

	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected != 1 {
		return ErrCredentialNotFound
	}

	return nil
}

func (c CredentialSQL) ListByFilters(
	ctx context.Context,
	connector source.Type,
	health source.HealthStatus,
	credentialType []model.CredentialType,
) ([]model.Credential, error) {
	var creds []model.Credential

	tx := c.db.DB.WithContext(ctx)

	if connector != source.Nil {
		tx = tx.Where("connector_type = ?", connector)
	}

	if health != source.HealthStatusNil {
		tx = tx.Where("health_status = ?", health)
	}

	if len(credentialType) > 0 {
		tx = tx.Where("credential_type IN ?", credentialType)
	}

	tx = tx.Find(&creds)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return creds, nil
}
