package auth0

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"io/ioutil"
	"math/rand"
	"net/http"
	url2 "net/url"
)

type Service struct {
	domain         string
	clientID       string
	clientSecret   string
	ConnectionID   string
	RedirectURL    string
	OrganizationID string
	InviteTTL      int

	token string
}

func New(domain, clientID, clientSecret, connectionID, orgID, redirectURL string, inviteTTL int) *Service {
	return &Service{
		domain:         domain,
		clientID:       clientID,
		clientSecret:   clientSecret,
		ConnectionID:   connectionID,
		RedirectURL:    redirectURL,
		OrganizationID: orgID,
		InviteTTL:      inviteTTL,
		token:          "",
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

func (a *Service) GetUser(userId string) (*User, error) {
	if err := a.fillToken(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v2/users/%s", a.domain, userId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+a.token)
	res, err := http.DefaultClient.Do(req)
	if res.StatusCode != http.StatusOK {
		r, _ := ioutil.ReadAll(res.Body)
		return nil, fmt.Errorf("[GetUser] invalid status code: %d, body=%s", res.StatusCode, string(r))
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

func (a *Service) SearchByEmail(email string) ([]User, error) {
	if err := a.fillToken(); err != nil {
		return nil, err
	}

	encoded := url2.Values{}
	encoded.Set("email", email)

	url := fmt.Sprintf("%s/api/v2/users-by-email?%s", a.domain, encoded.Encode())
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+a.token)
	res, err := http.DefaultClient.Do(req)
	if res.StatusCode != http.StatusOK {
		r, _ := ioutil.ReadAll(res.Body)
		return nil, fmt.Errorf("[SearchByEmail] invalid status code: %d, body=%s", res.StatusCode, string(r))
	}

	r, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var resp []User
	err = json.Unmarshal(r, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *Service) CreateUser(email, wsName string, role api.Role) error {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")
	password := make([]rune, 32)
	i := 0
	for ; i < 10; i++ {
		password[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	letterRunes = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	for ; i < 10; i++ {
		password[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	letterRunes = []rune("0123456789")
	for ; i < 5; i++ {
		password[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	letterRunes = []rune("~!|")
	for ; i < 2; i++ {
		password[i] = letterRunes[rand.Intn(len(letterRunes))]
	}

	usr := CreateUserRequest{
		Email:         email,
		EmailVerified: false,
		AppMetadata: Metadata{
			WorkspaceAccess: map[string]api.Role{
				wsName: role,
			},
			GlobalAccess: nil,
		},
		Password:   string(password),
		Connection: a.ConnectionID,
	}

	if err := a.fillToken(); err != nil {
		return err
	}

	body, err := json.Marshal(usr)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v2/users", a.domain)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	fmt.Println("POST", url)
	fmt.Println(body)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+a.token)
	res, err := http.DefaultClient.Do(req)
	if res.StatusCode != http.StatusCreated {
		r, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("[CreateUser] invalid status code: %d, body=%s", res.StatusCode, string(r))
	}
	return nil
}

func (a *Service) CreatePasswordChangeTicket(email string) (*CreatePasswordChangeTicketResponse, error) {
	request := CreatePasswordChangeTicketRequest{
		ResultUrl:              a.RedirectURL,
		UserId:                 "",
		ClientId:               a.clientID,
		OrganizationId:         a.OrganizationID,
		ConnectionId:           a.ConnectionID,
		Email:                  email,
		TtlSec:                 a.InviteTTL,
		MarkEmailAsVerified:    false,
		IncludeEmailInRedirect: false,
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
	res, err := http.DefaultClient.Do(req)
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
	if err := a.fillToken(); err != nil {
		return err
	}

	js, err := json.Marshal(appMetadata)
	if err != nil {
		return err
	}

	js = []byte(fmt.Sprintf(`{"app_metadata": %s}`, string(js)))

	url := fmt.Sprintf("%s/api/v2/users/%s", a.domain, userId)
	req, err := http.NewRequest("PATCH", url, bytes.NewReader(js))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+a.token)
	req.Header.Add("Content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if res.StatusCode != http.StatusOK {
		r, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("[PatchUserAppMetadata] invalid status code: %d, body=%s", res.StatusCode, string(r))
	}
	return nil
}

func (a *Service) SearchUsersByWorkspace(wsName string) ([]User, error) {
	if err := a.fillToken(); err != nil {
		return nil, err
	}
	url, err := url2.Parse(fmt.Sprintf("%s/api/v2/users", a.domain))
	if err != nil {
		return nil, err
	}

	url.Query().Add("search_engine", "v3")
	url.Query().Add("q", "_exists_:app_metadata.access."+wsName)
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+a.token)
	res, err := http.DefaultClient.Do(req)
	if res.StatusCode != http.StatusOK {
		r, _ := ioutil.ReadAll(res.Body)
		return nil, fmt.Errorf("[SearchUsersByWorkspace] invalid status code: %d, body=%s", res.StatusCode, string(r))
	}

	r, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var resp []User
	err = json.Unmarshal(r, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
