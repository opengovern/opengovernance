package worker

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gitlab.com/keibiengine/keibi-engine/pkg/config"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

func (j *Job) PopulateSteampipeConfig(elasticSearchConfig config.ElasticSearch, AccountID string) error {
	cfg := map[string]interface{}{}

	var accountID string
	switch j.Connector {
	case source.CloudAWS:
		creds, err := AWSAccountConfigFromMap(cfg)
		if err != nil {
			return err
		}
		fmt.Println(creds.AccountID)

		err = BuildSpecFile("aws", elasticSearchConfig, accountID)
		if err != nil {
			return err
		}

		err = PopulateEnv(elasticSearchConfig, accountID)
		if err != nil {
			return err
		}
	case source.CloudAzure:
		creds, err := AzureSubscriptionConfigFromMap(cfg)
		if err != nil {
			return err
		}

		fmt.Println(creds.SubscriptionID)

		err = BuildSpecFile("azure", elasticSearchConfig, accountID)
		if err != nil {
			return err
		}

		err = BuildSpecFile("azuread", elasticSearchConfig, accountID)
		if err != nil {
			return err
		}

		err = PopulateEnv(elasticSearchConfig, accountID)
		if err != nil {
			return err
		}
	default:
		return errors.New("error: invalid source type")
	}
	return nil
}

func PopulateEnv(config config.ElasticSearch, accountID string) error {
	err := os.Setenv("STEAMPIPE_ACCOUNT_ID", accountID)
	if err != nil {
		return err
	}
	err = os.Setenv("ES_ADDRESS", config.Address)
	if err != nil {
		return err
	}
	err = os.Setenv("ES_USERNAME", config.Username)
	if err != nil {
		return err
	}
	err = os.Setenv("ES_PASSWORD", config.Password)
	if err != nil {
		return err
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
