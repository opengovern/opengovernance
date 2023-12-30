package repository

import (
	"context"

	"github.com/kaytu-io/kaytu-engine/services/integration/db"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"gorm.io/gorm"
)

type CredConn interface {
	DeleteConnection(context.Context, model.Connection) error
	DeleteCredential(context.Context, model.Credential) error
}

type CredConnSQL struct {
	db db.Database
}

func NewCredConnSQL(db db.Database) CredConn {
	return CredConnSQL{db: db}
}

func (c CredConnSQL) DeleteCredential(ctx context.Context, cred model.Credential) error {
	if err := c.db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(new(model.Credential), "WHERE id = ?", cred.ID.String()).Error; err != nil {
			return err
		}

		if err := tx.Delete(new(model.Connection), "WHERE credential_id = ?", cred.ID.String()).Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// DeleteConnection delete the given connection and when connection is deleted, it removes its credential
// if it doesn't have any other connections.
func (c CredConnSQL) DeleteConnection(ctx context.Context, conn model.Connection) error {
	if err := c.db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Where("id = ?", conn.ID.String()).
			Unscoped().
			Delete(new(model.Connection)).Error; err != nil {
			return err
		}

		var count int64

		if err := tx.
			Where("id = ?", conn.CredentialID.String()).
			Count(&count).Error; err != nil {
			return err
		}

		if count == 1 {
			if err := tx.
				Where("id = ?", conn.CredentialID.String()).
				Unscoped().
				Delete(&model.Credential{}).Error; err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}
