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

type SubscriptionConfig struct {
	SubscriptionID  string `json:"subscriptionId"`
	TenantID        string `json:"tenantId"`
	ObjectID        string `json:"objectId"`
	SecretID        string `json:"secretId"`
	ClientID        string `json:"clientId"`
	ClientSecret    string `json:"clientSecret"`
	CertificatePath string `json:"certificatePath"`
	CertificatePass string `json:"certificatePass"`
	Username        string `json:"username"`
	Password        string `json:"password"`
}

func AWSAccountConfigFromMap(config string) (AccountConfig, error) {
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

func AzureAccountConfigFromMap(config string) (SubscriptionConfig, error) {
	conf := make(map[string]any)
	err := json.Unmarshal([]byte(config), &conf)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
	}

	mj, err := json.Marshal(conf)
	if err != nil {
		return SubscriptionConfig{}, err
	}

	var c SubscriptionConfig
	err = json.Unmarshal(mj, &c)
	if err != nil {
		return SubscriptionConfig{}, err
	}

	return c, nil
}

func ConfigureAWSAccount(config string) error {
	c, err := AWSAccountConfigFromMap(config)
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

func ConfigureAzureAccount(config string) error {
	c, err := AzureAccountConfigFromMap(config)
	if err != nil {
		return err
	}

	err = os.Setenv("AZURE_SUBSCRIPTION_ID", c.SubscriptionID)
	if err != nil {
		return err
	}
	err = os.Setenv("AZURE_TENANT_ID", c.TenantID)
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
	err = os.Setenv("AZURE_SUBSCRIPTION_ID", "")
	if err != nil {
		return err
	}
	err = os.Setenv("AZURE_TENANT_ID", "")
	if err != nil {
		return err
	}
	return nil
}
