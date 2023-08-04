package onboard

import (
	"fmt"

	"github.com/kaytu-io/kaytu-engine/pkg/migrator/db"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func Run(logger *zap.Logger, conf postgres.Config, folder string) error {
	conf.DB = "onboard"
	orm, err := postgres.NewClient(&conf, logger)
	if err != nil {
		logger.Error("failed to create postgres client", zap.Error(err))
		return fmt.Errorf("new postgres client: %w", err)
	}
	dbm := db.Database{ORM: orm}

	parser := GitParser{}
	err = parser.ExtractConnectionGroups(folder)
	if err != nil {
		return err
	}

	err = dbm.ORM.Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&onboard.ConnectionGroup{}).Where("1 = 1").Unscoped().Delete(&onboard.ConnectionGroup{}).Error
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
