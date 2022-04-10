package auth

import (
	"fmt"

	"gorm.io/gorm"
)

// Database is the to be used for interacting with the Auth Service database.
type Database struct {
	orm *gorm.DB
}

// Initialize created the required tables and schema in the database.
func (db Database) Initialize() error {
	err := db.orm.AutoMigrate(
		&RoleBinding{},
	)
	if err != nil {
		return err
	}

	return nil
}

// GetUserRoleBinding returns the role binding for a user with the given userId.
func (db Database) GetUserRoleBinding(userId string) (RoleBinding, error) {
	var rb RoleBinding
	tx := db.orm.Model(&RoleBinding{}).Where(RoleBinding{UserID: userId}).First(&rb)
	if tx.Error != nil {
		return RoleBinding{}, tx.Error
	}

	return rb, nil
}

// GetAllUserRoleBindings returns the list of all role bindings in the database.
func (db Database) GetAllUserRoleBindings() ([]RoleBinding, error) {
	var rbs []RoleBinding
	tx := db.orm.Model(&RoleBinding{}).Find(&rbs)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return rbs, nil
}

// UpdateRoleBinding updates the role binding for the specified userId.
func (db Database) UpdateRoleBinding(rb *RoleBinding) error {
	tx := db.orm.Model(&RoleBinding{}).
		Where(RoleBinding{UserID: rb.UserID}).
		Updates(rb)
	if tx.Error != nil {
		return tx.Error
	} else if tx.RowsAffected != 1 {
		return fmt.Errorf("update role binding: user with id %s doesn't exist", rb.UserID)
	}

	return nil
}

// GetRoleBindingOrCreate gets a role binding for a user or create it if it doesn't exit.
func (db Database) GetRoleBindingOrCreate(rb *RoleBinding) error {
	tx := db.orm.Model(&RoleBinding{}).Where(RoleBinding{UserID: rb.UserID}).FirstOrCreate(rb)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
