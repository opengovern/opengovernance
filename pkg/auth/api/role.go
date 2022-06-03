package api

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	AdminRole  Role = "ADMIN"
	EditorRole Role = "EDITOR"
	ViewerRole Role = "VIEWER"
)

type PutRoleBindingRequest struct {
	UserID uuid.UUID `json:"userId" validate:"required"`
	Role   Role      `json:"role" validate:"required"`
}

type RoleBinding struct {
	WorkspaceName string    `json:"workspaceName"`
	Role          Role      `json:"role"`
	AssignedAt    time.Time `json:"assignedAt"`
}

type GetRoleBindingResponse RoleBinding

type GetRoleBindingsResponse []RoleBinding

type WorkspaceRoleBinding struct {
	UserID     uuid.UUID `json:"userId"`
	Role       Role      `json:"role"`
	AssignedAt time.Time `json:"assignedAt"`
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
