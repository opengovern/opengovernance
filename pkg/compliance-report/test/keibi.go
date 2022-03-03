package test

import (
	"encoding/json"

	compliancereport "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report"
)

type SourceConfigMock struct {
}

// GetOrganizations() (pathRefs []string, err error)

func (s SourceConfigMock) Delete(pathRef string) error {
	return nil
}
func (s SourceConfigMock) Read(pathRef string) (config map[string]interface{}, err error) {
	switch pathRef {
	case "azure":
		cfg := compliancereport.AzureSubscriptionConfig{
			SubscriptionID: "a",
		}
		c, err := json.Marshal(cfg)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(c, &config)
		if err != nil {
			return nil, err
		}

		return config, nil
	}
	return nil, nil
}

func (s SourceConfigMock) Write(pathRef string, config map[string]interface{}) (err error) {
	return nil
}
