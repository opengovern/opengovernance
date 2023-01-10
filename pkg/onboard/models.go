package onboard

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	"gorm.io/datatypes"
)

type AssetDiscoveryMethodType string

const (
	AssetDiscoveryMethodTypeScheduled AssetDiscoveryMethodType = "scheduled"
)

type SourceHealthState string

const (
	SourceHealthStateHealthy          SourceHealthState = "healthy"
	SourceHealthStateUnhealthy        SourceHealthState = "unhealthy"
	SourceHealthStateInitialDiscovery SourceHealthState = "initial_discovery"
)

type SourceCreationMethod string

const (
	SourceCreationMethodManual SourceCreationMethod = "manual"
)

type Source struct {
	ID          uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"` // Auto-generated UUID
	SourceId    string    `gorm:"index:idx_source_id,unique"`                      // AWS Account ID, Azure Subscription ID, ...
	Name        string    `gorm:"not null"`
	Email       string
	Type        source.Type `gorm:"not null"`
	Description string
	ConfigRef   string
	Enabled     bool

	AssetDiscoveryMethod    AssetDiscoveryMethodType `gorm:"not null;default:'scheduled'"`
	AssetDiscoveryFrequency int64                    `gorm:"not null;default:21600"` // in seconds

	HealthState SourceHealthState `gorm:"not null;default:'unhealthy'"`

	//Connector Connector `gorm:"foreignKey:Type;references:Code"`

	CreationMethod SourceCreationMethod `gorm:"not null;default:'manual'"`

	Metadata datatypes.JSON

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime `gorm:"index"`
}

func NewAWSSource(accountID, accountName, accountDescription, accountEmail, accountOrganization string) Source {
	id := uuid.New()
	provider := source.CloudAWS

	metadata := map[string]interface{}{
		"account_id": accountID,
	}
	if accountID != accountName {
		metadata["account_name"] = accountName
	}
	if accountOrganization != "" {
		metadata["account_organization"] = accountOrganization
		metadata["is_organization_member"] = true
	}
	if accountDescription != "" {
		metadata["account_description"] = accountDescription
	}
	if accountEmail != "" {
		metadata["account_email"] = accountEmail
	}

	marshalMetadata, err := json.Marshal(metadata)
	if err != nil {
		marshalMetadata = []byte("{}")
	}

	s := Source{
		ID:          id,
		SourceId:    accountID,
		Name:        accountName,
		Email:       accountEmail,
		Description: accountDescription,
		Type:        provider,
		ConfigRef:   fmt.Sprintf("sources/%s/%s", strings.ToLower(string(provider)), id),
		Enabled:     true,
		Metadata:    datatypes.JSON(marshalMetadata),
	}

	if len(strings.TrimSpace(s.Name)) == 0 {
		s.Name = s.SourceId
	}

	return s
}

func NewAzureSource(in api.SourceAzureRequest) Source {
	id := uuid.New()
	provider := source.CloudAzure

	// SPN -> PK: TenantID & ClientID
	s := Source{
		ID:          id,
		SourceId:    in.Config.SubscriptionId,
		Name:        in.Name,
		Description: in.Description,
		Type:        provider,
		ConfigRef:   fmt.Sprintf("sources/%s/%s", strings.ToLower(string(provider)), id),
		Enabled:     true,
	}

	if len(strings.TrimSpace(s.Name)) == 0 {
		s.Name = s.SourceId
	}

	return s
}

func (s Source) toSourceResponse() *api.CreateSourceResponse {
	return &api.CreateSourceResponse{
		ID: s.ID,
	}
}

func NewAzureSourceWithSPN(in api.SourceAzureSPNRequest) Source {
	id := uuid.New()
	provider := source.CloudAzure

	s := Source{
		ID:          id,
		SourceId:    in.SubscriptionId,
		Name:        in.Name,
		Description: in.Description,
		Type:        provider,
		Enabled:     true,
		ConfigRef:   fmt.Sprintf("sources/%s/spn/%s", strings.ToLower(string(provider)), in.SPNId),
	}

	if len(strings.TrimSpace(s.Name)) == 0 {
		s.Name = s.SourceId
	}

	return s
}

type SPN struct {
	ID        uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	TenantId  string    `gorm:"index:idx_tenant_client_id,unique"`
	ClientId  string    `gorm:"index:idx_tenant_client_id,unique"`
	ConfigRef string

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime `gorm:"index"`
}

func (s SPN) toSPNResponse() *api.CreateSPNResponse {
	return &api.CreateSPNResponse{
		ID: s.ID,
	}
}

func NewSPN(in api.CreateSPNRequest) SPN {
	id := uuid.New()
	provider := source.CloudAzure

	return SPN{
		ID:        id,
		ClientId:  in.Config.ClientId,
		TenantId:  in.Config.TenantId,
		ConfigRef: fmt.Sprintf("sources/%s/spn/%s", strings.ToLower(string(provider)), id),
	}
}

type ConnectorDirectionType string

const (
	ConnectorDirectionTypeIngress ConnectorDirectionType = "ingress"
	ConnectorDirectionTypeEgress  ConnectorDirectionType = "egress"
	ConnectorDirectionTypeBoth    ConnectorDirectionType = "both"
)

type Connector struct {
	Code             source.Type `gorm:"primaryKey"`
	Name             string
	Description      string
	Direction        ConnectorDirectionType `gorm:"default:'ingress'"`
	Enabled          bool
	StartSupportDate time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime `gorm:"index"`
}
