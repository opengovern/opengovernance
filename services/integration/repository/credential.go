package repository

import (
	"context"
	"errors"

	"github.com/kaytu-io/kaytu-engine/services/integration/db"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"gorm.io/gorm/clause"
)

var ErrDuplicateCredential = errors.New("didn't create credential due to id conflict")

type Credential interface {
	Create(context.Context, *model.Credential) error
}

type CredentialSQL struct {
	db db.Database
}

func NewCredentialSQL(db db.Database) Credential {
	return CredentialSQL{
		db: db,
	}
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
