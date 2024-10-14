package api

import (
	"github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/opengovernance/services/integration/api/entity"
	"time"
)

type CreateAPIKeyRequest struct {
	Name     string   `json:"name"`                                                 // Name of the key
	RoleName api.Role `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the role
}

type CreateAPIKeyResponse struct {
	ID        uint      `json:"id" example:"1"`                                       // Unique identifier for the key
	Name      string    `json:"name" example:"example"`                               // Name of the key
	Active    bool      `json:"active" example:"true"`                                // Activity state of the key
	CreatedAt time.Time `json:"createdAt" example:"2023-03-31T09:36:09.855Z"`         // Creation timestamp in UTC
	RoleName  api.Role  `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the role
	Token     string    `json:"token"`                                                // Token of the key
}

type WorkspaceApiKey struct {
	ID            uint      `json:"id" example:"1"`                                       // Unique identifier for the key
	CreatedAt     time.Time `json:"createdAt" example:"2023-03-31T09:36:09.855Z"`         // Creation timestamp in UTC
	UpdatedAt     time.Time `json:"updatedAt" example:"2023-04-21T08:53:09.928Z"`         // Last update timestamp in UTC
	Name          string    `json:"name" example:"example"`                               // Name of the key
	RoleName      api.Role  `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the role
	CreatorUserID string    `json:"creatorUserID" example:"auth|123456789"`               // Unique identifier of the user who created the key
	Active        bool      `json:"active" example:"true"`                                // Activity state of the key
	MaskedKey     string    `json:"maskedKey" example:"abc...de"`                         // Masked key
}

type UpdateKeyRoleRequest struct {
	ID       uint     `json:"id"`                                                   // Unique identifier for the key
	RoleName api.Role `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the role
}

type CreateUserRequest struct {
	EmailAddress string    `json:"email_address"`
	Role         *api.Role `json:"role" enums:"admin,editor,viewer" example:"admin"`
	Password     *string   `json:"password"`
}

type UpdateUserRequest struct {
	EmailAddress string    `json:"email_address"`
	Role         *api.Role `json:"role" enums:"admin,editor,viewer" example:"admin"`
	Password     *string   `json:"password"`
}

type SetupRequest struct {
	CreateUser struct {
		EmailAddress string `json:"email_address"`
		Password     string `json:"password"`
	} `json:"create_user"`
	IncludeSampleData bool                          `json:"include_sample_data"`
	AwsCredentials    *entity.AWSCredentialConfig   `json:"aws_credentials"`
	AzureCredentials  *entity.AzureCredentialConfig `json:"azure_credentials"`
}

type SetupResponse struct {
	CreatedUser      bool    `json:"created_user"`
	Metadata         string  `json:"metadata"`
	SampleDataImport *string `json:"sample_data_import"`
	AwsTriggerID     *string `json:"aws_trigger_id"`
	AzureTriggerID   *string `json:"azure_trigger_id"`
}

type CheckRequest struct {
	AwsCredentials   *entity.AWSCredentialConfig   `json:"aws_credentials"`
	AzureCredentials *entity.AzureCredentialConfig `json:"azure_credentials"`
}

type ResetUserPasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}
