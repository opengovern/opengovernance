package auth0

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	url2 "net/url"
)

type Service struct {
	domain       string
	clientID     string
	clientSecret string

	token string
}

func New(domain, clientID, clientSecret string) *Service {
	return &Service{
		domain:       domain,
		clientID:     clientID,
		clientSecret: clientSecret,
		token:        "",
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
		return nil, fmt.Errorf("[GetUser] invalid status code: %d", res.StatusCode)
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

func (a *Service) PatchUserAppMetadata(userId string, appMetadata Metadata) error {
	if err := a.fillToken(); err != nil {
		return err
	}

	js, err := json.Marshal(appMetadata)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v2/users/%s", a.domain, userId)
	req, err := http.NewRequest("PATCH", url, bytes.NewReader(js))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+a.token)
	req.Header.Add("Content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("[GetUser] invalid status code: %d", res.StatusCode)
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
		return nil, fmt.Errorf("[SearchUsersByWorkspace] invalid status code: %d", res.StatusCode)
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
