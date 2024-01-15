package insight

import (
	"github.com/kaytu-io/kaytu-engine/services/migrator/job/migrations/shared"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type Insight struct {
	ID          uint          `json:"ID" yaml:"ID"`
	Query       *shared.Query `json:"Query" yaml:"Query"`
	Connector   source.Type   `json:"Connector" yaml:"Connector"`
	ShortTitle  string        `json:"ShortTitle" yaml:"ShortTitle"`
	LongTitle   string        `json:"LongTitle" yaml:"LongTitle"`
	Description string        `json:"Description" yaml:"Description"`
	LogoURL     *string       `json:"LogoURL" yaml:"LogoURL"`

	Tags     map[string][]string `json:"Tags" yaml:"Tags"`
	Links    []string            `json:"Links" yaml:"Links"`
	Enabled  bool                `json:"Enabled" yaml:"Enabled"`
	Internal bool                `json:"Internal" yaml:"Internal"`
}

type InsightGroup struct {
	ID          uint    `json:"ID" yaml:"ID"`
	ShortTitle  string  `json:"ShortTitle" yaml:"ShortTitle"`
	LongTitle   string  `json:"LongTitle" yaml:"LongTitle"`
	Description string  `json:"Description" yaml:"Description"`
	LogoURL     *string `json:"LogoURL" yaml:"LogoURL"`

	InsightIDs []uint `json:"InsightIDs" yaml:"InsightIDs"`
}
