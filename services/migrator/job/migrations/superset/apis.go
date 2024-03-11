package superset

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"
	"time"
)

type supersetWrapper struct {
	logger *zap.Logger

	httpClient http.Client

	BaseURL       string
	AdminPassword string

	AccessToken  string
	RefreshToken string
}

func newSupersetWrapper(logger *zap.Logger, baseURL string, adminPassword string) (*supersetWrapper, error) {
	httpClient := http.Client{
		Timeout: 30 * time.Second,
	}
	sw := &supersetWrapper{
		logger:        logger,
		httpClient:    httpClient,
		BaseURL:       baseURL,
		AdminPassword: adminPassword,
	}

	err := sw.doAuth()
	if err != nil {
		logger.Error("failed to authenticate", zap.Error(err))
		return nil, err
	}

	return sw, nil
}

func (w *supersetWrapper) doAuth() error {
	if w.AccessToken == "" && w.RefreshToken == "" {
		response, err := w.securityLoginV1()
		if err != nil {
			return err
		}
		w.AccessToken = response.AccessToken
		w.RefreshToken = response.RefreshToken
	} else {
		response, err := w.securityRefreshV1()
		if err != nil {
			return err
		}
		w.AccessToken = response.AccessToken
	}
	return nil
}

func (w *supersetWrapper) doRequest(method, path string, auth bool, body any, response any) error {
	var jsonBody []byte
	var err error
	if body != nil {
		jsonBody, err = json.Marshal(body)
		if err != nil {
			w.logger.Error("failed to marshal request body", zap.Error(err), zap.String("path", path), zap.String("method", method), zap.Any("body", body))
			return err
		}
	}

	httpRequest, err := http.NewRequest(method, w.BaseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		w.logger.Error("failed to create http request", zap.Error(err), zap.String("path", path), zap.String("method", method), zap.Any("body", body))
		return err
	}

	httpRequest.Header.Set("Content-Type", "application/json")
	if auth {
		httpRequest.Header.Set("Authorization", "Bearer "+w.AccessToken)
	}
	httpRequest.Header.Set("accept", "application/json")

	httpResponse, err := w.httpClient.Do(httpRequest)
	if err != nil {
		w.logger.Error("failed to do http request", zap.Error(err), zap.String("path", path), zap.String("method", method), zap.Any("body", body))
		return err
	}

	defer httpResponse.Body.Close()
	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		responseBody, err := io.ReadAll(httpResponse.Body)
		if err != nil {
			w.logger.Error("failed to read response body", zap.Error(err), zap.String("path", path), zap.String("method", method), zap.Any("body", body), zap.Any("response", httpResponse))
		}
		w.logger.Error("http request failed", zap.String("path", path), zap.String("method", method), zap.Any("body", body), zap.Any("response", httpResponse))
		return fmt.Errorf("http request failed with status code %d - %s", httpResponse.StatusCode, string(responseBody))
	}

	err = json.NewDecoder(httpResponse.Body).Decode(response)
	if err != nil {
		w.logger.Error("failed to decode response body", zap.Error(err), zap.String("path", path), zap.String("method", method), zap.Any("body", body), zap.Any("response", httpResponse))
		return err
	}

	return nil
}

type securityLoginV1Request struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Provider string `json:"provider"`
	Refresh  bool   `json:"refresh"`
}

type securityLoginV1Response struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (w *supersetWrapper) securityLoginV1() (*securityLoginV1Response, error) {
	body := securityLoginV1Request{
		Username: "admin",
		Password: w.AdminPassword,
		Provider: "db",
		Refresh:  true,
	}
	var response securityLoginV1Response
	err := w.doRequest(http.MethodPost, "/api/v1/security/login", false, body, &response)
	if err != nil {
		w.logger.Error("failed to login", zap.Error(err))
		return nil, err
	}
	return &response, nil
}

type securityRefreshV1Response struct {
	AccessToken string `json:"access_token"`
}

func (w *supersetWrapper) securityRefreshV1() (*securityRefreshV1Response, error) {
	var response securityRefreshV1Response

	at := w.AccessToken
	w.AccessToken = w.RefreshToken
	defer func() {
		w.AccessToken = at
	}()
	err := w.doRequest(http.MethodPost, "/api/v1/security/refresh", true, make(map[string]any), &response)
	if err != nil {
		w.logger.Error("failed to refresh", zap.Error(err))
		return nil, err
	}
	return &response, nil
}

type createDatabaseV1Request struct {
	DatabaseName        string `json:"database_name"`
	Engine              string `json:"engine"`
	ConfigurationMethod string `json:"configuration_method"`
	EngineInformation   struct {
		DisableSSHTunneling bool `json:"disable_ssh_tunneling"`
		SupportsFileUpload  bool `json:"supports_file_upload"`
	}
	Driver                   string `json:"driver"`
	SqlAlchemyUriPlaceholder string `json:"sqlalchemy_uri_placeholder"`
	Extra                    string `json:"extra"`
	ExposeInSqllab           bool   `json:"expose_in_sqllab"`
	Parameters               struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		Database string `json:"database"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	MaskedEncryptedExtra string `json:"masked_encrypted_extra"`
}

func (w *supersetWrapper) createDatabaseV1(request createDatabaseV1Request) error {
	response := make(map[string]any)
	return w.doRequest(http.MethodPost, "/api/v1/database/", true, request, &response)
}

type database struct {
	AllowCtas                 bool   `json:"allow_ctas"`
	AllowCvas                 bool   `json:"allow_cvas"`
	AllowDml                  bool   `json:"allow_dml"`
	AllowFileUpload           bool   `json:"allow_file_upload"`
	AllowRunAsync             bool   `json:"allow_run_async"`
	AllowsCostEstimate        string `json:"allows_cost_estimate"`
	AllowsSubquery            bool   `json:"allows_subquery"`
	AllowsVirtualTableExplore bool   `json:"allows_virtual_table_explore"`
	Backend                   string `json:"backend"`
	ChangedBy                 struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	} `json:"changed_by"`
	ChangedOn               string `json:"changed_on"`
	ChangedOnDeltaHumanized string `json:"changed_on_delta_humanized"`
	CreatedBy               struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	} `json:"created_by"`
	DatabaseName       string `json:"database_name"`
	DisableDataPreview bool   `json:"disable_data_preview"`
	EngineInformation  struct {
		DisableSSHTunneling bool `json:"disable_ssh_tunneling"`
		SupportsFileUpload  bool `json:"supports_file_upload"`
	} `json:"engine_information"`
	ExploreDatabaseID int    `json:"explore_database_id"`
	ExposeInSqllab    bool   `json:"expose_in_sqllab"`
	Extra             string `json:"extra"`
	ForceCtasSchema   string `json:"force_ctas_schema"`
	ID                int    `json:"id"`
	UUID              string `json:"uuid"`
}

type listDatabaseV1Response struct {
	Count  int        `json:"count"`
	Result []database `json:"result"`
}

func (w *supersetWrapper) listDatabaseV1() (*listDatabaseV1Response, error) {
	var response listDatabaseV1Response
	err := w.doRequest(http.MethodGet, "/api/v1/database/", true, nil, &response)
	if err != nil {
		w.logger.Error("failed to list databases", zap.Error(err))
		return nil, err
	}
	return &response, nil
}
