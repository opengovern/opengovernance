package auth

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func NewDatabase(orm *gorm.DB) Database {
	return Database{
		orm: orm,
	}
}

// Database is the to be used for interacting with the Auth Service database.
type Database struct {
	orm *gorm.DB
}

// Initialize created the required tables and schema in the database.
func (db Database) Initialize() error {
	err := db.orm.AutoMigrate(
		&User{},
		&RoleBinding{},
		&Invitation{},
	)
	if err != nil {
		return err
	}

	return nil
}

// GetRoleBindingsOfUser returns the list of all role bindings for the user.
func (db Database) GetRoleBindingsOfUser(userId uuid.UUID) ([]RoleBinding, error) {
	var rbs []RoleBinding
	tx := db.orm.
		Model(&RoleBinding{}).
		Where(RoleBinding{UserID: userId}).
		Find(&rbs)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return rbs, nil
}

// GetRoleBindingsOfWorkspace returns the list of all role bindings for the host.
func (db Database) GetRoleBindingsOfWorkspace(workspaceName string) ([]RoleBinding, error) {
	var rbs []RoleBinding
	tx := db.orm.
		Model(&RoleBinding{}).
		Where(RoleBinding{WorkspaceName: workspaceName}).
		Find(&rbs)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return rbs, nil
}

// GetRoleBindingForWorkspace returns the role binding in the workspace for the given user.
func (db Database) GetRoleBindingForWorkspace(userID uuid.UUID, workspaceName string) (RoleBinding, error) {
	var rbs RoleBinding
	tx := db.orm.
		Model(&RoleBinding{}).
		Where(RoleBinding{
			UserID:        userID,
			WorkspaceName: workspaceName,
		}).
		First(&rbs)
	if tx.Error != nil {
		return RoleBinding{}, tx.Error
	}

	return rbs, nil
}

// CreateOrUpdateRoleBinding updates the role binding for the specified userId.
func (db Database) CreateOrUpdateRoleBinding(rb *RoleBinding) error {
	tx := db.orm.
		Model(&RoleBinding{}).
		Where(RoleBinding{UserID: rb.UserID, WorkspaceName: rb.WorkspaceName}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "workspace_name"}},
			DoUpdates: clause.AssignmentColumns([]string{"role", "assigned_at"}),
		}).
		Create(rb)
	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected != 1 {
		return fmt.Errorf("update role binding: user with id %s doesn't exist", rb.UserID)
	}

	return nil
}

// UpdateRoleBinding updates the role binding for the specified userId.
func (db Database) UpdateRoleBinding(rb *RoleBinding) error {
	tx := db.orm.
		Model(&RoleBinding{}).
		Where(RoleBinding{UserID: rb.UserID, WorkspaceName: rb.WorkspaceName}).
		Updates(map[string]interface{}{
			"role":        rb.Role,
			"assigned_at": rb.AssignedAt,
		})
	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected != 1 {
		return fmt.Errorf("update role binding: user with id %s doesn't exist", rb.UserID)
	}

	return nil
}

func (db Database) CreateBindingIfNotExists(rb *RoleBinding) error {
	tx := db.orm.
		Model(&RoleBinding{}).
		Where(RoleBinding{UserID: rb.UserID, WorkspaceName: rb.WorkspaceName}).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		Create(rb)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) GetUserByEmail(email string) (User, error) {
	var au User
	tx := db.orm.
		Model(&User{}).
		Where(User{Email: email}).
		First(&au)
	if tx.Error != nil {
		return User{}, tx.Error
	}

	return au, nil
}

func (db Database) GetUserByID(id uuid.UUID) (User, error) {
	var au User
	tx := db.orm.
		Model(&User{}).
		Where(User{ID: id}).
		First(&au)
	if tx.Error != nil {
		return User{}, tx.Error
	}

	return au, nil
}

func (db Database) GetUserByExternalID(extId string) (User, error) {
	var au User
	tx := db.orm.
		Model(&User{}).
		Where(User{ExternalID: extId}).
		First(&au)
	if tx.Error != nil {
		return User{}, tx.Error
	}

	return au, nil
}

func (db Database) CreateUser(user *User) error {
	tx := db.orm.
		Create(user)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) CreateInvitation(invitation *Invitation) error {
	tx := db.orm.
		Create(invitation)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) GetInvitationByID(invID uuid.UUID) (Invitation, error) {
	var inv Invitation
	tx := db.orm.
		Model(&Invitation{}).
		Where(Invitation{
			ID: invID,
		}).
		First(&inv)
	if tx.Error != nil {
		return Invitation{}, tx.Error
	}

	return inv, nil
}

func (db Database) ListInvitesByWorkspaceName(workspaceName string) ([]Invitation, error) {
	var inv []Invitation
	tx := db.orm.
		Model(&Invitation{}).
		Where(Invitation{
			WorkspaceName: workspaceName,
		}).
		Find(&inv)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return inv, nil
}

func (db Database) DeleteInvitation(invID uuid.UUID) error {
	tx := db.orm.
		Delete(&Invitation{ID: invID})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) CountRoleBindings(workspaceName string) (int64, error) {
	var count int64
	tx := db.orm.Model(&RoleBinding{}).
		Where(RoleBinding{WorkspaceName: workspaceName}).Count(&count)
	if tx.Error != nil {
		return 0, tx.Error
	}
	return count, nil
}
