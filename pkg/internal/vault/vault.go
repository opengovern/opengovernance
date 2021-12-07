package vault

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/kubernetes"
)

type KeibiVault interface {
	// GetOrganizations() (pathRefs []string, err error)

	DeleteOrganization(pathRef string) error
	NewOrganization(orgId uuid.UUID) (pathRef uuid.UUID, err error)

	DeleteSourceConfig(pathRef string) error
	ReadSourceConfig(pathRef string) (config interface{}, err error)
	WriteSourceConfig(orgId uuid.UUID, sourceId uuid.UUID, sourceType string, config interface{}) (configRef string, err error)
}

type Vault struct {
	client *vault.Client
}

func NewVault(vaultAddress string) (*Vault, error) {
	conf := vault.DefaultConfig()
	conf.Address = vaultAddress

	c, err := vault.NewClient(conf)
	if err != nil {
		return nil, fmt.Errorf("new vault: error occured")
	}

	return &Vault{client: c}, nil
}

func (v *Vault) AuthenticateUsingTokenPath(roleName string, tokenPath string) error {
	k8sAuth, _ := auth.NewKubernetesAuth(
		roleName,
		auth.WithServiceAccountTokenPath(tokenPath),
	)

	authInfo, err := v.client.Auth().Login(context.TODO(), k8sAuth)
	if err != nil {
		return fmt.Errorf("authenticate using token path error - %q", err.Error()) // TODO format & make meaningful
	}
	if authInfo == nil {
		return fmt.Errorf("authenticate using token path error") // TODO format & make meaningful
	}

	return nil
}

func (v *Vault) AuthenticateUsingJwt(roleName string, token string) error {
	k8sAuth, _ := auth.NewKubernetesAuth(
		roleName,
		auth.WithServiceAccountToken(token),
	)

	authInfo, err := v.client.Auth().Login(context.TODO(), k8sAuth)
	if err != nil {
		return fmt.Errorf("authenticate using token path error - %q", err.Error()) // TODO: format & make meaningful
	}
	if authInfo == nil {
		return fmt.Errorf("authenticate using token path error") // TODO: format & make meaningful
	}

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

func (v *Vault) ReadSourceConfig(pathRef string) (config interface{}, err error) {
	secret, err := v.client.Logical().Read(pathRef)
	config = secret.Data["config"].(string)

	return config, err
}

func (v *Vault) DeleteOrganization(pathRef string) error {
	_, err := v.client.Logical().Delete(pathRef)
	return err
}

func (v *Vault) DeleteSourceConfig(pathRef string) error {
	_, err := v.client.Logical().Delete(pathRef)
	return err
}
