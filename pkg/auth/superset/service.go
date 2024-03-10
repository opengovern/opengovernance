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

type ListDashboardsItem struct {
	CertificationDetails interface{} `json:"certification_details"`
	CertifiedBy          interface{} `json:"certified_by"`
	ChangedBy            struct {
		FirstName string `json:"first_name"`
		Id        int    `json:"id"`
		LastName  string `json:"last_name"`
	} `json:"changed_by"`
	ChangedByName           string `json:"changed_by_name"`
	ChangedOnDeltaHumanized string `json:"changed_on_delta_humanized"`
	ChangedOnUtc            string `json:"changed_on_utc"`
	CreatedBy               struct {
		FirstName string `json:"first_name"`
		Id        int    `json:"id"`
		LastName  string `json:"last_name"`
	} `json:"created_by"`
	CreatedOnDeltaHumanized string      `json:"created_on_delta_humanized"`
	Css                     interface{} `json:"css"`
	DashboardTitle          string      `json:"dashboard_title"`
	Id                      int         `json:"id"`
	IsManagedExternally     bool        `json:"is_managed_externally"`
	JsonMetadata            interface{} `json:"json_metadata"`
	Owners                  []struct {
		FirstName string `json:"first_name"`
		Id        int    `json:"id"`
		LastName  string `json:"last_name"`
	} `json:"owners"`
	PositionJson interface{}   `json:"position_json"`
	Published    bool          `json:"published"`
	Roles        []interface{} `json:"roles"`
	Slug         interface{}   `json:"slug"`
	Status       string        `json:"status"`
	Tags         []interface{} `json:"tags"`
	ThumbnailUrl string        `json:"thumbnail_url"`
	Url          string        `json:"url"`
}

type ListDashboardsResponse struct {
	Count        int                  `json:"count"`
	Ids          []int                `json:"ids"`
	ListColumns  []string             `json:"list_columns"`
	ListTitle    string               `json:"list_title"`
	OrderColumns []string             `json:"order_columns"`
	Result       []ListDashboardsItem `json:"result"`
}

type GetEmbeddedDashboardResponse struct {
	Result struct {
		AllowedDomains []interface{} `json:"allowed_domains"`
		ChangedBy      struct {
			FirstName string `json:"first_name"`
			Id        int    `json:"id"`
			LastName  string `json:"last_name"`
			Username  string `json:"username"`
		} `json:"changed_by"`
		ChangedOn   string `json:"changed_on"`
		DashboardId string `json:"dashboard_id"`
		Uuid        string `json:"uuid"`
	} `json:"result"`
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

func (s *SupersetService) ListDashboards(token string) ([]ListDashboardsItem, error) {
	url := fmt.Sprintf("%s/api/v1/dashboard/", s.BaseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if res.StatusCode != http.StatusOK {
		r, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("[Login] invalid status code: %d, body=%s", res.StatusCode, string(r))
	}

	r, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var resp ListDashboardsResponse
	err = json.Unmarshal(r, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Result, nil
}

func (s *SupersetService) GetEmbeddedUUID(token string, id int) (string, error) {
	url := fmt.Sprintf("%s/api/v1/dashboard/%d/embedded", s.BaseURL, id)

	req, err := http.NewRequest("GET", url, nil)
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

	var resp GetEmbeddedDashboardResponse
	err = json.Unmarshal(r, &resp)
	if err != nil {
		return "", err
	}

	return resp.Result.Uuid, nil
}
