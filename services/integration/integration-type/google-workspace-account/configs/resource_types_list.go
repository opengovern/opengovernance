package configs

var TablesToResourceTypes = map[string]string{
	"google_workspace_user":          "GoogleWorkspace/User",
	"google_workspace_user_alias":    "GoogleWorkspace/UserAlias",
	"google_workspace_group":         "GoogleWorkspace/Group",
	"google_workspace_group_member":  "GoogleWorkspace/GroupMember",
	"google_workspace_org_unit":      "GoogleWorkspace/OrgUnit",
	"google_workspace_mobile_device": "GoogleWorkspace/MobileDevice",
	"google_workspace_chrome_device": "GoogleWorkspace/ChromeDevice",
	"google_workspace_role":          "GoogleWorkspace/Role",
}

var ResourceTypesList = []string{
	"GoogleWorkspace/User",
	"GoogleWorkspace/UserAlias",
	"GoogleWorkspace/Group",
	"GoogleWorkspace/GroupMember",
	"GoogleWorkspace/OrgUnit",
	"GoogleWorkspace/MobileDevice",
	"GoogleWorkspace/ChromeDevice",
	"GoogleWorkspace/Role",
}
