package compliancebuilder

import (
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
)

type Builder interface {
	Process(resource types.Finding)
	Build() []kafka.Doc
	Cleanup(summarizeJobID uint) error
}
