package resourcebuilder

import (
	"github.com/kaytu-io/kaytu-util/pkg/es"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
)

type Builder interface {
	Process(resource es.LookupResource)
	Build() []kafka.Doc
	Cleanup(summarizeJobID uint) error
}
