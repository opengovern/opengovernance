package insight

import "github.com/kaytu-io/kaytu-util/pkg/source"

type Insight struct {
	ID          uint
	QueryID     string
	Connector   source.Type
	ShortTitle  string
	LongTitle   string
	Description string
	LogoURL     *string

	Tags     map[string][]string
	Links    []string
	Enabled  bool
	Internal bool
}
