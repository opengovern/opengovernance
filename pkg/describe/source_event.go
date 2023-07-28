package describe

import (
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type SourceAction string

const (
	SourceDelete SourceAction = "DELETE"
)

type SourceEvent struct {
	Action     SourceAction
	SourceID   string
	AccountID  string
	SourceType source.Type
	Secret     string
}
