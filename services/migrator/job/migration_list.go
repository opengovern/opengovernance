package job

import (
	// "github.com/opengovern/opengovernance/services/migrator/job/migrations/analytics"
	// "github.com/opengovern/opengovernance/services/migrator/job/migrations/compliance"
	"github.com/opengovern/opengovernance/services/migrator/job/migrations/elasticsearch"
	// "github.com/opengovern/opengovernance/services/migrator/job/migrations/inventory"
	// "github.com/opengovern/opengovernance/services/migrator/job/migrations/onboard"
	// "github.com/opengovern/opengovernance/services/migrator/job/migrations/resource_collection"
	// "github.com/opengovern/opengovernance/services/migrator/job/migrations/resource_info"
	// "github.com/opengovern/opengovernance/services/migrator/job/migrations/workspace"
	"github.com/opengovern/opengovernance/services/migrator/job/types"
		// "github.com/opengovern/opengovernance/services/migrator/job/migrations/auth"

)

var migrations = map[string]types.Migration{
	// "workspace":           workspace.Migration{},
	// "onboard":             onboard.Migration{},
	// "inventory":           inventory.Migration{},
	// "resource_collection": resource_collection.Migration{},
	"elasticsearch":       elasticsearch.Migration{},
	// "compliance":          compliance.Migration{},
	// "analytics":           analytics.Migration{},
	// "resource_info":       resource_info.Migration{},
	// "auth":              auth.Migration{},
}
