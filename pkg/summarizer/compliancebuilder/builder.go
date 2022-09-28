package compliancebuilder

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"
)

type Builder interface {
	Process(resource es.Finding)
	PopulateHistory(lastDayJobID, lastWeekJobID, lastQuarterJobID, lastYearJobID uint) error
	Build() []kafka.Doc
	Cleanup(scheduleJobID uint) error
}
