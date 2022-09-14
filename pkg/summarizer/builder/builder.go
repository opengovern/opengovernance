package builder

import (
	describe "gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"
)

type Builder interface {
	Process(resource describe.LookupResource)
	PopulateHistory() error
	Build() []kafka.Doc
}
