package utils

import (
	"errors"
	"github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/opengovernance/pkg/authV2/db"
	"time"
)

type Service struct {
	domain       string
	clientID     string
	clientSecret string
	appClientID  string
	Connection   string
	InviteTTL    int

	database db.Database
}
type User struct {
	ID            uint      `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	FullName      string    `json:"full_name"`
	LastLogin     time.Time `json:"last_login"`
	Username      string    `json:"username"`
	Role          string    `json:"role"`
	IsActive      bool      `json:"is_active"`

	ExternalId string `json:"external_id"`
}

func DbUserToApi(u *db.User) (*User, error) {
	if u == nil {
		return nil, nil
	}

	return &User{
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		FullName:      u.FullName,
		LastLogin:     u.LastLogin,
		Username:      string(u.Username),
		Role:          string(u.Role),
		ExternalId:    u.ExternalId,
		ID:            u.ID,
		IsActive:      u.IsActive,
	}, nil
}

func GetOrCreateUser(userID string, email string, database db.Database) (*User, error) {

	if userID == "" {
		return nil, errors.New("GetOrCreateUser: empty user id")
	}

	user, err := database.GetUserByExternalID(userID)
	if err != nil {
		return nil, err
	}

	if user == nil {
		user = &db.User{
			Email:      email,
			Username:   email,
			FullName:   email,
			ExternalId: userID,
			Role:       api.ViewerRole,
		}
		err = database.CreateUser(user)
		if err != nil {
			return nil, err
		}
	}

	if !user.IsActive {
		return nil, errors.New("user disabled")
	}

	resp, err := DbUserToApi(user)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
func New(domain, appClientID, clientID, clientSecret, connection string, inviteTTL int, database db.Database) *Service {
	return &Service{
		domain:       domain,
		appClientID:  appClientID,
		clientID:     clientID,
		clientSecret: clientSecret,
		Connection:   connection,
		InviteTTL:    inviteTTL,
		database:     database,
	}
}

func GetUser(id string, database db.Database) (*User, error) {
	user, err := database.GetUserByExternalID(id)
	if err != nil {
		return nil, err
	}

	resp, err := DbUserToApi(user)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func UpdateUserLastLogin(userId string, lastLogin *time.Time, database db.Database) error {

	err := database.UpdateUserLastLoginWithExternalID(userId, lastLogin)

	if err != nil {
		return err
	}

	return nil
}
