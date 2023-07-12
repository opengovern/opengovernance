package resourcebuilder

import (
	describe "github.com/kaytu-io/kaytu-engine/pkg/describe/es"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
)

type Builder interface {
	Process(resource describe.LookupResource)
	Build() []kafka.Doc
	Cleanup(summarizeJobID uint) error
}
