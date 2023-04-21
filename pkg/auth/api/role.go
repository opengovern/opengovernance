package api

import (
	"time"
)

type Role string

const (
	KeibiAdminRole Role = "keibi-admin"
	AdminRole      Role = "admin"
	EditorRole     Role = "editor"
	ViewerRole     Role = "viewer"
)

type PutRoleBindingRequest struct {
	UserID string `json:"userId" validate:"required"`                                           // Unique identifier for the User
	Role   Role   `json:"role" validate:"required" example:"admin" enums:"admin,editor,viewer"` // Name of the role
}
type RolesListResponse struct {
	Role        Role   `json:"role" example:"admin" enums:"admin,editor,viewer"`
	Description string `json:"description" example:"The Administrator role is a super user role with all of the capabilities that can be assigned to a role, and its enables access to all data & configuration on a Kaytu Workspace. You cannot edit or delete the Administrator role."`
	UserCount   int    `json:"userCount" example:"1"`
}

type RoleDetailsResponse struct {
	Role        Role              `json:"role" example:"admin" enums:"admin,editor,viewer"`
	Description string            `json:"description" example:"The Administrator role is a super user role with all of the capabilities that can be assigned to a role, and its enables access to all data & configuration on a Kaytu Workspace. You cannot edit or delete the Administrator role."`
	UserCount   int               `json:"userCount" example:"1"`
	Users       []GetUserResponse `json:"users"`
}

type UserRoleBinding struct {
	WorkspaceID string `json:"workspaceID"`                      // Unique identifier for the Workspace
	Role        Role   `json:"role" enums:"admin,editor,viewer"` // Name of the binding Role
}

type GetRoleBindingResponse UserRoleBinding

type GetRoleBindingsResponse struct {
	RoleBindings []UserRoleBinding `json:"roleBindings"` // List of user roles in each workspace
	GlobalRoles  *Role             `json:"globalRoles"`  // Global Access
}

type Membership struct {
	WorkspaceID   string    `json:"workspaceID"`                      // Unique identifier for the workspace
	WorkspaceName string    `json:"workspaceName"`                    // Name of the Workspace
	Role          Role      `json:"role" enums:"admin,editor,viewer"` // Name of the role
	AssignedAt    time.Time `json:"assignedAt"`                       // Assignment timestamp in UTC
	LastActivity  time.Time `json:"lastActivity"`                     // Last activity timestamp in UTC
}

type InviteStatus string

const (
	InviteStatus_ACCEPTED InviteStatus = "ACCEPTED"
	InviteStatus_PENDING  InviteStatus = "PENDING"
)

type WorkspaceRoleBinding struct {
	UserID       string       `json:"userId"`                           // Unique identifier for the user
	UserName     string       `json:"userName"`                         // Username
	Email        string       `json:"email"`                            // Email address of the user
	Role         Role         `json:"role" enums:"admin,editor,viewer"` // Name of the role
	Status       InviteStatus `json:"status"`                           // Invite status
	LastActivity time.Time    `json:"lastActivity"`                     // Last activity timestamp in UTC
	CreatedAt    time.Time    `json:"createdAt"`                        // Creation timestamp in UTC
}

type GetWorkspaceRoleBindingResponse []WorkspaceRoleBinding // List of Workspace Role Binding objects

type GetUserResponse struct {
	UserID        string       `json:"userId"`                           // Unique identifier for the user
	UserName      string       `json:"userName"`                         // Username
	Email         string       `json:"email"`                            // Email address of the user
	EmailVerified bool         `json:"emailVerified"`                    // Is email verified or not
	Role          Role         `json:"role" enums:"admin,editor,viewer"` // Name of the role in the specified workspace
	Status        InviteStatus `json:"status"`                           // Invite status
	LastActivity  time.Time    `json:"lastActivity"`                     // Last activity timestamp in UTC
	CreatedAt     time.Time    `json:"createdAt"`                        // Creation timestamp in UTC
	Blocked       bool         `json:"blocked"`                          // Is the user blocked or not
}

type GetUsersResponse []GetUserResponse // List of Workspace Role Binding objects

type GetUsersRequest struct {
	Email         *string `json:"email" example:"sample@gmail.com"`
	EmailVerified *bool   `json:"emailVerified" example:"true"`
	Role          *Role   `json:"role" enums:"admin,editor,viewer" example:"admin"`
}

type RoleUser struct {
	UserID        string       `json:"userId"`                                           // Unique identifier for the user
	UserName      string       `json:"userName"`                                         // Username
	Email         string       `json:"email"`                                            // Email address of the user
	EmailVerified bool         `json:"emailVerified"`                                    // Is email verified or not
	Role          Role         `json:"role" enums:"admin,editor,viewer" example:"admin"` // Name of the role
	Workspaces    []string     `json:"workspaces"`                                       // A list of workspace ids which the user has the specified role in them
	Status        InviteStatus `json:"status"`                                           // Invite status
	LastActivity  time.Time    `json:"lastActivity"`                                     // Last activity timestamp in UTC
	CreatedAt     time.Time    `json:"createdAt"`                                        // Creation timestamp in UTC
	Blocked       bool         `json:"blocked"`                                          // Is the user blocked or not
}

type GetRoleUsersResponse []RoleUser // List of Role User objects

type DeleteRoleBindingRequest struct {
	UserID string `json:"userId" validate:"required"` // Unique identifier for the user
}

type InviteRequest struct {
	Email string `json:"email" validate:"required,email"`                  // User email address
	Role  Role   `json:"role" enums:"admin,editor,viewer" example:"admin"` // Name of the role
}
type RoleBinding struct {
	UserID        string `json:"userId"`                                           // Unique identifier for the user
	WorkspaceID   string `json:"workspaceID"`                                      // Unique identifier for the workspace
	WorkspaceName string `json:"workspaceName"`                                    // Name of the workspace
	Role          Role   `json:"role" enums:"admin,editor,viewer" example:"admin"` // Name of the binding role
}
