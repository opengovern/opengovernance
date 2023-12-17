package db

import (
	"errors"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/config"
	"strconv"
	"strings"

	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"gorm.io/gorm/clause"

	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Database struct {
	Orm *gorm.DB
}

func NewDatabase(settings config.Config, logger *zap.Logger) (*Database, error) {
	cfg := postgres.Config{
		Host:    settings.Postgres.Host,
		Port:    settings.Postgres.Port,
		User:    settings.Postgres.Username,
		Passwd:  settings.Postgres.Password,
		DB:      settings.Postgres.DB,
		SSLMode: settings.Postgres.SSLMode,
	}
	orm, err := postgres.NewClient(&cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new postgres client: %w", err)
	}
	if err := orm.AutoMigrate(&Organization{}, &Workspace{}, &Credential{}, &MasterCredential{}, &WorkspaceTransaction{}); err != nil {
		return nil, fmt.Errorf("gorm migrate: %w", err)
	}
	return &Database{Orm: orm}, nil
}

func (s *Database) CreateWorkspace(m *Workspace) error {
	return s.Orm.Model(&Workspace{}).Create(m).Error
}

func (s *Database) GetReservedWorkspace() (*Workspace, error) {
	var workspace Workspace
	if err := s.Orm.Model(&Workspace{}).Preload(clause.Associations).
		Where("status = ? OR status = ?", api.StateID_Reserved, api.StateID_Reserving).
		First(&workspace).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &workspace, nil
}

func (s *Database) UpdateWorkspace(m *Workspace) error {
	return s.Orm.Model(&Workspace{}).Where("id = ?", m.ID).Updates(m).Error
}

func (s *Database) UpdateWorkspaceStatus(id string, status api.StateID) error {
	return s.Orm.Model(&Workspace{}).Where("id = ?", id).Update("status", status).Error
}

func (s *Database) UpdateWorkspaceOpenSearchEndpoint(id string, openSearchEndpoint string) error {
	return s.Orm.Model(&Workspace{}).Where("id = ?", id).Update("open_search_endpoint", openSearchEndpoint).Error
}

func (s *Database) SetWorkspaceCreated(id string) error {
	return s.Orm.Model(&Workspace{}).Where("id = ?", id).Update("is_created", true).Error
}

func (s *Database) SetWorkspaceBootstrapInputFinished(id string) error {
	return s.Orm.Model(&Workspace{}).Where("id = ?", id).Update("is_bootstrap_input_finished", true).Error
}

func (s *Database) DeleteWorkspace(id string) error {
	return s.Orm.Where("id = ?", id).Unscoped().Delete(&Workspace{}).Error
}

func (s *Database) GetWorkspace(id string) (*Workspace, error) {
	var workspace Workspace
	if err := s.Orm.Model(&Workspace{}).Preload(clause.Associations).Where(Workspace{ID: id}).First(&workspace).Error; err != nil {
		return nil, err
	}
	return &workspace, nil
}

func (s *Database) GetWorkspaceByName(name string) (*Workspace, error) {
	var workspace Workspace
	if err := s.Orm.Model(&Workspace{}).Preload(clause.Associations).Where(Workspace{Name: name}).First(&workspace).Error; err != nil {
		return nil, err
	}
	return &workspace, nil
}

func (s *Database) ListWorkspacesByOwner(ownerId string) ([]*Workspace, error) {
	var workspaces []*Workspace
	if err := s.Orm.Model(&Workspace{}).Preload(clause.Associations).Where(Workspace{OwnerId: &ownerId}).Find(&workspaces).Error; err != nil {
		return nil, err
	}
	return workspaces, nil
}

func (s *Database) ListWorkspaces() ([]*Workspace, error) {
	var workspaces []*Workspace
	if err := s.Orm.Model(&Workspace{}).Preload(clause.Associations).Find(&workspaces).Error; err != nil {
		return nil, err
	}
	return workspaces, nil
}

func (s *Database) UpdateWorkspaceOwner(workspaceUUID string, newOwnerID string) error {
	return s.Orm.Model(&Workspace{}).Where("id = ?", workspaceUUID).Update("owner_id", newOwnerID).Error
}

func (s *Database) SetWorkspaceAnalyticsJobID(workspaceID string, jobID uint) error {
	return s.Orm.Model(&Workspace{}).Where("id = ?", workspaceID).Update("analytics_job_id", jobID).Error
}

func (s *Database) SetWorkspaceInsightsJobIDs(workspaceID string, jobIDs []uint) error {
	var jobIDstr []string
	for _, j := range jobIDs {
		jobIDstr = append(jobIDstr, strconv.FormatInt(int64(j), 10))
	}
	str := strings.Join(jobIDstr, ",")
	return s.Orm.Model(&Workspace{}).Where("id = ?", workspaceID).Update("insight_jobs_id", str).Error
}

func (s *Database) SetWorkspaceComplianceTriggered(workspaceID string) error {
	return s.Orm.Model(&Workspace{}).Where("id = ?", workspaceID).Update("compliance_triggered", true).Error
}

func (s *Database) UpdateWorkspaceName(workspaceUUID string, newName string) error {
	return s.Orm.Model(&Workspace{}).Where("id = ?", workspaceUUID).Update("name", newName).Error
}

func (s *Database) UpdateWorkspaceTier(workspaceUUID string, newTier api.Tier) error {
	return s.Orm.Model(&Workspace{}).Where("id = ?", workspaceUUID).Update("tier", newTier).Error
}

func (s *Database) UpdateWorkspaceOrganization(workspaceUUID string, newOrganizationID uint) error {
	return s.Orm.Model(&Workspace{}).Where("id = ?", workspaceUUID).Update("organization_id", newOrganizationID).Error
}

func (s *Database) CreateOrganization(m *Organization) error {
	return s.Orm.Model(&Organization{}).Create(m).Error
}

func (s *Database) DeleteOrganization(id uint) error {
	return s.Orm.Where("id = ?", id).Unscoped().Delete(&Organization{}).Error
}

func (s *Database) GetOrganization(id uint) (*Organization, error) {
	var organization Organization
	if err := s.Orm.Model(&Organization{}).Where("id = ?", id).First(&organization).Error; err != nil {
		return nil, err
	}
	return &organization, nil
}

func (s *Database) ListOrganizations() ([]*Organization, error) {
	var organizations []*Organization
	if err := s.Orm.Model(&Organization{}).Find(&organizations).Error; err != nil {
		return nil, err
	}
	return organizations, nil
}

func (s *Database) UpdateOrganization(newOrganization Organization) error {
	return s.Orm.Model(&Organization{}).Where("id = ?", newOrganization.ID).Updates(newOrganization).Error
}

func (s *Database) UpdateCredentialWSID(prevId string, newID string) error {
	return s.Orm.Model(&Credential{}).Where("workspace_id = ?", prevId).Update("workspace_id", newID).Error
}

func (s *Database) UpdateWorkspaceAWSUser(workspaceID string, arn *string) error {
	return s.Orm.Model(&Workspace{}).Where("id = ?", workspaceID).Update("aws_user_arn", arn).Error

}

func (s *Database) HandleDeletedWorkspaces() error {
	return s.Orm.Where("status = ?", api.StateID_Deleted).Unscoped().Delete(&Workspace{}).Error
}
