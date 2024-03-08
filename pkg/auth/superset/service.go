//go:generate oapi-codegen -package=examplepkg -generate=types,client,spec -o=examplepkg/example-client.go ./docs/_openapi.json

package superset

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type SupersetService struct {
	BaseURL            string
	username, password string
}

func New(baseURL, username, password string) *SupersetService {
	return &SupersetService{
		BaseURL:  baseURL,
		username: username,
		password: password,
	}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Provider string `json:"provider"`
	Refresh  bool   `json:"refresh"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type GuestUser struct {
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type Resource struct {
	Type string `json:"type"`
	Id   string `json:"id"`
}

type RLS struct {
	Clause string `json:"clause"`
}

type GuestTokenRequest struct {
	User      GuestUser  `json:"user"`
	Resources []Resource `json:"resources"`
	Rls       []RLS      `json:"rls"`
}

type GuestTokenResponse struct {
	Token string `json:"token"`
}

func (s *SupersetService) Login() (string, error) {
	url := fmt.Sprintf("%s/api/v1/security/login", s.BaseURL)

	request := LoginRequest{
		Username: s.username,
		Password: s.password,
		Provider: "db",
		Refresh:  false,
	}
	reqBody, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if res.StatusCode != http.StatusOK {
		r, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("[Login] invalid status code: %d, body=%s", res.StatusCode, string(r))
	}

	r, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var resp LoginResponse
	err = json.Unmarshal(r, &resp)
	if err != nil {
		return "", err
	}

	return resp.AccessToken, nil
}

func (s *SupersetService) GuestToken(token string, request GuestTokenRequest) (string, error) {
	url := fmt.Sprintf("%s/api/v1/security/guest_token/", s.BaseURL)
	reqBody, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if res.StatusCode != http.StatusOK {
		r, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("[Login] invalid status code: %d, body=%s", res.StatusCode, string(r))
	}

	r, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var resp GuestTokenResponse
	err = json.Unmarshal(r, &resp)
	if err != nil {
		return "", err
	}

	return resp.Token, nil
}
