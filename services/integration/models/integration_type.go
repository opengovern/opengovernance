package models

import (
	"github.com/opengovern/opengovernance/services/integration/api/models"
)

type IntegrationType struct {
	ID               int64
	Name             string
	Label            string
	Tier             string
	Annotations      map[string]string
	Labels           map[string]string
	ShortDescription string
	Description      string
	Logo             string
	Enabled          bool
}

func (it *IntegrationType) ToApi() (*models.IntegrationType, error) {
	return &models.IntegrationType{
		ID:               it.ID,
		Name:             it.Name,
		Label:            it.Label,
		Description:      it.Description,
		ShortDescription: it.ShortDescription,
		Tier:             it.Tier,
		Logo:             it.Logo,
		Enabled:          it.Enabled,
		Labels:           it.Labels,
		Annotations:      it.Annotations,
	}, nil
}
