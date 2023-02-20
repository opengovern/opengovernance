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
	UserID string `json:"userId" validate:"required"`
	Role   Role   `json:"role" validate:"required"`
}

type UserRoleBinding struct {
	WorkspaceID string `json:"workspaceID"`
	Role        Role   `json:"role"`
}

type GetRoleBindingResponse UserRoleBinding

type GetRoleBindingsResponse struct {
	RoleBindings []UserRoleBinding `json:"roleBindings"`
	GlobalRoles  *Role             `json:"globalRoles"`
}

type Membership struct {
	WorkspaceID   string    `json:"workspaceID"`
	WorkspaceName string    `json:"workspaceName"`
	Role          Role      `json:"role"`
	AssignedAt    time.Time `json:"assignedAt"`
	LastActivity  time.Time `json:"lastActivity"`
}

type InviteStatus string

const (
	InviteStatus_ACCEPTED InviteStatus = "ACCEPTED"
	InviteStatus_PENDING  InviteStatus = "PENDING"
)

type WorkspaceRoleBinding struct {
	UserID       string       `json:"userId"`
	UserName     string       `json:"userName"`
	Email        string       `json:"email"`
	Role         Role         `json:"role"`
	Status       InviteStatus `json:"status"`
	LastActivity time.Time    `json:"lastActivity"`
}

type GetWorkspaceRoleBindingResponse []WorkspaceRoleBinding

type DeleteRoleBindingRequest struct {
	UserID string `json:"userId" validate:"required"`
}

type InviteRequest struct {
	Email string `json:"email" validate:"required,email"`
	Role  Role   `json:"role"`
}

type RoleBinding struct {
	UserID        string `json:"userId"`
	WorkspaceName string `json:"workspaceName"`
	Role          Role   `json:"role"`
}
