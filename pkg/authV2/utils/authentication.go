package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgtype"
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
	FullName    string    `json:"full_name"`
	LastLogin     time.Time `json:"last_login"`
	Username      string    `json:"username"`
	Role			string `json:"role"`
	IsActive        bool	`json:"is_active"`
	IsDeleted        bool	`json:"is_deleted"`

	
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
		FullName:    u.FullName,
		LastLogin:     u.LastLogin,
		Username:      string(u.Role),
		IsActive: u.IsActive,
		IsDeleted: u.IsDeleted,


		
	}, nil
}

func  GetOrCreateUser(userID string, email string, database db.Database ) (*User, error) {
	
	if userID == "" {
		return nil, errors.New("GetOrCreateUser: empty user id")
	}

	user, err := database.GetUserByExternalID(userID)
	if err != nil {
		return nil, err
	}

	if user == nil  {
		user = &db.User{
			Email:        email,
			Username:     email,
			FullName:         email,
			ExternalId: userID,
			Role:         api.ViewerRole,
			
		}
		err = database.CreateUser(user)
		if err != nil {
			return nil, err
		}
	}

	if user.IsActive {
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



func (a *Service) GetUser(id uuid.UUID) (*db.User, error) {
	user, err := a.database.GetUser(id)
	if err != nil {
		return nil, err
	}

	resp, err := DbUserToApi(user)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *Service) SearchByEmail(email string) ([]db.User, error) {
	users, err := a.database.GetUsersByEmail(email)
	if err != nil {
		return nil, err
	}

	var resp []db.User
	for _, user := range users {
		u, err := DbUserToApi(&user)
		if err != nil {
			return nil, err
		}

		resp = append(resp, *u)
	}

	return resp, nil
}

func (a *Service) AddUser(user *db.User, role api.Role) error {
	appMetadataJSON, err := json.Marshal(user.AppMetadata)
	if err != nil {
		return err
	}

	appMetadataJsonb := pgtype.JSONB{}
	err = appMetadataJsonb.Set(appMetadataJSON)
	if err != nil {
		return err
	}

	userMetadataJSON, err := json.Marshal(user.UserMetadata)
	if err != nil {
		return err
	}

	userMetadataJsonb := pgtype.JSONB{}
	err = userMetadataJsonb.Set(userMetadataJSON)
	if err != nil {
		return err
	}

	err = a.database.CreateUser(&db.User{
		UserUuid:      uuid.New(),
		Username:      user.Email,
		Name:          user.Email,
		IdLifecycle:   db.UserLifecycleActive,
		Role:          role,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		UserId:        user.UserId,
		LastLogin:     user.LastLogin,
		AppMetadata:   appMetadataJsonb,
		Blocked:       user.Blocked,
		FamilyName:    user.FamilyName,
		GivenName:     user.GivenName,
		LastIp:        user.LastIp,
		Locale:        user.Locale,
		LoginsCount:   user.LoginsCount,
		Multifactor:   user.Multifactor,
		Nickname:      user.Nickname,
		PhoneNumber:   user.PhoneNumber,
		PhoneVerified: user.PhoneVerified,
		UserMetadata:  userMetadataJsonb,
		Picture:       user.Picture,
	})
	if err != nil {
		return err
	}

	return nil
}

func (a *Service) CreateUser(email, wsName string, role api.Role) (*db.User, error) {
	usr := &User{
		Email:         email,
		EmailVerified: false,
		UserId:        fmt.Sprintf("dex|%s", email),
		AppMetadata: Metadata{
			WorkspaceAccess: map[string]api.Role{
				wsName: role,
			},
			GlobalAccess: nil,
		},
	}
	return usr, a.AddUser(usr, role)
}

func (a *Service) DeleteUser(userId string) error {
	err := a.DeleteUser(userId)
	if err != nil {
		return err
	}
	return nil
}

func  UpdateUserLastLogin(userId string, lastLogin *time.Time,database db.Database) error {
	

	err := database.UpdateUserLastLoginWithExternalID(userId,  lastLogin)

	if err != nil {
		return err
	}

	return nil
}

func (a *Service) SearchUsersByWorkspace(wsID string) ([]User, error) {
	users, err := a.database.GetUsersByWorkspace(wsID)
	if err != nil {
		return nil, err
	}

	var resp []User
	for _, user := range users {
		u, err := DbUserToApi(&user)
		if err != nil {
			return nil, err
		}
		resp = append(resp, *u)
	}
	return resp, nil
}

func (a *Service) SearchUsers(wsID string, email *string, emailVerified *bool, role *api.Role) ([]User, error) {
	users, err := a.database.SearchUsers(wsID, email, emailVerified)
	if err != nil {
		return nil, err
	}

	var apiUsers []User
	for _, user := range users {
		u, err := DbUserToApi(&user)
		if err != nil {
			return nil, err
		}
		apiUsers = append(apiUsers, *u)
	}
	var resp []User
	if role != nil {
		for _, user := range apiUsers {
			if func() bool {
				for _, r := range user.AppMetadata.WorkspaceAccess {
					if r == *role {
						return true
					}
				}
				return false
			}() {
				resp = append(resp, user)
			}
		}
	} else {
		resp = apiUsers
	}
	return resp, nil
}
