package api

import (
	"time"
)

type Role string

const (
	AdminRole  Role = "ADMIN"
	EditorRole Role = "EDITOR"
	ViewerRole Role = "VIEWER"
)

type ErrorResponse struct {
	Message string `json:"message"`
}

type PutRoleBindingRequest struct {
	UserID string `json:"userId" validate:"required"`
	Role   Role   `json:"role" validate:"required"`
}

type GetRoleBindingRequest struct {
	UserID string `json:"userId" validate:"required"`
}

type RoleBinding struct {
	UserID     string    `json:"userId"`
	Name       string    `json:"name"`
	Emails     []string  `json:"emails"`
	Role       Role      `json:"role"`
	AssignedAt time.Time `json:"assignedAt"`
}

type GetRoleBindingResponse RoleBinding

type GetRoleBindingsResponse []RoleBinding

type DeleteRoleBindingRequest struct {
	UserID string `json:"userId" validate:"required"`
}
