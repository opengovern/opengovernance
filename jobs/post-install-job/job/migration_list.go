package job

import (
	"github.com/opengovern/opengovernance/jobs/post-install-job/job/migrations/auth"
	"github.com/opengovern/opengovernance/jobs/post-install-job/job/migrations/compliance"
	"github.com/opengovern/opengovernance/jobs/post-install-job/job/migrations/elasticsearch"
	"github.com/opengovern/opengovernance/jobs/post-install-job/job/migrations/integration"
	"github.com/opengovern/opengovernance/jobs/post-install-job/job/migrations/inventory"
	"github.com/opengovern/opengovernance/jobs/post-install-job/job/migrations/metadata"
	"github.com/opengovern/opengovernance/jobs/post-install-job/job/migrations/resource_collection"
	"github.com/opengovern/opengovernance/jobs/post-install-job/job/migrations/resource_info"
	"github.com/opengovern/opengovernance/jobs/post-install-job/job/types"
)

var migrations = map[string]types.Migration{

	"elasticsearch":       elasticsearch.Migration{},
	
}

var manualMigrations =map[string]types.Migration{
	"metadata":            metadata.Migration{},
	"integration":         integration.Migration{},
	"inventory":           inventory.Migration{},
	"resource_collection": resource_collection.Migration{},
	"elasticsearch":       elasticsearch.Migration{},
	"compliance":          compliance.Migration{},
	"resource_info":       resource_info.Migration{},
	"auth":                auth.Migration{},
}
