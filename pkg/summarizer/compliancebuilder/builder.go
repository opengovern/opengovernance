package compliancebuilder

import (
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/es"
)

type Builder interface {
	Process(resource es.Finding) error
	PopulateHistory(lastDayJobID, lastWeekJobID, lastQuarterJobID, lastYearJobID uint) error
	Build() []kafka.Doc
	Cleanup(scheduleJobID uint) error
}
