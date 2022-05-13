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

func NewDatabase(settings *Config) (*Database, error) {
	dns := fmt.Sprintf(`host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=GMT`,
		settings.Host,
		settings.Port,
		settings.User,
		settings.Password,
		settings.DBName,
	)

	db, err := gorm.Open(postgres.Open(dns), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("gorm open: %w", err)
	}
	if err := db.AutoMigrate(&Workspace{}); err != nil {
		return nil, fmt.Errorf("gorm migrate: %w", err)
	}
	return &Database{db: db}, nil
}

func (s *Database) CreateWorkspace(m *Workspace) error {
	return s.db.Model(&Workspace{}).Create(m).Error
}

func (s *Database) UpdateWorkspaceStatus(workspaceId uuid.UUID, status WorkspaceStatus) error {
	return s.db.Model(&Workspace{}).Where("workspace_id = ?", workspaceId).Update("status", status.String()).Error
}

func (s *Database) GetWorkspace(workspaceId uuid.UUID) (*Workspace, error) {
	var workspace Workspace
	if err := s.db.Model(&Workspace{}).Where(Workspace{WorkspaceId: workspaceId}).First(&workspace).Error; err != nil {
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
