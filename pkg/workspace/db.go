package workspace

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Database struct {
	db *gorm.DB
}

type Workspace struct {
	gorm.Model

	WorkspaceId string    `json:"workspace_id"`
	Name        uuid.UUID `json:"name"`
	OwnerId     string    `json:"owner_id"`
	Domain      string    `json:"domain"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
}

func (s *Database) Open(dns string) error {
	db, err := gorm.Open(postgres.Open(dns), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("gorm open: %w", err)
	}
	s.db = db

	return db.AutoMigrate()
}

func (s *Database) CreateWorkspace(m *Workspace) error {
	return s.db.Model(&Workspace{}).Create(m).Error
}

func (s *Database) UpdateWorkspaceStatus(workspaceId, status string) error {
	return s.db.Model(&Workspace{}).Where("workspace_id = ?", workspaceId).Update("status", status).Error
}

func (s *Database) GetWorkspace(workspaceId string) (*Workspace, error) {
	var workspace Workspace
	if err := s.db.Model(&Workspace{}).Where(Workspace{WorkspaceId: workspaceId}).Scan(&workspace).Error; err != nil {
		return nil, err
	}
	return &workspace, nil
}

func (s *Database) ListWorkspacesByOwner(ownerId string) ([]Workspace, error) {
	var workspaces []Workspace
	if err := s.db.Model(&Workspace{}).Where(Workspace{OwnerId: ownerId}).Scan(&workspaces).Error; err != nil {
		return nil, err
	}
	return workspaces, nil
}

func (s *Database) ListWorkspacesByStatus(status string) ([]Workspace, error) {
	var workspaces []Workspace
	if err := s.db.Model(&Workspace{}).Where(Workspace{Status: status}).Scan(&workspaces).Error; err != nil {
		return nil, err
	}
	return workspaces, nil
}
