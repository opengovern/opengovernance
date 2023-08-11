package api

import (
	"strings"
	"time"
)

type Role string

const (
	InternalRole   Role = "internal"
	KaytuAdminRole Role = "kaytu-admin"
	AdminRole      Role = "admin"
	EditorRole     Role = "editor"
	ViewerRole     Role = "viewer"
)

func GetRole(s string) Role {
	switch strings.TrimSpace(strings.ToLower(s)) {
	case string(InternalRole):
		return InternalRole
	case string(KaytuAdminRole):
		return KaytuAdminRole
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
	UserID   string `json:"userId" validate:"required" example:"auth|123456789"`                      // Unique identifier for the User
	RoleName Role   `json:"roleName" validate:"required" enums:"admin,editor,viewer" example:"admin"` // Name of the role
}
type RolesListResponse struct {
	RoleName    Role   `json:"roleName" enums:"admin,editor,viewer" example:"admin"`                                                                                                                                                                                                      // Name of the role
	Description string `json:"description" example:"The Administrator role is a super user role with all of the capabilities that can be assigned to a role, and its enables access to all data & configuration on a Kaytu Workspace. You cannot edit or delete the Administrator role."` // Role Description and accesses
	UserCount   int    `json:"userCount" example:"1"`                                                                                                                                                                                                                                     // Number of users having this role in the workspace
}

type RoleDetailsResponse struct {
	RoleName    Role               `json:"role" enums:"admin,editor,viewer" example:"admin"`                                                                                                                                                                                                          // Name of the role
	Description string             `json:"description" example:"The Administrator role is a super user role with all of the capabilities that can be assigned to a role, and its enables access to all data & configuration on a Kaytu Workspace. You cannot edit or delete the Administrator role."` // Role Description and accesses
	UserCount   int                `json:"userCount" example:"1"`                                                                                                                                                                                                                                     // Number of users having this role
	Users       []GetUsersResponse `json:"users"`                                                                                                                                                                                                                                                     // List of users having the role
}

type UserRoleBinding struct {
	WorkspaceID string `json:"workspaceID" example:"ws123456789"`                    // Unique identifier for the Workspace
	RoleName    Role   `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the binding Role
}

type GetRoleBindingResponse UserRoleBinding

type GetRoleBindingsResponse struct {
	RoleBindings []UserRoleBinding `json:"roleBindings"`                                            // List of user roles in each workspace
	GlobalRoles  *Role             `json:"globalRoles" enums:"admin,editor,viewer" example:"admin"` // Global Access
}

type Membership struct {
	WorkspaceID   string    `json:"workspaceID" example:"ws123456789"`                    // Unique identifier for the workspace
	WorkspaceName string    `json:"workspaceName" example:"demo"`                         // Name of the Workspace
	RoleName      Role      `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the role
	AssignedAt    time.Time `json:"assignedAt" example:"2023-03-31T09:36:09.855Z"`        // Assignment timestamp in UTC
	LastActivity  time.Time `json:"lastActivity" example:"2023-04-21T08:53:09.928Z"`      // Last activity timestamp in UTC
}

type InviteStatus string

const (
	InviteStatus_ACCEPTED InviteStatus = "accepted"
	InviteStatus_PENDING  InviteStatus = "pending"
)

type WorkspaceRoleBinding struct {
	UserID       string       `json:"userId" example:"auth|123456789"`                      // Unique identifier for the user
	UserName     string       `json:"userName" example:"John Doe"`                          // Username
	Email        string       `json:"email" example:"johndoe@example.com"`                  // Email address of the user
	RoleName     Role         `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the role
	Status       InviteStatus `json:"status" enums:"accepted,pending" example:"accepted"`   // Invite status
	LastActivity time.Time    `json:"lastActivity" example:"2023-04-21T08:53:09.928Z"`      // Last activity timestamp in UTC
	CreatedAt    time.Time    `json:"createdAt" example:"2023-03-31T09:36:09.855Z"`         // Creation timestamp in UTC
}

type GetWorkspaceRoleBindingResponse []WorkspaceRoleBinding // List of Workspace Role Binding objects

type GetUserResponse struct {
	UserID        string       `json:"userId" example:"auth|123456789"`                      // Unique identifier for the user
	UserName      string       `json:"userName" example:"John Doe"`                          // Username
	Email         string       `json:"email" example:"johndoe@example.com"`                  // Email address of the user
	EmailVerified bool         `json:"emailVerified" example:"true"`                         // Is email verified or not
	RoleName      Role         `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the role
	Status        InviteStatus `json:"status" enums:"accepted,pending" example:"accepted"`   // Invite status
	LastActivity  time.Time    `json:"lastActivity" example:"2023-04-21T08:53:09.928Z"`      // Last activity timestamp in UTC
	CreatedAt     time.Time    `json:"createdAt" example:"2023-03-31T09:36:09.855Z"`         // Creation timestamp in UTC
	Blocked       bool         `json:"blocked" example:"false"`                              // Is the user blocked or not
}
type GetUsersResponse struct {
	UserID        string `json:"userId" example:"auth|123456789"`                      // Unique identifier for the user
	UserName      string `json:"userName" example:"John Doe"`                          // Username
	Email         string `json:"email" example:"johndoe@example.com"`                  // Email address of the user
	EmailVerified bool   `json:"emailVerified" example:"true"`                         // Is email verified or not
	RoleName      Role   `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the role
}

type GetUsersRequest struct {
	Email         *string `json:"email" example:"johndoe@example.com"`
	EmailVerified *bool   `json:"emailVerified" example:"true"`                         // Filter by
	RoleName      *Role   `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Filter by role name
}

type RoleUser struct {
	UserID        string       `json:"userId" example:"auth|123456789"`                      // Unique identifier for the user
	UserName      string       `json:"userName" example:"John Doe"`                          // Username
	Email         string       `json:"email" example:"johndoe@example.com"`                  // Email address of the user
	EmailVerified bool         `json:"emailVerified" example:"true"`                         // Is email verified or not
	RoleName      Role         `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the role
	Workspaces    []string     `json:"workspaces" example:"demo"`                            // A list of workspace ids which the user has the specified role in them
	Status        InviteStatus `json:"status" enums:"accepted,pending" example:"accepted"`   // Invite status
	LastActivity  time.Time    `json:"lastActivity" example:"2023-04-21T08:53:09.928Z"`      // Last activity timestamp in UTC
	CreatedAt     time.Time    `json:"createdAt" example:"2023-03-31T09:36:09.855Z"`         // Creation timestamp in UTC
	Blocked       bool         `json:"blocked" example:"false"`                              // Is the user blocked or not
}

type GetRoleUsersResponse []RoleUser // List of Role User objects

type DeleteRoleBindingRequest struct {
	UserID string `json:"userId" validate:"required" example:"auth|123456789"` // Unique identifier for the user
}

type InviteRequest struct {
	Email    string `json:"email" validate:"required,email" example:"johndoe@example.com"` // User email address
	RoleName Role   `json:"roleName" enums:"admin,editor,viewer" example:"admin"`          // Name of the role
}
type RoleBinding struct {
	UserID        string `json:"userId" example:"auth|123456789"`                      // Unique identifier for the user
	WorkspaceID   string `json:"workspaceID" example:"ws123456789"`                    // Unique identifier for the workspace
	WorkspaceName string `json:"workspaceName" example:"demo"`                         // Name of the workspace
	RoleName      Role   `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the binding role
}
