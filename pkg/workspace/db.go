package workspace

import (
	"fmt"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/postgres"
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
	if err := orm.AutoMigrate(&Workspace{}); err != nil {
		return nil, fmt.Errorf("gorm migrate: %w", err)
	}
	return &Database{orm: orm}, nil
}

func (s *Database) CreateWorkspace(m *Workspace) error {
	return s.orm.Model(&Workspace{}).Create(m).Error
}

func (s *Database) UpdateWorkspaceStatus(id uuid.UUID, status WorkspaceStatus) error {
	return s.orm.Model(&Workspace{}).Where("id = ?", id).Update("status", status.String()).Error
}

func (s *Database) DeleteWorkspace(id uuid.UUID) error {
	return s.orm.Where("id = ?", id).Unscoped().Delete(&Workspace{}).Error
}

func (s *Database) GetWorkspace(id uuid.UUID) (*Workspace, error) {
	var workspace Workspace
	if err := s.orm.Model(&Workspace{}).Where(Workspace{ID: id}).First(&workspace).Error; err != nil {
		return nil, err
	}
	return &workspace, nil
}

func (s *Database) GetWorkspaceByName(name string) (*Workspace, error) {
	var workspace Workspace
	if err := s.orm.Model(&Workspace{}).Where(Workspace{Name: name}).First(&workspace).Error; err != nil {
		return nil, err
	}
	return &workspace, nil
}

func (s *Database) ListWorkspacesByOwner(ownerId uuid.UUID) ([]*Workspace, error) {
	var workspaces []*Workspace
	if err := s.orm.Model(&Workspace{}).Where(Workspace{OwnerId: ownerId}).Find(&workspaces).Error; err != nil {
		return nil, err
	}
	return workspaces, nil
}

func (s *Database) ListWorkspaces() ([]*Workspace, error) {
	var workspaces []*Workspace
	if err := s.orm.Model(&Workspace{}).Find(&workspaces).Error; err != nil {
		return nil, err
	}
	return workspaces, nil
}

func (s *Database) ListWorkspacesByStatus(status string) ([]*Workspace, error) {
	var workspaces []*Workspace
	if err := s.orm.Model(&Workspace{}).Where(Workspace{Status: status}).Find(&workspaces).Error; err != nil {
		return nil, err
	}
	return workspaces, nil
}

func (s *Database) UpdateWorkspaceOwner(workspaceUUID uuid.UUID, newOwnerID string) error {
	return s.orm.Model(&Workspace{}).Where("id = ?", workspaceUUID).Update("owner_id", newOwnerID).Error
}
