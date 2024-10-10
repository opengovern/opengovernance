package db

import (
	"errors"
	"fmt"
	"github.com/jackc/pgtype"
	"github.com/kaytu-io/kaytu-util/pkg/api"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type Database struct {
	Orm *gorm.DB
}

func (db Database) Initialize() error {
	err := db.Orm.AutoMigrate(
		&ApiKey{},
		&WorkspaceMap{},
		&User{},
		&Configuration{},
	)
	if err != nil {
		return err
	}

	return nil
}

func (db Database) GetKeyPair() ([]Configuration, error) {
	var s []Configuration
	tx := db.Orm.Model(&Configuration{}).
		Where("key = 'private_key' or key = 'public_key'").Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) AddConfiguration(c *Configuration) error {
	tx := db.Orm.Create(c)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) ListApiKeys(workspaceID string) ([]ApiKey, error) {
	var s []ApiKey
	tx := db.Orm.Model(&ApiKey{}).
		Where("workspace_id", workspaceID).
		Where("revoked", "false").
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) ListApiKeysForUser(userId string) ([]ApiKey, error) {
	var s []ApiKey
	tx := db.Orm.Model(&ApiKey{}).
		Where("creator_user_id", userId).
		Where("revoked", "false").
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) GetAPIKeysByRole(role api.Role, workspaceID string) ([]ApiKey, error) {
	var s []ApiKey
	tx := db.Orm.Model(&ApiKey{}).
		Where("workspace_id", workspaceID).
		Where("role", role).
		Where("revoked", "false").
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) CountApiKeys(workspaceID string) (int64, error) {
	var s int64
	tx := db.Orm.Model(&ApiKey{}).
		Where("workspace_id", workspaceID).
		Where("revoked", "false").
		Count(&s)
	if tx.Error != nil {
		return 0, tx.Error
	}
	return s, nil
}

func (db Database) CountApiKeysForUser(userID string) (int64, error) {
	var s int64
	tx := db.Orm.Model(&ApiKey{}).
		Where("creator_user_id", userID).
		Where("revoked", "false").
		Count(&s)
	if tx.Error != nil {
		return 0, tx.Error
	}
	return s, nil
}

func (db Database) GetApiKey(workspaceID string, id uint) (*ApiKey, error) {
	var s ApiKey
	tx := db.Orm.Model(&ApiKey{}).
		Where("workspace_id", workspaceID).
		Where("id", id).
		Where("revoked", "false").
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &s, nil
}

func (db Database) AddApiKey(key *ApiKey) error {
	tx := db.Orm.Create(key)

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) RevokeAPIKey(workspaceID string, id uint) error {
	tx := db.Orm.Model(&ApiKey{}).
		Where("workspace_id", workspaceID).
		Where("id", id).
		Updates(ApiKey{Revoked: true})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) RevokeUserAPIKey(userID string, id uint) error {
	tx := db.Orm.Model(&ApiKey{}).
		Where("creator_user_id", userID).
		Where("id", id).
		Updates(ApiKey{Revoked: true})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) UpdateActiveAPIKey(workspaceID string, id uint, value bool) error {
	tx := db.Orm.Model(&ApiKey{}).
		Where("workspace_id", workspaceID).
		Where("id", id).
		Update("active", value)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) UpdateAPIKeyRole(workspaceID string, id uint, role api.Role) error {
	tx := db.Orm.Model(&ApiKey{}).
		Where("workspace_id", workspaceID).
		Where("id", id).
		Update("role", role)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) UpsertWorkspaceMap(workspaceID string, name string) error {
	tx := db.Orm.Model(&WorkspaceMap{}).Clauses(
		clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"name"}),
		}).Create(&WorkspaceMap{ID: workspaceID, Name: name})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) ListWorkspaceMaps() ([]WorkspaceMap, error) {
	var s []WorkspaceMap
	tx := db.Orm.Model(&WorkspaceMap{}).
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) GetWorkspaceMapByID(workspaceID string) (*WorkspaceMap, error) {
	var s WorkspaceMap
	tx := db.Orm.Model(&WorkspaceMap{}).
		Where("id", workspaceID).
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &s, nil
}

func (db Database) GetWorkspaceMapByName(name string) (*WorkspaceMap, error) {
	var s WorkspaceMap
	tx := db.Orm.Model(&WorkspaceMap{}).
		Where("name", name).
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &s, nil
}

func (db Database) DeleteWorkspaceMapByID(id string) error {
	tx := db.Orm.Model(&WorkspaceMap{}).
		Where("id", id).
		Delete(&WorkspaceMap{})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) CreateUser(user *User) error {
	tx := db.Orm.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"user_uuid", "created_at", "updated_at", "email", "email_verified",
			"user_id", "role", "connector_id", "external_id",
			"user_metadata", "last_login", "app_metadata", "username", "phone_number", "phone_verified", "multifactor", "blocked"}),
	}).Create(user)

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) DeleteUser(userId string) error {
	tx := db.Orm.
		Where("user_id = ?", userId).
		Delete(&User{})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) DeleteUserWithEmail(emailAddress string) error {
	tx := db.Orm.
		Where("email = ?", emailAddress).
		Delete(&User{})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) GetUser(userId string) (*User, error) {
	var s User
	tx := db.Orm.Model(&User{}).
		Where("user_id = ?", userId).
		Find(&s)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	return &s, nil
}

func (db Database) UpdateUserAppMetadataAndLastLogin(userId string, metadata pgtype.JSONB, lastLogin *time.Time) error {
	tx := db.Orm.Model(&User{}).
		Where("user_id = ?", userId).
		Update("app_metadata", metadata)

	if lastLogin != nil {
		tx = tx.Update("last_login", lastLogin)
	}

	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) GetUsersByEmail(email string) ([]User, error) {
	var s []User
	tx := db.Orm.Model(&User{}).
		Where("email = ?", email).
		Find(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return s, nil
}

func (db Database) GetUserByEmail(email string) (*User, error) {
	var s User
	tx := db.Orm.Model(&User{}).
		Where("email = ?", email).
		First(&s)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &s, nil
}

func (db Database) GetUsersByWorkspace(ws string) ([]User, error) {
	var users []User
	query := fmt.Sprintf("SELECT * FROM users WHERE encode(app_metadata, 'escape')::jsonb->'workspaceAccess' ? '%s' AND deleted_at IS NULL", ws)
	err := db.Orm.Raw(query).Scan(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (db Database) SearchUsers(ws string, email *string, emailVerified *bool) ([]User, error) {
	var users []User
	query := fmt.Sprintf("SELECT * FROM users WHERE encode(app_metadata, 'escape')::jsonb->'workspaceAccess' ? '%s' AND deleted_at IS NULL", ws)

	if email != nil {
		query += fmt.Sprintf(" AND email = %s", *email)
	}

	if emailVerified != nil {
		query += fmt.Sprintf(" AND email_verified = %v", *emailVerified)
	}

	err := db.Orm.Raw(query).Scan(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (db Database) UserPasswordUpdated(email string) error {
	tx := db.Orm.Model(&User{}).
		Where("email = ?", email).
		Update("require_password_change", false)

	if tx.Error != nil {
		return tx.Error
	}
	return nil
}
