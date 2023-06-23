package resourcebuilder

import (
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	describe "github.com/kaytu-io/kaytu-engine/pkg/describe/es"
)

type Builder interface {
	Process(resource describe.LookupResource)
	PopulateHistory(lastDayJobID, lastWeekJobID, lastQuarterJobID, lastYearJobID uint) error
	Build() []kafka.Doc
	Cleanup(summarizeJobID uint) error
}
