package api

import (
	"time"
)

type Role string

const (
	KeibiAdminRole Role = "KEIBI-ADMIN"
	AdminRole      Role = "ADMIN"
	EditorRole     Role = "EDITOR"
	ViewerRole     Role = "VIEWER"
)

type PutRoleBindingRequest struct {
	UserID string `json:"userId" validate:"required"` // Unique identifier for the User
	Role   Role   `json:"role" validate:"required"`   // Name of the role
}

type UserRoleBinding struct {
	WorkspaceID string `json:"workspaceID"` // Unique identifier for the Workspace
	Role        Role   `json:"role"`        // Name of the binding Role
}

type GetRoleBindingResponse UserRoleBinding

type GetRoleBindingsResponse struct {
	RoleBindings []UserRoleBinding `json:"roleBindings"` // List of user roles in each workspace
	GlobalRoles  *Role             `json:"globalRoles"`  // Global Access
}

type Membership struct {
	WorkspaceID   string    `json:"workspaceID"`   // Unique identifier for the workspace
	WorkspaceName string    `json:"workspaceName"` // Name of the Workspace
	Role          Role      `json:"role"`          // Name of the role
	AssignedAt    time.Time `json:"assignedAt"`    // Assignment timestamp in UTC
	LastActivity  time.Time `json:"lastActivity"`  // Last activity timestamp in UTC
}

type InviteStatus string

const (
	InviteStatus_ACCEPTED InviteStatus = "ACCEPTED"
	InviteStatus_PENDING  InviteStatus = "PENDING"
)

type WorkspaceRoleBinding struct {
	UserID        string       `json:"userId"`        // Unique identifier for the user
	UserName      string       `json:"userName"`      // Username
	TenantId      string       `json:"tenantId"`      // Tenant Id
	Email         string       `json:"email"`         // Email address of the user
	EmailVerified bool         `json:"emailVerified"` // Is email verified or not
	Role          Role         `json:"role"`          // Name of the role
	Status        InviteStatus `json:"status"`        // Invite status
	LastActivity  time.Time    `json:"lastActivity"`  // Last activity timestamp in UTC
	CreatedAt     time.Time    `json:"createdAt"`     // Creation timestamp in UTC
	Blocked       bool         `json:"blocked"`       // Is the user blocked or not
}

type GetWorkspaceRoleBindingResponse []WorkspaceRoleBinding // List of Workspace Role Binding objects

type RoleUser struct {
	UserID        string       `json:"userId"`        // Unique identifier for the user
	UserName      string       `json:"userName"`      // Username
	TenantId      string       `json:"tenantId"`      // Tenant Id
	Email         string       `json:"email"`         // Email address of the user
	EmailVerified bool         `json:"emailVerified"` // Is email verified or not
	Role          Role         `json:"role"`          // Name of the role
	Workspaces    []string     `json:"workspaces"`    // A list of workspace ids which the user has the specified role in them
	Status        InviteStatus `json:"status"`        // Invite status
	LastActivity  time.Time    `json:"lastActivity"`  // Last activity timestamp in UTC
	CreatedAt     time.Time    `json:"createdAt"`     // Creation timestamp in UTC
	Blocked       bool         `json:"blocked"`       // Is the user blocked or not
}

type GetRoleUsersResponse []RoleUser // List of Role User objects

type DeleteRoleBindingRequest struct {
	UserID string `json:"userId" validate:"required"` // Unique identifier for the user
}

type InviteRequest struct {
	Email string `json:"email" validate:"required,email"` // User email address
	Role  Role   `json:"role"`                            // Name of the role
}

type RoleBinding struct {
	UserID        string `json:"userId"`        // Unique identifier for the user
	WorkspaceID   string `json:"workspaceID"`   // Unique identifier for the workspace
	WorkspaceName string `json:"workspaceName"` // Name of the workspace
	Role          Role   `json:"role"`          // Name of the binding role
}
