package job

import (
	"github.com/kaytu-io/kaytu-engine/services/migrator/job/migrations/analytics"
	"github.com/kaytu-io/kaytu-engine/services/migrator/job/migrations/compliance"
	"github.com/kaytu-io/kaytu-engine/services/migrator/job/migrations/elasticsearch"
	"github.com/kaytu-io/kaytu-engine/services/migrator/job/migrations/insight"
	"github.com/kaytu-io/kaytu-engine/services/migrator/job/migrations/inventory"
	"github.com/kaytu-io/kaytu-engine/services/migrator/job/migrations/onboard"
	"github.com/kaytu-io/kaytu-engine/services/migrator/job/migrations/resource_collection"
	"github.com/kaytu-io/kaytu-engine/services/migrator/job/migrations/superset"
	"github.com/kaytu-io/kaytu-engine/services/migrator/job/migrations/workspace"
	"github.com/kaytu-io/kaytu-engine/services/migrator/job/types"
)

var migrations = map[string]types.Migration{
	"workspace":           workspace.Migration{},
	"onboard":             onboard.Migration{},
	"inventory":           inventory.Migration{},
	"resource_collection": resource_collection.Migration{},
	"elasticsearch":       elasticsearch.Migration{},
	"insight":             insight.Migration{},
	"compliance":          compliance.Migration{},
	"analytics":           analytics.Migration{},
	"superset":            superset.Migration{},
}
