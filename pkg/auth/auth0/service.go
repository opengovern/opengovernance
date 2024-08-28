package auth0

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgtype"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/db"
	"github.com/kaytu-io/kaytu-util/pkg/api"
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

func (a *Service) GetOrCreateUser(userID, email string) (*User, error) {
	if userID == "" {
		return nil, errors.New("GetOrCreateUser: empty user id")
	}

	user, err := a.database.GetUser(userID)
	if err != nil {
		return nil, err
	}

	if user == nil || user.UserId == "" {
		var appMetadata Metadata
		appMetadata.WorkspaceAccess = map[string]api.Role{
			"main": api.AdminRole,
		}
		appMetadataJson, err := json.Marshal(appMetadata)
		if err != nil {
			return nil, err
		}

		appMetadataJsonb := pgtype.JSONB{}
		err = appMetadataJsonb.Set(appMetadataJson)
		if err != nil {
			return nil, err
		}

		userMetadataJsonb := pgtype.JSONB{}
		err = userMetadataJsonb.Set([]byte(""))
		if err != nil {
			return nil, err
		}

		user = &db.User{
			Email:        email,
			UserId:       userID,
			AppMetadata:  appMetadataJsonb,
			UserMetadata: userMetadataJsonb,
		}
		err = a.database.CreateUser(user)
		if err != nil {
			return nil, err
		}
	}

	resp, err := DbUserToApi(user)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *Service) GetUser(userID string) (*User, error) {
	user, err := a.database.GetUser(userID)
	if err != nil {
		return nil, err
	}

	resp, err := DbUserToApi(user)
	if err != nil {
		return nil, err
	}

	if resp.AppMetadata.WorkspaceAccess == nil {
		resp.AppMetadata.WorkspaceAccess = map[string]api.Role{}
	}
	resp.AppMetadata.WorkspaceAccess["main"] = api.AdminRole

	return resp, nil
}

func (a *Service) SearchByEmail(email string) ([]User, error) {
	users, err := a.database.GetUsersByEmail(email)
	if err != nil {
		return nil, err
	}

	var resp []User
	for _, user := range users {
		u, err := DbUserToApi(&user)
		if err != nil {
			return nil, err
		}

		if u.AppMetadata.WorkspaceAccess == nil {
			u.AppMetadata.WorkspaceAccess = map[string]api.Role{}
		}
		u.AppMetadata.WorkspaceAccess["main"] = api.AdminRole

		resp = append(resp, *u)
	}

	return resp, nil
}

func (a *Service) AddUser(user *User) error {
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
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		UserId:        user.UserId,
		LastLogin:     user.LastLogin,
		Name:          user.Name,
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
		Username:      user.Username,
	})
	if err != nil {
		return err
	}

	return nil
}

func (a *Service) CreateUser(email, wsName string, role api.Role) (*User, error) {
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
	return usr, a.AddUser(usr)
}

func (a *Service) DeleteUser(userId string) error {
	err := a.DeleteUser(userId)
	if err != nil {
		return err
	}
	return nil
}

func (a *Service) PatchUserAppMetadata(userId string, appMetadata Metadata) error {
	appMetadataJSON, err := json.Marshal(appMetadata)
	if err != nil {
		return err
	}

	jp := pgtype.JSONB{}
	err = jp.Set(appMetadataJSON)
	if err != nil {
		return err
	}

	err = a.database.UpdateUserAppMetadata(userId, jp)

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
