package db

import (
	"github.com/opengovern/opengovernance/pkg/describe/db/model"
	"gorm.io/gorm"
)

type Database struct {
	ORM *gorm.DB
}

func (db Database) Initialize() error {
	return db.ORM.AutoMigrate(&model.ComplianceJob{}, &model.ComplianceSummarizer{}, &model.ComplianceRunner{}, &model.CheckupJob{},
		&model.AnalyticsJob{}, &model.DescribeIntegrationJob{}, &model.IntegrationDiscovery{},
		&model.JobSequencer{}, &model.QueryRunnerJob{}, &model.QueryValidatorJob{},
	)
}
