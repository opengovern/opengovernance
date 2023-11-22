package v2

import (
	"encoding/json"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type CreateCredentialRequest struct {
	Connector source.Type `json:"connector" example:"Azure"`
	Config    any         `json:"config"`
}

type CreateCredentialResponse struct {
	ID string `json:"id"`
}

func (req CreateCredentialRequest) GetAWSConfig() (*AWSCredentialConfig, error) {
	configStr, err := json.Marshal(req.Config)
	if err != nil {
		return nil, err
	}

	config := AWSCredentialConfig{}
	err = json.Unmarshal(configStr, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

type AWSCredentialConfig struct {
	AccountID           string   `json:"accountID"`
	AssumeRoleName      string   `json:"assumeRoleName"`
	HealthCheckPolicies []string `json:"healthCheckPolicies"`
	ExternalId          *string  `json:"externalId"`
}

func (s AWSCredentialConfig) AsMap() map[string]any {
	in, err := json.Marshal(s)
	if err != nil {
		panic(err) // Don't expect any error
	}

	var out map[string]any
	if err := json.Unmarshal(in, &out); err != nil {
		panic(err) // Don't expect any error
	}

	return out
}
