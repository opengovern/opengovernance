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

func ConfigureAWSAccount(config string) (error, string, string, string) {
	c, err := AccountConfigFromMap(config)
	if err != nil {
		return err, "", "", ""
	}

	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	sessionToken := os.Getenv("AWS_SESSION_TOKEN")

	err = os.Setenv("AWS_ACCESS_KEY_ID", c.AccessKey)
	if err != nil {
		return err, "", "", ""
	}
	err = os.Setenv("AWS_SECRET_ACCESS_KEY", c.SecretKey)
	if err != nil {
		return err, "", "", ""
	}
	err = os.Setenv("AWS_SESSION_TOKEN", c.SessionToken)
	if err != nil {
		return err, "", "", ""
	}
	return nil, accessKey, secretKey, sessionToken
}

func ConfigureAWSManual(accessKey string, secretKey string, sessionToken string) error {
	err := os.Setenv("AWS_ACCESS_KEY_ID", accessKey)
	if err != nil {
		return err
	}
	err = os.Setenv("AWS_SECRET_ACCESS_KEY", secretKey)
	if err != nil {
		return err
	}
	err = os.Setenv("AWS_SESSION_TOKEN", sessionToken)
	if err != nil {
		return err
	}
	return nil
}
