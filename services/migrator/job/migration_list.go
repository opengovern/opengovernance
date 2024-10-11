package job

import (
	"github.com/kaytu-io/open-governance/services/migrator/job/migrations/analytics"
	"github.com/kaytu-io/open-governance/services/migrator/job/migrations/auth"
	"github.com/kaytu-io/open-governance/services/migrator/job/migrations/compliance"
	"github.com/kaytu-io/open-governance/services/migrator/job/migrations/elasticsearch"
	"github.com/kaytu-io/open-governance/services/migrator/job/migrations/inventory"
	"github.com/kaytu-io/open-governance/services/migrator/job/migrations/onboard"
	"github.com/kaytu-io/open-governance/services/migrator/job/migrations/resource_collection"
	"github.com/kaytu-io/open-governance/services/migrator/job/migrations/resource_info"
	"github.com/kaytu-io/open-governance/services/migrator/job/migrations/workspace"
	"github.com/kaytu-io/open-governance/services/migrator/job/types"
)

var migrations = map[string]types.Migration{
	"workspace":           workspace.Migration{},
	"onboard":             onboard.Migration{},
	"inventory":           inventory.Migration{},
	"resource_collection": resource_collection.Migration{},
	"elasticsearch":       elasticsearch.Migration{},
	"compliance":          compliance.Migration{},
	"analytics":           analytics.Migration{},
	"resource_info":       resource_info.Migration{},
	"auth":                auth.Migration{},
}
