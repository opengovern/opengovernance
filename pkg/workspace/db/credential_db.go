package db

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CredentialDatabase struct {
	orm *gorm.DB
}

func NewCredentialDatabase(settings *postgres.Config, logger *zap.Logger) (*CredentialDatabase, error) {
	orm, err := postgres.NewClient(settings, logger)
	if err != nil {
		return nil, fmt.Errorf("new postgres client: %w", err)
	}
	return &CredentialDatabase{orm: orm}, nil
}

func (db CredentialDatabase) GetCredentialByID(id uuid.UUID) (*Credential, error) {
	var cred Credential
	err := db.orm.Model(&Credential{}).
		Where("id = ?", id).
		Find(&cred).Error
	if err != nil {
		return nil, err
	}
	return &cred, nil
}

func (db CredentialDatabase) CreateCredential(cred *Credential) error {
	err := db.orm.Model(&Credential{}).
		Create(cred).Error
	if err != nil {
		return err
	}
	return nil
}
