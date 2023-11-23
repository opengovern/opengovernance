package v2

import (
	"encoding/json"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type CreateCredentialV2Request struct {
	Connector source.Type            `json:"connector" example:"Azure"`
	AWSConfig *AWSCredentialV2Config `json:"awsConfig"`
}

type CreateCredentialV2Response struct {
	ID string `json:"id"`
}

func (req CreateCredentialV2Request) GetAWSConfig() (*AWSCredentialV2Config, error) {
	configStr, err := json.Marshal(req.AWSConfig)
	if err != nil {
		return nil, err
	}

	config := AWSCredentialV2Config{}
	err = json.Unmarshal(configStr, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

type AWSCredentialV2Config struct {
	AccountID           string   `json:"accountID"`
	AssumeRoleName      string   `json:"assumeRoleName"`
	HealthCheckPolicies []string `json:"healthCheckPolicies"`
	ExternalId          *string  `json:"externalId"`
}

func (s AWSCredentialV2Config) AsMap() map[string]any {
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

func AWSCredentialV2ConfigFromMap(cnf map[string]any) (*AWSCredentialV2Config, error) {
	in, err := json.Marshal(cnf)
	if err != nil {
		return nil, err
	}

	var out AWSCredentialV2Config
	if err := json.Unmarshal(in, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
