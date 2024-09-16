package entity

import (
	"github.com/kaytu-io/open-governance/services/wastage/db/model"
	"time"
)

type Organization struct {
	OrganizationId string     `json:"organization_id"`
	PremiumUntil   *time.Time `json:"premium_until"`
}

// ToModel convert to model.Organization
func (o *Organization) ToModel() *model.Organization {
	return &model.Organization{
		OrganizationId: o.OrganizationId,
		PremiumUntil:   o.PremiumUntil,
	}
}
