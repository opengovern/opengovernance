package auth0

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgtype"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/db"
	"github.com/kaytu-io/kaytu-util/pkg/api"
	"io/ioutil"
	"math/rand"
	"net/http"
)

type Service struct {
	domain       string
	clientID     string
	clientSecret string
	appClientID  string
	Connection   string
	InviteTTL    int

	token string

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
		token:        "",
		database:     database,
	}
}

func (a *Service) fillToken() error {
	url := fmt.Sprintf("%s/oauth/token", a.domain)
	req := TokenRequest{
		ClientId:     a.clientID,
		ClientSecret: a.clientSecret,
		Audience:     fmt.Sprintf("%s/api/v2/", a.domain),
		GrantType:    "client_credentials",
	}
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	res, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		r, _ := ioutil.ReadAll(res.Body)
		str := ""
		if r != nil {
			str = string(r)
		}
		return fmt.Errorf("[fillToken] invalid status code: %d. res: %s", res.StatusCode, str)
	}

	r, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var resp TokenResponse
	err = json.Unmarshal(r, &resp)
	if err != nil {
		return err
	}

	a.token = resp.AccessToken
	return nil
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
		user = &db.User{
			Email:  email,
			UserId: userID,
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
	resp.AppMetadata.WorkspaceAccess["main"] = api.AdminRole

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

func (a *Service) CreateUser(email, wsName string, role api.Role) (*User, error) { // This should be deprecated
	var defaultPass = "kaytu23@"
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	randPass := make([]rune, 10)
	for i := 0; i < 10; i++ {
		randPass[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	password := fmt.Sprintf("%s%s", defaultPass, string(randPass))

	usr := CreateUserRequest{
		Email:         email,
		EmailVerified: false,
		AppMetadata: Metadata{
			WorkspaceAccess: map[string]api.Role{
				wsName: role,
			},
			GlobalAccess: nil,
		},
		Password:   password,
		Connection: a.Connection,
	}

	if err := a.fillToken(); err != nil {
		return nil, err
	}

	body, err := json.Marshal(usr)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v2/users", a.domain)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	fmt.Println("POST", url)
	fmt.Println(string(body))
	fmt.Println(body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+a.token)
	req.Header.Add("Content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		r, _ := ioutil.ReadAll(res.Body)
		return nil, fmt.Errorf("[CreateUser] invalid status code: %d, body=%s", res.StatusCode, string(r))
	}

	r, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var resp User
	err = json.Unmarshal(r, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (a *Service) DeleteUser(userId string) error {
	err := a.DeleteUser(userId)
	if err != nil {
		return err
	}
	return nil
}

func (a *Service) CreatePasswordChangeTicket(userId string) (*CreatePasswordChangeTicketResponse, error) { // I think this should be deprecated
	request := CreatePasswordChangeTicketRequest{
		UserId:   userId,
		ClientId: a.appClientID,
		TTLSec:   a.InviteTTL,
	}

	if err := a.fillToken(); err != nil {
		return nil, err
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v2/tickets/password-change", a.domain)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+a.token)
	req.Header.Add("Content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		r, _ := ioutil.ReadAll(res.Body)
		return nil, fmt.Errorf("[CreatePasswordChangeTicket] invalid status code: %d, body=%s", res.StatusCode, string(r))
	}

	r, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var resp CreatePasswordChangeTicketResponse
	err = json.Unmarshal(r, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
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
