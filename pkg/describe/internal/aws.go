package internal

import (
	"encoding/json"
	"fmt"
	"os"
)

type AccountConfig struct {
	AccountID      string   `json:"accountId"`
	Regions        []string `json:"regions"`
	SecretKey      string   `json:"secretKey"`
	AccessKey      string   `json:"accessKey"`
	SessionToken   string   `json:"sessionToken"`
	AssumeRoleName string   `json:"assumeRoleName"`
	ExternalID     *string  `json:"externalID,omitempty"`
}

func AccountConfigFromMap(config string) (AccountConfig, error) {
	conf := make(map[string]any)
	err := json.Unmarshal([]byte(config), &conf)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
	}

	mj, err := json.Marshal(conf)
	if err != nil {
		return AccountConfig{}, err
	}

	var c AccountConfig
	err = json.Unmarshal(mj, &c)
	if err != nil {
		return AccountConfig{}, err
	}

	return c, nil
}

func ConfigureAWSAccount(config string) error {
	c, err := AccountConfigFromMap(config)
	if err != nil {
		return err
	}
	err = os.Setenv("AWS_ACCESS_KEY_ID", c.AccessKey)
	if err != nil {
		return err
	}
	err = os.Setenv("AWS_SECRET_ACCESS_KEY", c.SecretKey)
	if err != nil {
		return err
	}
	err = os.Setenv("AWS_SESSION_TOKEN", c.SessionToken)
	if err != nil {
		return err
	}
	return nil
}

func RestartCredentials() error {
	err := os.Setenv("AWS_ACCESS_KEY_ID", "")
	if err != nil {
		return err
	}
	err = os.Setenv("AWS_SECRET_ACCESS_KEY", "")
	if err != nil {
		return err
	}
	err = os.Setenv("AWS_SESSION_TOKEN", "")
	if err != nil {
		return err
	}
	return nil
}
