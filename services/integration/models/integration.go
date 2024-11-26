package models

import (
	"database/sql"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgtype"
	"github.com/opengovern/og-util/pkg/integration"
	api "github.com/opengovern/opengovernance/services/integration/api/models"
	"time"
)

type IntegrationState string

const (
	IntegrationStateActive   IntegrationState = "ACTIVE"
	IntegrationStateInactive IntegrationState = "INACTIVE"
	IntegrationStateArchived IntegrationState = "ARCHIVED"
	IntegrationStateSample   IntegrationState = "SAMPLE_INTEGRATION"
)

type Integration struct {
	IntegrationID   uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"` // Auto-generated UUID
	ProviderID      string
	Name            string
	IntegrationType integration.Type
	Annotations     pgtype.JSONB
	Labels          pgtype.JSONB

	CredentialID uuid.UUID `gorm:"not null"` // Foreign key to Credential

	State     IntegrationState
	LastCheck *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime `gorm:"index"`
}

func (i *Integration) AddLabel(key, value string) (*pgtype.JSONB, error) {
	var labels map[string]string
	if i.Labels.Status == pgtype.Present {
		if err := json.Unmarshal(i.Labels.Bytes, &labels); err != nil {
			return nil, err
		}
	} else {
		labels = make(map[string]string)
	}

	labels[key] = value

	labelsJsonData, err := json.Marshal(labels)
	if err != nil {
		return nil, err
	}
	integrationLabelsJsonb := pgtype.JSONB{}
	err = integrationLabelsJsonb.Set(labelsJsonData)
	i.Labels = integrationLabelsJsonb

	return &integrationLabelsJsonb, nil
}

func (i *Integration) AddAnnotations(key, value string) (*pgtype.JSONB, error) {
	var annotation map[string]string
	if i.Annotations.Status == pgtype.Present {
		if err := json.Unmarshal(i.Annotations.Bytes, &annotation); err != nil {
			return nil, err
		}
	} else {
		annotation = make(map[string]string)
	}

	annotation[key] = value

	annotationsJsonData, err := json.Marshal(annotation)
	if err != nil {
		return nil, err
	}
	integrationAnnotationsJsonb := pgtype.JSONB{}
	err = integrationAnnotationsJsonb.Set(annotationsJsonData)
	i.Annotations = integrationAnnotationsJsonb

	return &integrationAnnotationsJsonb, nil
}

func (i *Integration) ToApi() (*api.Integration, error) {
	var labels map[string]string
	if i.Labels.Status == pgtype.Present {
		if err := json.Unmarshal(i.Labels.Bytes, &labels); err != nil {
			return nil, err
		}
	}

	var annotations map[string]string
	if i.Annotations.Status == pgtype.Present {
		if err := json.Unmarshal(i.Annotations.Bytes, &annotations); err != nil {
			return nil, err
		}
	}

	return &api.Integration{
		IntegrationID:   i.IntegrationID.String(),
		Name:            i.Name,
		ProviderID:      i.ProviderID,
		IntegrationType: i.IntegrationType,
		CredentialID:    i.CredentialID.String(),
		State:           api.IntegrationState(i.State),
		LastCheck:       i.LastCheck,
		Labels:          labels,
		Annotations:     annotations,
	}, nil
}
