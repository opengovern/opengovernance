package db

import (
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db/model"
	"gorm.io/gorm"
)

type Database struct {
	ORM *gorm.DB
}

func (db Database) Initialize() error {
	return db.ORM.AutoMigrate(&model.ComplianceJob{}, &model.ComplianceRunner{}, &model.InsightJob{}, &model.CheckupJob{},
		&model.AnalyticsJob{}, &model.Stack{}, &model.StackTag{},
		&model.StackEvaluation{}, &model.StackCredential{}, &model.DescribeConnectionJob{},
		&model.JobSequencer{},
	)
}
