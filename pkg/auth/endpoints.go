package auth

import (
	"net/http"

	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
)

func roleToPriority(role api.Role) int {
	switch role {
	case api.ViewerRole:
		return 0
	case api.EditorRole:
		return 1
	case api.AdminRole:
		return 2
	default:
		panic("unsupported role: " + role)
	}
}

func hasAccess(currRole, minRole api.Role) bool {
	return roleToPriority(currRole) >= roleToPriority(minRole)
}

type endpoint struct {
	Path        string
	Method      string
	MinimumRole api.Role
}

var endpoints = []endpoint{
	// ============ Auth Service ============
	{
		Path:        "/auth/api/v1/role/binding",
		Method:      http.MethodGet,
		MinimumRole: api.AdminRole,
	},
	{
		Path:        "/auth/api/v1/role/binding",
		Method:      http.MethodPut,
		MinimumRole: api.AdminRole,
	},
	{
		Path:        "/auth/api/v1/role/bindings",
		Method:      http.MethodGet,
		MinimumRole: api.AdminRole,
	},
	// ============ Inventory Service ============
	{
		Path:        "/inventory/api/v1/locations/:provider",
		Method:      http.MethodGet,
		MinimumRole: api.ViewerRole,
	},
	{
		Path:        "/inventory/api/v1/query",
		Method:      http.MethodGet,
		MinimumRole: api.ViewerRole,
	},
	{
		Path:        "/inventory/api/v1/query/:query_id",
		Method:      http.MethodPost,
		MinimumRole: api.ViewerRole,
	},
	{
		Path:        "/inventory/api/v1/resource",
		Method:      http.MethodPost,
		MinimumRole: api.ViewerRole,
	},
	{
		Path:        "/inventory/api/v1/resources",
		Method:      http.MethodPost,
		MinimumRole: api.ViewerRole,
	},
	{
		Path:        "/inventory/api/v1/resources/aws",
		Method:      http.MethodPost,
		MinimumRole: api.ViewerRole,
	},
	{
		Path:        "/inventory/api/v1/resources/azure",
		Method:      http.MethodPost,
		MinimumRole: api.ViewerRole,
	},
	{
		Path:        "/inventory/api/v1/reports/compliance/:source_id",
		Method:      http.MethodGet,
		MinimumRole: api.ViewerRole,
	},
	{
		Path:        "/inventory/api/v1/reports/compliance/:source_id/:report_id",
		Method:      http.MethodGet,
		MinimumRole: api.ViewerRole,
	},
	{
		Path:        "/inventory/api/v1/benchmarks",
		Method:      http.MethodGet,
		MinimumRole: api.ViewerRole,
	},
	{
		Path:        "/inventory/api/v1/benchmarks/tags",
		Method:      http.MethodGet,
		MinimumRole: api.ViewerRole,
	},
	{
		Path:        "/inventory/api/v1/benchmarks/:benchmarkId",
		Method:      http.MethodGet,
		MinimumRole: api.ViewerRole,
	},
	{
		Path:        "/inventory/api/v1/benchmarks/:benchmarkId/policies",
		Method:      http.MethodGet,
		MinimumRole: api.ViewerRole,
	},
	{
		Path:        "/inventory/api/v1/benchmarks/:benchmarkId/:sourceId/result",
		Method:      http.MethodGet,
		MinimumRole: api.ViewerRole,
	},
	{
		Path:        "/inventory/api/v1/benchmarks/:benchmarkId/:sourceId/result/policies",
		Method:      http.MethodGet,
		MinimumRole: api.ViewerRole,
	},
	// ============ Onboard Service ============
	{
		Path:        "/onboard/api/v1/source/aws",
		Method:      http.MethodPost,
		MinimumRole: api.EditorRole,
	},
	{
		Path:        "/onboard/api/v1/source/azure",
		Method:      http.MethodPost,
		MinimumRole: api.EditorRole,
	},
	{
		Path:        "/onboard/api/v1/discover/aws/accounts",
		Method:      http.MethodPost,
		MinimumRole: api.ViewerRole,
	},
	{
		Path:        "/onboard/api/v1/discover/azure/subscriptions",
		Method:      http.MethodPost,
		MinimumRole: api.ViewerRole,
	},
	{
		Path:        "/onboard/api/v1/providers",
		Method:      http.MethodGet,
		MinimumRole: api.ViewerRole,
	},
	{
		Path:        "/onboard/api/v1/providers/types",
		Method:      http.MethodGet,
		MinimumRole: api.ViewerRole,
	},
	{
		Path:        "/onboard/api/v1/source/:source_id",
		Method:      http.MethodGet,
		MinimumRole: api.ViewerRole,
	},
	{
		Path:        "/onboard/api/v1/source/:source_id",
		Method:      http.MethodDelete,
		MinimumRole: api.EditorRole,
	},
	// ============ Scheduler Service ============
	{
		Path:        "/schedule/api/v1/resource_type/:provider",
		Method:      http.MethodGet,
		MinimumRole: api.ViewerRole,
	},
	{
		Path:        "/schedule/api/v1/sources",
		Method:      http.MethodGet,
		MinimumRole: api.ViewerRole,
	},
	{
		Path:        "/schedule/api/v1/sources/:source_id/jobs/compliance",
		Method:      http.MethodPost,
		MinimumRole: api.ViewerRole,
	},
	{
		Path:        "/schedule/api/v1/sources/:source_id/jobs/compliance/refresh",
		Method:      http.MethodPost,
		MinimumRole: api.EditorRole,
	},
	{
		Path:        "/schedule/api/v1/sources/:source_id/jobs/describe",
		Method:      http.MethodPost,
		MinimumRole: api.ViewerRole,
	},
	{
		Path:        "/schedule/api/v1/sources/:source_id/jobs/describe/refresh",
		Method:      http.MethodPost,
		MinimumRole: api.EditorRole,
	},
	{
		Path:        "/schedule/api/v1/sources/:source_id/policy/:policy_id",
		Method:      http.MethodGet,
		MinimumRole: api.EditorRole,
	},
}
