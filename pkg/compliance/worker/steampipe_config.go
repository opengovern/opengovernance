package worker

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"gitlab.com/keibiengine/keibi-engine/pkg/config"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/vault"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

func (j *Job) PopulateSteampipeConfig(vault vault.SourceConfig, elasticSearchConfig config.ElasticSearch) error {
	cfg, err := vault.Read(j.ConfigReg)
	if err != nil {
		return err
	}

	var accountID string
	switch j.Connector {
	case source.CloudAWS:
		creds, err := AWSAccountConfigFromMap(cfg)
		if err != nil {
			return err
		}
		accountID = creds.AccountID

		err = BuildSpecFile("aws", elasticSearchConfig, accountID)
		if err != nil {
			return err
		}
	case source.CloudAzure:
		creds, err := AzureSubscriptionConfigFromMap(cfg)
		if err != nil {
			return err
		}
		accountID = creds.SubscriptionID

		err = BuildSpecFile("azure", elasticSearchConfig, accountID)
		if err != nil {
			return err
		}

		err = BuildSpecFile("azuread", elasticSearchConfig, accountID)
		if err != nil {
			return err
		}
	default:
		return errors.New("error: invalid source type")
	}
	return nil
}

func BuildSpecFile(plugin string, config config.ElasticSearch, accountID string) error {
	content := `
connection "` + plugin + `" {
  plugin = "` + plugin + `"
  addresses = ["` + config.Address + `"]
  username = "` + config.Username + `"
  password = "` + config.Password + `"
  accountID = "` + accountID + `"
}
`
	dirname, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	filePath := dirname + "/.steampipe/config/" + plugin + ".spc"
	os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	return ioutil.WriteFile(filePath, []byte(content), os.ModePerm)
}
