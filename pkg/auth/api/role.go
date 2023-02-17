package api

import (
	"time"

	"github.com/google/uuid"
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

type RoleBinding struct {
	WorkspaceID string    `json:"workspaceID"`
	Role        Role      `json:"role"`
	AssignedAt  time.Time `json:"assignedAt"`
}

type GetRoleBindingResponse RoleBinding

type GetRoleBindingsResponse struct {
	RoleBindings []RoleBinding `json:"roleBindings"`
	GlobalRoles  *Role         `json:"globalRoles"`
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

type InviteUserRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type InviteUserResponse struct {
	UserID uuid.UUID `json:"userId"`
}

type InviteRequest struct {
	Email string `json:"email" validate:"required,email"`
	Role  Role   `json:"role"`
}

type InviteResponse struct {
	InviteID uuid.UUID `json:"inviteId"`
}

type InviteItem struct {
	Email string `json:"email"`
}
