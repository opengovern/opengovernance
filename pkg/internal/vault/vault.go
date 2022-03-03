package vault

import (
	"context"
	"fmt"
	"log"

	vault "github.com/hashicorp/vault/api"
)

//go:generate mockery --name SourceConfig
type SourceConfig interface {
	Write(pathRef string, config map[string]interface{}) (err error)
	Read(pathRef string) (config map[string]interface{}, err error)
	Delete(pathRef string) error
}

type vaultSourceConfig struct {
	client *vault.Client
}

func NewSourceConfig(vaultAddress string, auth vault.AuthMethod) (SourceConfig, error) {
	conf := vault.DefaultConfig()
	conf.Address = vaultAddress

	c, err := vault.NewClient(conf)
	if err != nil {
		return nil, fmt.Errorf("new source config vault: %w", err)
	}

	secret, err := c.Auth().Login(context.TODO(), auth)
	if err != nil {
		return nil, fmt.Errorf("vault authenticate: %w", err)
	}

	vault := vaultSourceConfig{client: c}
	if err := vault.watchSecret(secret); err != nil {
		return nil, err
	}

	return vault, nil
}

func (v vaultSourceConfig) watchSecret(s *vault.Secret) error {
	if s.Renewable {
		return nil
	}

	watcher, err := v.client.NewLifetimeWatcher(&vault.LifetimeWatcherInput{
		Secret: s,
	})
	if err != nil {
		return err
	}
	go watcher.Start()

	go func() {
		for {
			select {
			case err := <-watcher.DoneCh():
				if err != nil {
					// TODO: Don't fail here. Instead return error to upstream to handle as needed
					log.Fatal(err)
				}

				// Renewal is now over
			case renewal := <-watcher.RenewCh():
				fmt.Printf("Successfully renewed secret %s at %s\n", renewal.Secret.RequestID, renewal.RenewedAt.String())
			}
		}
	}()

	return nil
}

func (v vaultSourceConfig) Write(pathRef string, config map[string]interface{}) error {
	_, err := v.client.Logical().Write(pathRef, config)
	if err != nil {
		return err
	}

	return nil
}

func (v vaultSourceConfig) Read(pathRef string) (map[string]interface{}, error) {
	config, err := v.client.Logical().Read(pathRef)
	if err != nil {
		return nil, err
	}

	if config == nil {
		return nil, fmt.Errorf("invalid pathRef: %s", pathRef)
	}

	return config.Data, nil
}

func (v vaultSourceConfig) Delete(pathRef string) error {
	_, err := v.client.Logical().Delete(pathRef)
	return err
}
