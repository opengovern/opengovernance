package auth

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
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
		&Invitation{},
	)
	if err != nil {
		return err
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
