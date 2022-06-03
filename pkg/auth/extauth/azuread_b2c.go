package extauth

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	maxResponseBody = 4 << 20 // 4MiB max body size to prevent reading some malformed responses
)

var graphApiURI = "https://graph.microsoft.com/v1.0/users/"

var (
	ErrUserNotExists = errors.New("user not exists in Azure service")
	ErrUserExists    = errors.New("user already exists in Azure service")
)

//go:generate mockery --name Provider
type Provider interface {
	CreateUser(ctx context.Context, email string) (AzureADUser, error)
	FetchUser(ctx context.Context, id string) (AzureADUser, error)
}

type azureB2CProvider struct {
	azureTenantID       string
	azureClientID       string
	azureClientSecret   string
	azureIdentityIssuer string
	token               string
	tokenExpire         time.Time
	logger              *zap.Logger
	tokenMutex          sync.RWMutex
}

func NewAzureADB2CProvider(
	ctx context.Context,
	azureTenantID,
	azureClientID,
	azureClientSecret,
	azureIdentityIssuer string,
	logger *zap.Logger,
) (Provider, error) {
	c := &azureB2CProvider{
		azureTenantID:       azureTenantID,
		azureClientID:       azureClientID,
		azureClientSecret:   azureClientSecret,
		azureIdentityIssuer: azureIdentityIssuer,
		logger:              logger,
	}
	err := c.refreshToken(ctx)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *azureB2CProvider) FetchUser(ctx context.Context, id string) (AzureADUser, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return AzureADUser{}, fmt.Errorf("error getting token for get user: %s. User id:%s", err, id)
	}
	user, err := c.getUser(ctx, id, token)
	if err != nil {
		if errors.Is(err, ErrUserNotExists) {
			return AzureADUser{}, ErrUserNotExists
		}
		return AzureADUser{}, fmt.Errorf("error getting user: %s. User id:%s", err, user.ID)
	}

	return user, nil
}

func (c *azureB2CProvider) CreateUser(ctx context.Context, email string) (AzureADUser, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return AzureADUser{}, fmt.Errorf("error getting token for create user: %s. User email:%s", err, email)
	}
	user, err := c.createUser(ctx, email, token)
	if err != nil {
		if errors.Is(err, ErrUserExists) {
			return AzureADUser{}, ErrUserExists
		}
		return AzureADUser{}, fmt.Errorf("error creating user: %s. User email:%s", err, email)
	}
	err = c.setChangePassword(ctx, user.ID, token)
	if err != nil {
		return AzureADUser{}, fmt.Errorf("error setting change password: %s. User email:%s", err, email)
	}

	return user, nil
}

func (c *azureB2CProvider) getUser(ctx context.Context, userID, token string) (AzureADUser, error) {
	if userID == "" {
		return AzureADUser{}, ErrUserNotExists
	}
	url := graphApiURI + userID
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		url,
		nil)
	if err != nil {
		c.logger.Error("getUser request",
			zap.String("requestURL", string(graphApiURI)),
			zap.Error(err))

		return AzureADUser{}, err
	}
	request.Header.Add("Authorization", token)
	request.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		c.logger.Error("getUser send request error",
			zap.Error(err))
		return AzureADUser{}, err
	}
	defer resp.Body.Close()

	statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !statusOK {

		err = handleAzureErrors(resp, userID)

		c.logger.Warn("getUser error",
			zap.String("requestURL", string(url)),
			zap.Error(err),
		)
		return AzureADUser{}, err
	}

	var user AzureADUser
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return AzureADUser{}, err
	}

	err = json.Unmarshal(body, &user)

	if err != nil {
		c.logger.Error("response body unmarshall error: ",
			zap.String("body", string(body)),
			zap.Error(err))

		return AzureADUser{}, err
	}

	c.logger.Info("User exists",
		zap.String("userEmail:", user.Mail),
		zap.String("userID:", user.ID))
	return user, nil
}

func (c *azureB2CProvider) createUser(ctx context.Context, email, token string) (AzureADUser, error) {
	password, err := generatePassword()
	if err != nil {
		return AzureADUser{}, err
	}
	userData, err := c.userData(email, password)
	if err != nil {
		c.logger.Error("creating user data error: ",
			zap.Error(err))
		return AzureADUser{}, err
	}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		graphApiURI,
		bytes.NewReader(userData))
	if err != nil {
		c.logger.Error("createUser request",
			zap.String("requestBody", string(userData)),
			zap.String("requestURL", string(graphApiURI)),
			zap.Error(err))

		return AzureADUser{}, err
	}

	request.Header.Add("Authorization", token)
	request.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		c.logger.Error("createUser send request error",
			zap.Error(err))
		return AzureADUser{}, err
	}
	defer resp.Body.Close()

	statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !statusOK {

		err = handleAzureErrors(resp, "")

		c.logger.Warn("createUser error",
			zap.String("requestBody", string(userData)),
			zap.String("requestURL", string(graphApiURI)),
			zap.Error(err),
		)
		return AzureADUser{}, err
	}

	var user AzureADUser
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return AzureADUser{}, err
	}

	err = json.Unmarshal(body, &user)
	if err != nil {
		c.logger.Error("response body unmarshall error: ",
			zap.String("body", string(body)),
			zap.Error(err))

		return AzureADUser{}, err
	}
	user.PasswordProfile.Password = password
	c.logger.Info("User created",
		zap.String("userEmail:", user.Mail),
		zap.String("userID:", user.ID))
	return user, nil
}

func (c *azureB2CProvider) setChangePassword(ctx context.Context, userID, token string) error {
	url := graphApiURI + userID
	patchUser := &AzureADUser{
		PasswordProfile: PasswordProfile{
			ForceChangePasswordNextSignIn: true,
		},
	}
	jsonBody, err := json.Marshal(patchUser)
	if err != nil {
		c.logger.Error("setChangePassword marshalling error",
			zap.Error(err))
		return err
	}
	request, err := http.NewRequest(
		http.MethodPatch,
		url,
		strings.NewReader(string(jsonBody)))
	if err != nil {
		c.logger.Error("setChangePassword request",
			zap.String("requestBody", string(jsonBody)),
			zap.String("requestURL", string(request.RequestURI)),
			zap.Error(err))

		return err
	}
	request = request.WithContext(ctx)
	request.Header.Add("Authorization", token)
	request.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		c.logger.Error("setChangePassword send request error",
			zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !statusOK {
		err = handleAzureErrors(resp, userID)
		c.logger.Warn("setChangePassword error",
			zap.String("requestBody", string(jsonBody)),
			zap.String("requestURL", string(url)),
			zap.Error(err),
		)

		return err
	}
	c.logger.Info("ForceChangePassword updated")
	return nil
}

func (c *azureB2CProvider) getToken(ctx context.Context) (string, error) {
	c.tokenMutex.RLock()
	if c.tokenExpire.After(time.Now()) {
		defer c.tokenMutex.RUnlock()
		return c.token, nil
	}
	c.tokenMutex.RUnlock()

	c.tokenMutex.Lock()
	defer c.tokenMutex.Unlock()

	// double check resource locking
	if c.tokenExpire.After(time.Now()) {
		return c.token, nil
	}
	err := c.refreshToken(ctx)
	if err != nil {
		c.logger.Error("refreshToken error: ",
			zap.Error(err))
		return "", err
	}
	secret, err := maskSecret(c.azureClientID)
	if err != nil {
		c.logger.Error("mask secret error: ",
			zap.Error(err))
		return "", err
	}
	c.logger.Info("token refreshed for",
		zap.String("clientID", secret))
	return c.token, nil
}

func (c *azureB2CProvider) refreshToken(ctx context.Context) error {
	cred, err := azidentity.NewClientSecretCredential(
		c.azureTenantID,
		c.azureClientID,
		c.azureClientSecret,
		nil)
	if err != nil {
		return err
	}
	policy := policy.TokenRequestOptions{Scopes: []string{"https://graph.microsoft.com/.default"}}

	accesToken, err := cred.GetToken(ctx, policy)
	if err != nil {
		return err
	}

	c.token = "Bearer " + accesToken.Token
	c.tokenExpire = accesToken.ExpiresOn
	return nil
}

func (c *azureB2CProvider) userData(email, password string) ([]byte, error) {
	user := &AzureADUser{
		AccountEnabled: true,
		DisplayName:    email,
		Mail:           email,
		Identities: []Identities{
			{
				SignInType:       "emailAddress",
				Issuer:           c.azureIdentityIssuer,
				IssuerAssignedId: email,
			},
		},
		Creationtype: "LocalAccount",
		PasswordProfile: PasswordProfile{
			Password:                      password,
			ForceChangePasswordNextSignIn: false,
		},
		PasswordPolicies: "DisablePasswordExpiration",
	}

	jsonUserData, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}
	return jsonUserData, nil
}

func handleAzureErrors(resp *http.Response, userInfo string) error {
	var azureErrors AzureErrors
	err := json.NewDecoder(resp.Body).Decode(&azureErrors)
	if err != nil {
		return err
	}
	switch {
	case strings.Contains(azureErrors.Error.Message, fmt.Sprintf("Resource '%s' does not exist ", userInfo)):
		return fmt.Errorf("%w : user id :%s ", ErrUserNotExists, userInfo)
	case strings.Contains(azureErrors.Error.Message, "Another object with the same value "):
		return fmt.Errorf("%w : email :%s ", ErrUserExists, userInfo)
	}
	return errors.New(azureErrors.Error.Message)
}

func maskSecret(secret string) (string, error) {
	if len(secret) > 4 {
		return secret[:4] + "****", nil
	}
	return "", fmt.Errorf("incorrect secret value")
}

func generatePassword() (string, error) {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const digits = "0123456789"
	randomLetters := make([]byte, 6)
	lettersLength := big.NewInt(int64(len(letters)))
	digitsLength := big.NewInt(int64(len(digits)))

	for i := 0; i < 6; i++ {
		num, err := rand.Int(rand.Reader, lettersLength)
		if err != nil {
			return "", err
		}
		randomLetters[i] = letters[num.Int64()]
	}

	randomDigits := make([]byte, 4)
	for i := 0; i < 4; i++ {
		num, err := rand.Int(rand.Reader, digitsLength)
		if err != nil {
			return "", err
		}
		randomDigits[i] = digits[num.Int64()]
	}

	titleLetter := cases.Title(language.Und)
	password := titleLetter.String(string(randomLetters)) + string(randomDigits)

	return password, nil
}
