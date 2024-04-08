package model

import (
	insightapi "github.com/kaytu-io/kaytu-engine/pkg/insight/api"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gorm.io/gorm"
)

type InsightJob struct {
	gorm.Model
	InsightID          uint   `gorm:"index:idx_source_id_insight_id"`
	SourceID           string `gorm:"index:idx_source_id_insight_id"`
	AccountID          string
	SourceType         source.Type
	Status             insightapi.InsightJobStatus
	FailureMessage     string
	ResourceCollection *string
}
