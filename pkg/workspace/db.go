package workspace

import (
	"fmt"

	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"gorm.io/gorm/clause"

	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Database struct {
	orm *gorm.DB
}

func NewDatabase(settings *Config, logger *zap.Logger) (*Database, error) {
	cfg := postgres.Config{
		Host:    settings.Host,
		Port:    settings.Port,
		User:    settings.User,
		Passwd:  settings.Password,
		DB:      settings.DBName,
		SSLMode: settings.SSLMode,
	}
	orm, err := postgres.NewClient(&cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new postgres client: %w", err)
	}
	if err := orm.AutoMigrate(&Organization{}, &Workspace{}); err != nil {
		return nil, fmt.Errorf("gorm migrate: %w", err)
	}
	return &Database{orm: orm}, nil
}

func (s *Database) CreateWorkspace(m *Workspace) error {
	return s.orm.Model(&Workspace{}).Create(m).Error
}

func (s *Database) UpdateWorkspaceStatus(id string, status api.WorkspaceStatus) error {
	return s.orm.Model(&Workspace{}).Where("id = ?", id).Update("status", status.String()).Error
}

func (s *Database) DeleteWorkspace(id string) error {
	return s.orm.Where("id = ?", id).Unscoped().Delete(&Workspace{}).Error
}

func (s *Database) GetWorkspace(id string) (*Workspace, error) {
	var workspace Workspace
	if err := s.orm.Model(&Workspace{}).Preload(clause.Associations).Where(Workspace{ID: id}).First(&workspace).Error; err != nil {
		return nil, err
	}
	return &workspace, nil
}

func (s *Database) GetWorkspaceByName(name string) (*Workspace, error) {
	var workspace Workspace
	if err := s.orm.Model(&Workspace{}).Preload(clause.Associations).Where(Workspace{Name: name}).First(&workspace).Error; err != nil {
		return nil, err
	}
	return &workspace, nil
}

func (s *Database) ListWorkspacesByOwner(ownerId string) ([]*Workspace, error) {
	var workspaces []*Workspace
	if err := s.orm.Model(&Workspace{}).Preload(clause.Associations).Where(Workspace{OwnerId: ownerId}).Find(&workspaces).Error; err != nil {
		return nil, err
	}
	return workspaces, nil
}

func (s *Database) ListWorkspaces() ([]*Workspace, error) {
	var workspaces []*Workspace
	if err := s.orm.Model(&Workspace{}).Preload(clause.Associations).Find(&workspaces).Error; err != nil {
		return nil, err
	}
	return workspaces, nil
}

func (s *Database) ListWorkspacesByStatus(status api.WorkspaceStatus) ([]*Workspace, error) {
	var workspaces []*Workspace
	if err := s.orm.Model(&Workspace{}).Preload(clause.Associations).Where(Workspace{Status: status}).Find(&workspaces).Error; err != nil {
		return nil, err
	}
	return workspaces, nil
}

func (s *Database) UpdateWorkspaceOwner(workspaceUUID string, newOwnerID string) error {
	return s.orm.Model(&Workspace{}).Where("id = ?", workspaceUUID).Update("owner_id", newOwnerID).Error
}

func (s *Database) UpdateWorkspaceName(workspaceUUID string, newName string) error {
	return s.orm.Model(&Workspace{}).Where("id = ?", workspaceUUID).Update("name", newName).Error
}

func (s *Database) UpdateWorkspaceTier(workspaceUUID string, newTier api.Tier) error {
	return s.orm.Model(&Workspace{}).Where("id = ?", workspaceUUID).Update("tier", newTier).Error
}

func (s *Database) UpdateWorkspaceOrganization(workspaceUUID string, newOrganizationID uint) error {
	return s.orm.Model(&Workspace{}).Where("id = ?", workspaceUUID).Update("organization_id", newOrganizationID).Error
}

func (s *Database) CreateOrganization(m *Organization) error {
	return s.orm.Model(&Organization{}).Create(m).Error
}

func (s *Database) DeleteOrganization(id uint) error {
	return s.orm.Where("id = ?", id).Unscoped().Delete(&Organization{}).Error
}

func (s *Database) GetOrganization(id uint) (*Organization, error) {
	var organization Organization
	if err := s.orm.Model(&Organization{}).Where("id = ?", id).First(&organization).Error; err != nil {
		return nil, err
	}
	return &organization, nil
}

func (s *Database) ListOrganizations() ([]*Organization, error) {
	var organizations []*Organization
	if err := s.orm.Model(&Organization{}).Find(&organizations).Error; err != nil {
		return nil, err
	}
	return organizations, nil
}

func (s *Database) UpdateOrganization(newOrganization Organization) error {
	return s.orm.Model(&Organization{}).Where("id = ?", newOrganization.ID).Updates(newOrganization).Error
}
