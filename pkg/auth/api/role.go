package api

import (
	"strings"
	"time"
)

type Role string

const (
	KeibiAdminRole Role = "keibi-admin"
	AdminRole      Role = "admin"
	EditorRole     Role = "editor"
	ViewerRole     Role = "viewer"
)

func GetRole(s string) Role {
	switch strings.TrimSpace(strings.ToLower(s)) {
	case string(KeibiAdminRole):
		return KeibiAdminRole
	case string(AdminRole):
		return AdminRole
	case string(EditorRole):
		return EditorRole
	case string(ViewerRole):
		return ViewerRole
	default:
		return ""
	}

}

type PutRoleBindingRequest struct {
<<<<<<< HEAD
	UserID   string `json:"userId" validate:"required" example:"sampleID"`                            // Unique identifier for the User
	RoleName Role   `json:"roleName" validate:"required" example:"admin" enums:"admin,editor,viewer"` // Name of the role
}
type RolesListResponse struct {
	RoleName    Role   `json:"roleName" example:"admin" enums:"admin,editor,viewer"`                                                                                                                                                                                                      // Name of the role
	Description string `json:"description" example:"The Administrator role is a super user role with all of the capabilities that can be assigned to a role, and its enables access to all data & configuration on a Kaytu Workspace. You cannot edit or delete the Administrator role."` // Role Description and accesses
	UserCount   int    `json:"userCount" example:"1"`                                                                                                                                                                                                                                     // Number of usershaving this role
}

type RoleDetailsResponse struct {
	RoleName    Role               `json:"role" example:"admin" enums:"admin,editor,viewer"`                                                                                                                                                                                                          // Name of the role
	Description string             `json:"description" example:"The Administrator role is a super user role with all of the capabilities that can be assigned to a role, and its enables access to all data & configuration on a Kaytu Workspace. You cannot edit or delete the Administrator role."` // Role Description and accesses
	UserCount   int                `json:"userCount" example:"1"`                                                                                                                                                                                                                                     // Number of users having this role
	Users       []GetUsersResponse `json:"users"`                                                                                                                                                                                                                                                     // List of users having this role
}

type UserRoleBinding struct {
	WorkspaceID string `json:"workspaceID" example:"sampleID"`                       // Unique identifier for the Workspace
=======
	UserID   string `json:"userId" validate:"required"`                                               // Unique identifier for the User
	RoleName Role   `json:"roleName" validate:"required" example:"admin" enums:"admin,editor,viewer"` // Name of the role
}
type RolesListResponse struct {
	RoleName    Role   `json:"roleName" example:"admin" enums:"admin,editor,viewer"` //
	Description string `json:"description" example:"The Administrator role is a super user role with all of the capabilities that can be assigned to a role, and its enables access to all data & configuration on a Kaytu Workspace. You cannot edit or delete the Administrator role."`
	UserCount   int    `json:"userCount" example:"1"`
}

type RoleDetailsResponse struct {
	RoleName    Role               `json:"role" example:"admin" enums:"admin,editor,viewer"`
	Description string             `json:"description" example:"The Administrator role is a super user role with all of the capabilities that can be assigned to a role, and its enables access to all data & configuration on a Kaytu Workspace. You cannot edit or delete the Administrator role."`
	UserCount   int                `json:"userCount" example:"1"`
	Users       []GetUsersResponse `json:"users"`
}

type UserRoleBinding struct {
	WorkspaceID string `json:"workspaceID"`                                          // Unique identifier for the Workspace
>>>>>>> 0c30a1a6b2f64066d9405859ce1968e90c1ad6d9
	RoleName    Role   `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the binding Role
}

type GetRoleBindingResponse UserRoleBinding

type GetRoleBindingsResponse struct {
	RoleBindings []UserRoleBinding `json:"roleBindings"` // List of user roles in each workspace
	GlobalRoles  *Role             `json:"globalRoles"`  // Global Access
}

type Membership struct {
<<<<<<< HEAD
	WorkspaceID   string    `json:"workspaceID" example:"sampleID"`                       // Unique identifier for the workspace
	WorkspaceName string    `json:"workspaceName" example:"demo"`                         // Name of the Workspace
	RoleName      Role      `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the role
	AssignedAt    time.Time `json:"assignedAt" example:"2023-03-31T09:36:09.855Z"`        // Assignment timestamp in UTC
	LastActivity  time.Time `json:"lastActivity" example:"2023-04-21T08:53:09.928Z"`      // Last activity timestamp in UTC
=======
	WorkspaceID   string    `json:"workspaceID"`                                          // Unique identifier for the workspace
	WorkspaceName string    `json:"workspaceName"`                                        // Name of the Workspace
	RoleName      Role      `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the role
	AssignedAt    time.Time `json:"assignedAt"`                                           // Assignment timestamp in UTC
	LastActivity  time.Time `json:"lastActivity"`                                         // Last activity timestamp in UTC
>>>>>>> 0c30a1a6b2f64066d9405859ce1968e90c1ad6d9
}

type InviteStatus string

const (
	InviteStatus_ACCEPTED InviteStatus = "accepted"
	InviteStatus_PENDING  InviteStatus = "pending"
)

type WorkspaceRoleBinding struct {
	UserID       string       `json:"userId" example:"sampleID"`                            // Unique identifier for the user
	UserName     string       `json:"userName" example:"sampleName"`                        // Username
	Email        string       `json:"email" example:"sample@gmail.com"`                     // Email address of the user
	RoleName     Role         `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the role
	Status       InviteStatus `json:"status" enums:"accepted,pending" example:"pending"`    // Invite status
<<<<<<< HEAD
	LastActivity time.Time    `json:"lastActivity" example:"2023-04-21T08:53:09.928Z"`      // Last activity timestamp in UTC
	CreatedAt    time.Time    `json:"createdAt" example:"2023-03-31T09:36:09.855Z"`         // Creation timestamp in UTC
=======
	LastActivity time.Time    `json:"lastActivity"`                                         // Last activity timestamp in UTC
	CreatedAt    time.Time    `json:"createdAt"`                                            // Creation timestamp in UTC
>>>>>>> 0c30a1a6b2f64066d9405859ce1968e90c1ad6d9
}

type GetWorkspaceRoleBindingResponse []WorkspaceRoleBinding // List of Workspace Role Binding objects

type GetUserResponse struct {
	UserID        string       `json:"userId" example:"sampleID"`                            // Unique identifier for the user
	UserName      string       `json:"userName" example:"sampleName"`                        // Username
	Email         string       `json:"email" example:"sample@gmail.com"`                     // Email address of the user
	EmailVerified bool         `json:"emailVerified" example:"true"`                         // Is email verified or not
	RoleName      Role         `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the role
	Status        InviteStatus `json:"status" enums:"accepted,pending" example:"pending"`    // Invite status
<<<<<<< HEAD
	LastActivity  time.Time    `json:"lastActivity" example:"2023-04-21T08:53:09.928Z"`      // Last activity timestamp in UTC
	CreatedAt     time.Time    `json:"createdAt" example:"2023-03-31T09:36:09.855Z"`         // Creation timestamp in UTC
=======
	LastActivity  time.Time    `json:"lastActivity"`                                         // Last activity timestamp in UTC
	CreatedAt     time.Time    `json:"createdAt"`                                            // Creation timestamp in UTC
>>>>>>> 0c30a1a6b2f64066d9405859ce1968e90c1ad6d9
	Blocked       bool         `json:"blocked" example:"false"`                              // Is the user blocked or not
}

type GetUsersResponse struct {
	UserID        string `json:"userId" example:"sampleID"`                            // Unique identifier for the user
	UserName      string `json:"userName" example:"sampleName"`                        // Username
	Email         string `json:"email" example:"sample@gmail.com"`                     // Email address of the user
	EmailVerified bool   `json:"emailVerified" example:"true"`                         // Is email verified or not
	RoleName      Role   `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the role
}

type GetUsersRequest struct {
	Email         *string `json:"email" example:"sample@gmail.com"`
	EmailVerified *bool   `json:"emailVerified" example:"true"`                         // Filter by
	RoleName      *Role   `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Filter by role name
}

type RoleUser struct {
	UserID        string       `json:"userId" example:"sampleID"`                            // Unique identifier for the user
	UserName      string       `json:"userName" example:"sampleName"`                        // Username
	Email         string       `json:"email" example:"sample@gmail.com"`                     // Email address of the user
	EmailVerified bool         `json:"emailVerified" example:"true"`                         // Is email verified or not
	RoleName      Role         `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the role
<<<<<<< HEAD
	Workspaces    []string     `json:"workspaces" example:"demo"`                            // A list of workspace ids which the user has the specified role in them
	Status        InviteStatus `json:"status" enums:"accepted,pending" example:"pending"`    // Invite status
	LastActivity  time.Time    `json:"lastActivity" example:"2023-04-21T08:53:09.928Z"`      // Last activity timestamp in UTC
	CreatedAt     time.Time    `json:"createdAt" example:"2023-03-31T09:36:09.855Z"`         // Creation timestamp in UTC
=======
	Workspaces    []string     `json:"workspaces" example:"demoWorkspace"`                   // A list of workspace ids which the user has the specified role in them
	Status        InviteStatus `json:"status" enums:"accepted,pending" example:"pending"`    // Invite status
	LastActivity  time.Time    `json:"lastActivity"`                                         // Last activity timestamp in UTC
	CreatedAt     time.Time    `json:"createdAt"`                                            // Creation timestamp in UTC
>>>>>>> 0c30a1a6b2f64066d9405859ce1968e90c1ad6d9
	Blocked       bool         `json:"blocked" example:"false"`                              // Is the user blocked or not
}

type GetRoleUsersResponse []RoleUser // List of Role User objects

type DeleteRoleBindingRequest struct {
	UserID string `json:"userId" validate:"required" example:"sampleID"` // Unique identifier for the user
}

type InviteRequest struct {
	Email    string `json:"email" validate:"required,email" example:"sample@gmail.com"` // User email address
	RoleName Role   `json:"roleName" enums:"admin,editor,viewer" example:"admin"`       // Name of the role
}
type RoleBinding struct {
<<<<<<< HEAD
	UserID        string `json:"userId" example:"sampleID"`                            // Unique identifier for the user
	WorkspaceID   string `json:"workspaceID" example:"sampleID"`                       // Unique identifier for the workspace
	WorkspaceName string `json:"workspaceName" example:"demo"`                         // Name of the workspace
=======
	UserID        string `json:"userId"`                                               // Unique identifier for the user
	WorkspaceID   string `json:"workspaceID"`                                          // Unique identifier for the workspace
	WorkspaceName string `json:"workspaceName"`                                        // Name of the workspace
>>>>>>> 0c30a1a6b2f64066d9405859ce1968e90c1ad6d9
	RoleName      Role   `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the binding role
}
