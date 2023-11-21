package db

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"gorm.io/gorm"
)

type Workspace struct {
	gorm.Model

	ID                       string              `json:"id"`
	Name                     string              `gorm:"uniqueIndex" json:"name"`
	AWSUserARN               *string             `json:"aws_user_arn"`
	OwnerId                  *string             `json:"owner_id"`
	URI                      string              `json:"uri"`
	Status                   api.WorkspaceStatus `json:"status"`
	Description              string              `json:"description"`
	Size                     api.WorkspaceSize   `json:"workspace_size"`
	Tier                     api.Tier            `json:"tier"`
	OrganizationID           *int                `json:"organization_id"`
	Organization             *Organization       `json:"organization" gorm:"foreignKey:OrganizationID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	IsCreated                bool                `json:"is_created"`
	IsBootstrapInputFinished bool                `json:"is_bootstrap_input_finished"`
	AnalyticsJobID           uint                `json:"analytics_job_id"`
	InsightJobsID            string              `json:"insight_jobs_id"`
	ComplianceTriggered      bool                `json:"complianceTriggered"`
}

func (w *Workspace) ToAPI() api.Workspace {
	var org *api.Organization
	if w.Organization != nil {
		v := w.Organization.ToAPI()
		org = &v
	}

	return api.Workspace{
		ID:                       w.ID,
		Name:                     w.Name,
		OwnerId:                  w.OwnerId,
		URI:                      w.URI,
		Status:                   w.Status,
		Description:              w.Description,
		Tier:                     w.Tier,
		Organization:             org,
		Size:                     w.Size,
		CreatedAt:                w.Model.CreatedAt,
		IsCreated:                w.IsCreated,
		IsBootstrapInputFinished: w.IsBootstrapInputFinished,
	}
}

type Organization struct {
	gorm.Model

	CompanyName  string `json:"companyName"`
	Url          string `json:"url"`
	Address      string `json:"address"`
	City         string `json:"city"`
	State        string `json:"state"`
	Country      string `json:"country"`
	ContactPhone string `json:"contactPhone"`
	ContactEmail string `json:"contactEmail"`
	ContactName  string `json:"contactName"`
}

func (o *Organization) ToAPI() api.Organization {
	return api.Organization{
		ID:           o.ID,
		CompanyName:  o.CompanyName,
		Url:          o.Url,
		Address:      o.Address,
		City:         o.City,
		State:        o.State,
		Country:      o.Country,
		ContactPhone: o.ContactPhone,
		ContactEmail: o.ContactEmail,
		ContactName:  o.ContactName,
	}
}
