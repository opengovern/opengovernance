package test

import (
	"encoding/json"
	"github.com/google/uuid"
	compliancereport "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report"
)

type KeibiMock struct {
}

// GetOrganizations() (pathRefs []string, err error)

func (s KeibiMock) DeleteOrganization(pathRef string) error {
	return nil
}

func (s KeibiMock) NewOrganization(orgId uuid.UUID) (pathRef string, err error) {
	return "", nil
}

func (s KeibiMock) DeleteSourceConfig(pathRef string) error {
	return nil
}
func (s KeibiMock) ReadSourceConfig(pathRef string) (config map[string]interface{}, err error) {
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
func (s KeibiMock) WriteSourceConfig(orgId uuid.UUID, sourceId uuid.UUID, sourceType string, config interface{}) (configRef string, err error) {
	return "", nil
}
