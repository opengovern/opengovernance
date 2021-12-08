package vault

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	vault "github.com/hashicorp/vault/api"
)

type Keibi interface {
	// GetOrganizations() (pathRefs []string, err error)

	DeleteOrganization(pathRef string) error
	NewOrganization(orgId uuid.UUID) (pathRef string, err error)

	DeleteSourceConfig(pathRef string) error
	ReadSourceConfig(pathRef string) (config map[string]interface{}, err error)
	WriteSourceConfig(orgId uuid.UUID, sourceId uuid.UUID, sourceType string, config interface{}) (configRef string, err error)
}

type Vault struct {
	client *vault.Client
}

func NewVault(vaultAddress string, auth vault.AuthMethod) (Keibi, error) {
	conf := vault.DefaultConfig()
	conf.Address = vaultAddress

	c, err := vault.NewClient(conf)
	if err != nil {
		return nil, fmt.Errorf("new vault: %w", err)
	}

	secret, err := c.Auth().Login(context.TODO(), auth)
	if err != nil {
		return nil, fmt.Errorf("vault authenticate: %w", err)
	}

	vault := &Vault{client: c}
	vault.watchSecret(secret)

	return vault, nil
}

func (v *Vault) watchSecret(s *vault.Secret) error {
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
				fmt.Printf("Successfully renewed: %#v\n", renewal)
			}
		}
	}()

	return nil
}

const (
	OrgRoot     string = "organizations"
	OrgRootPath string = "organizations/"
)

func (v *Vault) GetOrganizations() (pathRefs []string, err error) {
	secret, err := v.client.Logical().List(OrgRootPath)
	if err != nil {
		return nil, err // TODO: format & make meaningful
	}
	if secret == nil {
		return nil, fmt.Errorf("organizations root is not available or hasn't been configured yet") // TODO: format & make meaningful
	}

	orgIds := []string{}
	for _, val := range secret.Data["keys"].([]interface{}) {
		orgIds = append(orgIds, val.(string))
	}

	return orgIds, nil
}

func (v *Vault) NewOrganization(orgId uuid.UUID) (pathRef string, err error) {
	path := fmt.Sprintf("%s/%s", OrgRoot, orgId)
	secret, err := v.client.Logical().Write(path, map[string]interface{}{
		"metadata": "",
	})

	fmt.Println(secret)

	return path, err
}

func (v *Vault) WriteSourceConfig(orgId uuid.UUID, sourceId uuid.UUID, sourceType string, config interface{}) (configRef string, err error) {
	path := fmt.Sprintf("%s/%s/sources/%s/%s", OrgRoot, orgId, strings.ToLower(sourceType), sourceId)
	secret, err := v.client.Logical().Write(path, map[string]interface{}{
		"config": config,
	})

	fmt.Println(secret)

	return path, err
}

func (v *Vault) ReadSourceConfig(pathRef string) (config map[string]interface{}, err error) {
	secret, err := v.client.Logical().Read(pathRef)
	if err != nil {
		return nil, err
	}

	if secret == nil {
		return nil, fmt.Errorf("invalid pathRef: %s", pathRef)
	}

	return secret.Data["config"].(map[string]interface{}), nil
}

func (v *Vault) DeleteOrganization(pathRef string) error {
	_, err := v.client.Logical().Delete(pathRef)
	return err
}

func (v *Vault) DeleteSourceConfig(pathRef string) error {
	_, err := v.client.Logical().Delete(pathRef)
	return err
}
