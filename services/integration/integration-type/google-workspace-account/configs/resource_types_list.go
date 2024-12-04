package configs

var TablesToResourceTypes = map[string]string{
	"google_workspace_user":              "GoogleWorkspace/User",
	"google_workspace_user_alias":        "GoogleWorkspace/UserAlias",
	"google_workspace_group":             "GoogleWorkspace/Group",
	"google_workspace_group_member":      "GoogleWorkspace/GroupMember",
	"google_workspace_group_alias":       "GoogleWorkspace/GroupAlias",
	"google_workspace_org_unit":          "GoogleWorkspace/OrgUnit",
	"google_workspace_mobile_device":     "GoogleWorkspace/MobileDevice",
	"google_workspace_chrome_device":     "GoogleWorkspace/ChromeDevice",
	"google_workspace_role":              "GoogleWorkspace/Role",
	"google_workspace_role_assignment":   "GoogleWorkspace/RoleAssignment",
	"google_workspace_domain":            "GoogleWorkspace/Domain",
	"google_workspace_domain_alias":      "GoogleWorkspace/DomainAlias",
	"google_workspace_privilege":         "GoogleWorkspace/Privilege",
	"google_workspace_resource_building": "GoogleWorkspace/ResourceBuilding",
	"google_workspace_resource_calender": "GoogleWorkspace/ResourceCalender",
	"google_workspace_resource_feature":  "GoogleWorkspace/ResourceFeature",
}

var ResourceTypesList = []string{
	"GoogleWorkspace/User",
	"GoogleWorkspace/UserAlias",
	"GoogleWorkspace/Group",
	"GoogleWorkspace/GroupMember",
	"GoogleWorkspace/GroupAlias",
	"GoogleWorkspace/OrgUnit",
	"GoogleWorkspace/MobileDevice",
	"GoogleWorkspace/ChromeDevice",
	"GoogleWorkspace/Role",
	"GoogleWorkspace/RoleAssignment",
	"GoogleWorkspace/Domain",
	"GoogleWorkspace/DomainAlias",
	"GoogleWorkspace/Privilege",
	"GoogleWorkspace/ResourceBuilding",
	"GoogleWorkspace/ResourceCalender",
	"GoogleWorkspace/ResourceFeature",
}
