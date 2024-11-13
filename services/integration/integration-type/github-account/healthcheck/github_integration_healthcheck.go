package healthcheck

import (
	"context"
	"fmt"
	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v55/github"
	"golang.org/x/oauth2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type ClientCredential interface {
	GetClient(ctx context.Context) (*github.Client, error)
}

type ClientAccessTokenCredential struct {
	Token   string `json:"token"`
	BaseURL string `json:"base_url"`
}

func (c *ClientAccessTokenCredential) GetClient(ctx context.Context) (*github.Client, error) {
	var client *github.Client
	// Authentication with Github access token
	if strings.HasPrefix(c.Token, "ghp_") {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: c.Token},
		)
		tc := oauth2.NewClient(ctx, ts)
		client = github.NewClient(tc)
	}
	// Authentication Using App Installation Access Token or OAuth Access token
	if strings.HasPrefix(c.Token, "ghs_") || strings.HasPrefix(c.Token, "gho_") {
		client = github.NewClient(&http.Client{Transport: &oauth2Transport{
			Token: c.Token,
		}})
	}
	// If the base URL was provided then set it on the client. Used for enterprise installs.
	if c.BaseURL != "" {
		uv4, err := url.Parse(c.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("github.base_url is invalid: %s", c.BaseURL)
		}
		if uv4.String() != "https://api.github.com/" {
			uv4.Path = uv4.Path + "api/v3/"
		}
		// The upload URL is not set as it's not currently required
		conn, err := github.NewClient(client.Client()).WithEnterpriseURLs(uv4.String(), "")
		if err != nil {
			return nil, fmt.Errorf("error creating GitHub client: %v", err)
		}
		conn.BaseURL = uv4
		client = conn
	}
	return client, nil
}

type ClientAppInstallationCredential struct {
	AppId          string `json:"app_id"`
	InstallationId string `json:"installation_id"`
	PrivateKeyPath string `json:"private_key_path"`
	BaseURL        string `json:"base_url"`
}

func (c *ClientAppInstallationCredential) GetClient(ctx context.Context) (*github.Client, error) {
	var client *github.Client
	// Authentication as Github APP Installation authentication
	ghAppId, err := strconv.ParseInt(c.AppId, 10, 64)
	if err != nil {
		return nil, err
	}
	ghInstallationId, err := strconv.ParseInt(c.InstallationId, 10, 64)
	if err != nil {
		return nil, err
	}
	itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, ghAppId, ghInstallationId, c.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("Error occurred in 'connect()' during GitHub App Installation client creation: " + err.Error())
	}
	client = github.NewClient(&http.Client{Transport: itr})
	// If the base URL was provided then set it on the client. Used for enterprise installs.
	if c.BaseURL != "" {
		uv4, err := url.Parse(c.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("github.base_url is invalid: %s", c.BaseURL)
		}
		if uv4.String() != "https://api.github.com/" {
			uv4.Path = uv4.Path + "api/v3/"
		}
		// The upload URL is not set as it's not currently required
		conn, err := github.NewClient(client.Client()).WithEnterpriseURLs(uv4.String(), "")
		if err != nil {
			return nil, fmt.Errorf("error creating GitHub client: %v", err)
		}
		conn.BaseURL = uv4
		client = conn
	}
	return client, nil
}

func createClientDetail(ctx context.Context, config Config) (*github.Client, error) {
	var credential ClientCredential
	if config.Token != "" {
		credential = &ClientAccessTokenCredential{
			Token:   config.Token,
			BaseURL: config.BaseURL,
		}
		return credential.GetClient(ctx)
	} else if config.AppId != "" && config.InstallationId != "" && config.PrivateKeyPath != "" {
		credential = &ClientAppInstallationCredential{
			AppId:          config.AppId,
			InstallationId: config.InstallationId,
			PrivateKeyPath: config.PrivateKeyPath,
			BaseURL:        config.BaseURL,
		}
		return credential.GetClient(ctx)
	}
	return nil, nil
}

type GithubClient struct {
	ID           int64
	Name         string
	Type         string
	ClientDetail *github.Client
}

func newGithubClient(clientDetail *github.Client) *GithubClient {
	return &GithubClient{
		ClientDetail: clientDetail,
	}
}

func (client *GithubClient) IsHealthy() (bool, error) {
	user, _, err := client.ClientDetail.Users.Get(context.Background(), "")
	if err != nil {
		return false, err
	}
	client.ID = *user.ID
	client.Name = *user.Login
	client.Type = *user.Type
	return true, nil
}

type Config struct {
	Token          string `json:"token"`
	BaseURL        string `json:"base_url"`
	AppId          string `json:"app_id"`
	InstallationId string `json:"installation_id"`
	PrivateKeyPath string `json:"private_key_path"`
}

func checkCredentials(config Config) error {
	if config.Token == "" && (config.AppId == "" || config.InstallationId == "" || config.PrivateKeyPath == "") {
		return fmt.Errorf("'token' or 'app_id', 'installation_id' and 'private_key' must be set in the connection configuration")
	}
	// Return error for unsupported token by prefix
	if config.Token != "" && !strings.HasPrefix(config.Token, "ghs_") && !strings.HasPrefix(config.Token, "ghp_") && !strings.HasPrefix(config.Token, "gho_") {
		return fmt.Errorf("wrong token format. tokens should start with ghs_ or ghp_ or gho_")
	}
	return nil
}

func GithubIntegrationHealthcheck(config Config) (bool, error) {
	ctx := context.Background()
	err := checkCredentials(config)
	if err != nil {
		return false, err
	}
	clientDetail, err := createClientDetail(ctx, config)
	if err != nil {
		return false, err
	}
	githubClient := newGithubClient(clientDetail)
	return githubClient.IsHealthy()
}

// oauth2Transport is a http.RoundTripper that authenticates all requests
type oauth2Transport struct {
	Token string
}

func (t *oauth2Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "Bearer "+t.Token)
	return http.DefaultTransport.RoundTrip(clone)
}
