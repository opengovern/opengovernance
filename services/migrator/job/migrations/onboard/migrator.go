package onboard

import (
	"context"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"github.com/kaytu-io/kaytu-engine/services/migrator/config"
	"github.com/kaytu-io/kaytu-engine/services/migrator/db"

	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Migration struct {
}

func (m Migration) IsGitBased() bool {
	return true
}
func (m Migration) AttachmentFolderPath() string {
	return config.ConnectionGroupGitPath
}

func (m Migration) Run(ctx context.Context, conf config.MigratorConfig, logger *zap.Logger) error {
	orm, err := postgres.NewClient(&postgres.Config{
		Host:    conf.PostgreSQL.Host,
		Port:    conf.PostgreSQL.Port,
		User:    conf.PostgreSQL.Username,
		Passwd:  conf.PostgreSQL.Password,
		DB:      "onboard",
		SSLMode: conf.PostgreSQL.SSLMode,
	}, logger)
	if err != nil {
		return fmt.Errorf("new postgres client: %w", err)
	}
	dbm := db.Database{ORM: orm}

	parser := GitParser{}
	err = parser.ExtractConnectionGroups(m.AttachmentFolderPath())
	if err != nil {
		return err
	}

	err = dbm.ORM.Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&model.ConnectionGroup{}).Where("1 = 1").Unscoped().Delete(&model.ConnectionGroup{}).Error
		if err != nil {
			logger.Error("failed to delete connection groups", zap.Error(err))
			return err
		}

		for _, connectionGroup := range parser.connectionGroups {
			err = tx.Clauses(clause.OnConflict{
				DoNothing: true,
			}).Create(&connectionGroup).Error
			if err != nil {
				logger.Error("failed to create connection group", zap.Error(err))
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failure in connection group transaction: %w", err)
	}

	return nil
}
