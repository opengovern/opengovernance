package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt"
	_ "github.com/golang-jwt/jwt/v4"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	urls "gitlab.com/keibiengine/keibi-engine/pkg/cli/consts"
	apiOnboard "gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"
	workspace "gitlab.com/keibiengine/keibi-engine/pkg/workspace/api"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

func GetConfig(cmd *cobra.Command, workspaceNameRequired bool) (*Config, error) {
	home := os.Getenv("HOME")
	data, err := os.ReadFile(home + "/.kaytu/config.json")
	if err != nil {
		return nil, fmt.Errorf("[getConfig] : %v", err)

	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("[getConfig] : %v", err)
	}

	if config.AccessToken == "" {
		return nil, fmt.Errorf("please log in first")
	}

	checkEXP, err := CheckExpirationTime(config.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("[getConfig] : %v", err)
	}
	if checkEXP == true {
		return nil, fmt.Errorf("accessToken was expire please loging againe ")
	}
	if workspaceNameRequired {
		workspaceName := cmd.Flags().Lookup("workspace-name").Value.String()
		if workspaceName != "" {
			config.DefaultWorkspace = workspaceName
		} else {
			config.DefaultWorkspace = "demo"
		}
	}

	return &config, nil
}

func RemoveConfig() error {
	home := os.Getenv("HOME")
	err := os.Remove(home + "/.kaytu/config.json")
	if err != nil {
		return fmt.Errorf("[removeConfig] : %v", err)
	}
	return nil
}

func AddConfig(accessToken string) error {
	var data Config
	data.AccessToken = accessToken
	configs, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("[addConfig] : %v", err)
	}
	home := os.Getenv("HOME")
	_, err = os.Stat(home + "/.kaytu")
	if err != nil {
		err = os.Mkdir(home+"/.kaytu", os.ModePerm)
		if err != nil {
			return fmt.Errorf("[addConfig] : %v", err)
		}
	}

	err = os.WriteFile(home+"/.kaytu/config.json", configs, os.ModePerm)
	if err != nil {
		return fmt.Errorf("[addConfig] : %v", err)
	}
	return nil
}

func RequestAbout(accessToken string) (ResponseAbout, error) {
	req, err := http.NewRequest("GET", urls.UrlAbout, nil)
	if err != nil {
		return ResponseAbout{}, fmt.Errorf("[requestAbout] : %v", err)
	}
	req.Header.Add("content-type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return ResponseAbout{}, fmt.Errorf("[requestAbout] : %v", err)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return ResponseAbout{}, fmt.Errorf("[requestAbout] : %v", err)
	}
	err = res.Body.Close()
	if err != nil {
		return ResponseAbout{}, fmt.Errorf("[requestAbout] : %v", err)
	}
	response := ResponseAbout{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return ResponseAbout{}, fmt.Errorf("[requestAbout] : %v", err)
	}
	return response, nil
}

func RequestDeviceCode() (string, error) {
	payload := DeviceCodeRequest{
		ClientId: ClientID,
		Scope:    "openid profil email api:read",
		Audience: "https://app.keibi.io",
	}
	payloadEncode, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("[requestDeviceCode] : %v", err)
	}

	req, err := http.NewRequest("POST", urls.UrlLogin+"/oauth/device/code", bytes.NewBuffer(payloadEncode))
	if err != nil {
		return "", fmt.Errorf("[requestDeviceCode] : %v", err)
	}
	req.Header.Add("content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("[requestDeviceCode] : %v", err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("[requestDeviceCode] : %v", err)
	}
	err = res.Body.Close()
	if err != nil {
		return "", fmt.Errorf("[requestDeviceCode] : %v", err)

	}

	response := DeviceCodeResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", fmt.Errorf("[requestDeviceCode] : %v", err)
	}

	fmt.Println("open this url in your browser:")
	fmt.Println(response.VerificationUrlComplete)

	return response.DeviceCode, nil
}

func AccessToken(deviceCode string) (string, error) {
	payload := RequestAccessToken{
		GrantType:  "urn:ietf:params:oauth:grant-type:device_code",
		DeviceCode: deviceCode,
		ClientId:   ClientID,
	}

	for {
		payloadEncoded, err := json.Marshal(payload)
		if err != nil {
			fmt.Printf("[AccessToken]: %v", err)
			time.Sleep(TimeSleep)
			continue
		}

		req, err := http.NewRequest("POST", urls.UrlLogin+"/oauth/token", bytes.NewBuffer(payloadEncoded))
		if err != nil {
			fmt.Printf("[AccessToken]: %v", err)
			time.Sleep(TimeSleep)
			continue
		}
		req.Header.Add("content-type", "application/json")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			time.Sleep(TimeSleep)
			continue
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Printf("[AccessToken]: %v", err)
			time.Sleep(TimeSleep)
			continue
		}

		err = res.Body.Close()
		if err != nil {
			fmt.Printf("[AccessToken]: %v", err)
			time.Sleep(TimeSleep)
			continue
		}

		response := ResponseAccessToken{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			fmt.Printf("[AccessToken]: %v", err)
			time.Sleep(TimeSleep)
			continue
		}

		if response.AccessToken != "" {
			return response.AccessToken, nil
		} else {
			time.Sleep(TimeSleep)
			continue
		}
	}
}

func CheckExpirationTime(accessToken string) (bool, error) {
	token, _, err := new(
		jwt.Parser).ParseUnverified(accessToken, jwt.MapClaims{})
	if err != nil {
		return false, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false, err
	}

	var tm time.Time
	switch iat := claims["exp"].(type) {
	case float64:
		tm = time.Unix(int64(iat), 0)
	case json.Number:
		v, _ := iat.Int64()
		tm = time.Unix(v, 0)
	}
	timeNow := time.Now()
	if tm.Before(timeNow) {
		return true, nil
	} else if tm.After(timeNow) {
		return false, nil
	} else {
		return true, err
	}
}

func RequestWorkspaces(accessToken string) ([]workspace.WorkspaceResponse, error) {
	req, err := http.NewRequest("GET", urls.UrlWorkspace, nil)
	if err != nil {
		return nil, fmt.Errorf("[RequestWorkspaces] : %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Add("content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[RequestWorkspaces] : %v", err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("[RequestWorkspaces] : %v", err)
	}

	err = res.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("[RequestWorkspaces] : %v", err)
	}
	var responseUnmarshal []workspace.WorkspaceResponse
	err = json.Unmarshal(body, &responseUnmarshal)
	if err != nil {
		return nil, fmt.Errorf("[RequestWorkspaces] : %v", err)
	}

	return responseUnmarshal, nil
}

func IamGetUsers(workspaceName string, accessToken string, email string, emailVerified bool, role string) ([]api.GetUserResponse, error) {
	roleTypeRole := api.Role(role)
	request := RequestGetIamUsers{
		email,
		emailVerified,
		roleTypeRole,
	}
	reqBody, err := json.Marshal(request)
	if err != nil {
		return []api.GetUserResponse{{}}, err
	}
	req, err := http.NewRequest("GET", urls.Url+workspaceName+"/auth/api/v1/users", bytes.NewBuffer(reqBody))
	if err != nil {
		return []api.GetUserResponse{{}}, err
	}
	req.Header.Add("Content-type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return []api.GetUserResponse{{}}, err
	}
	bodyResponse, err := io.ReadAll(res.Body)
	if err != nil {
		return []api.GetUserResponse{{}}, err
	}
	if res.StatusCode != http.StatusOK {
		return []api.GetUserResponse{{}}, fmt.Errorf("[IamGetUsers] invalid status code: %d, body=%s", res.StatusCode, string(bodyResponse))
	}
	err = res.Body.Close()
	if err != nil {
		return []api.GetUserResponse{{}}, err
	}
	var response []api.GetUserResponse
	err = json.Unmarshal(bodyResponse, &response)
	if err != nil {
		return []api.GetUserResponse{{}}, err
	}
	return response, nil
}

func IamGetUserDetails(accessToken string, workspaceName string, userId string) (ResponseUserDetails, error) {
	req, err := http.NewRequest("GET", urls.Url+workspaceName+"/auth/api/v1/user/"+userId, nil)
	if err != nil {
		return ResponseUserDetails{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Add("content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return ResponseUserDetails{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return ResponseUserDetails{}, err
	}
	if res.StatusCode != http.StatusOK {
		return ResponseUserDetails{}, fmt.Errorf("[IamGetUserDetails] invalid status code: %d, body=%s", res.StatusCode, string(body))
	}
	err = res.Body.Close()
	if err != nil {
		return ResponseUserDetails{}, err
	}
	var response ResponseUserDetails
	err = json.Unmarshal(body, &response)
	if err != nil {
		return ResponseUserDetails{}, err
	}
	return response, nil
}

func IamDeleteUserInvite(workspacesName string, accessToken string, userId string) (string, error) {
	req, err := http.NewRequest("DELETE", urls.Url+workspacesName+"/auth/api/v1/user/invite", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-Type", "application/json")
	query := req.URL.Query()
	query.Set("userId", userId)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	err = res.Body.Close()
	if err != nil {
		return "", err
	}
	if res.StatusCode == http.StatusOK {
		return "user invite deleted ", nil
	} else {
		return "deleting user was fail", fmt.Errorf("[IamDeleteUserInvite] invalid status code: %d", res.StatusCode)
	}
}

func IamDeleteUserAccess(workspacesName string, accessToken string, userId string) (string, error) {
	req, err := http.NewRequest("DELETE", urls.Url+workspacesName+"/auth/api/v1/user/role/binding", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-Type", "application/json")
	query := req.URL.Query()
	query.Set("userId", userId)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	err = res.Body.Close()
	if err != nil {
		return "", err
	}
	if res.StatusCode == http.StatusOK {
		return "user access deleted ", nil
	} else {
		return "deleting user was fail", fmt.Errorf("[IamDeleteUserAccess] invalid status code: %d", res.StatusCode)
	}
}

func IamCreateUser(workspaceName string, accessToken string, email string, role string) (string, error) {
	request := api.InviteRequest{
		Email:    email,
		RoleName: api.Role(role),
	}
	reqBody, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("GET", urls.Url+workspaceName+"/auth/api/v1/user/invite", bytes.NewBuffer(reqBody))
	req.Header.Set("Authorization", "Bearer "+accessToken)

	req.Header.Add("Content-type", "application/json")
	if err != nil {
		return "", err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	if res.StatusCode == http.StatusOK {
		return "user created successfully", nil
	} else {
		fmt.Println("status : ", res.Status)
		return "creat user was fail", nil
	}
}

func IamUpdateUser(workspaceName string, accessToken string, role string, userID string) (string, error) {
	request := api.PutRoleBindingRequest{RoleName: api.Role(role), UserID: userID}
	reqBody, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("PUT", urls.Url+workspaceName+"/auth/api/v1/user/role/binding", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	err = res.Body.Close()
	if err != nil {
		return "", err
	}
	if res.StatusCode == http.StatusOK {
		return "user updated successfully ", nil
	} else {
		return "updating user was fail", fmt.Errorf("[IamUpdateUser] invalid status code: %d", res.StatusCode)
	}
}

func IamListRoles(WorkspacesName string, accessToken string) ([]RolesListResponse, error) {
	req, err := http.NewRequest("GET", urls.Url+WorkspacesName+"/auth/api/v1/roles", nil)
	if err != nil {
		return []RolesListResponse{{}}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return []RolesListResponse{{}}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return []RolesListResponse{{}}, err
	}
	if res.StatusCode != http.StatusOK {
		return []RolesListResponse{{}}, fmt.Errorf("[IamListRoles] invalid status code: %d, body : %v", res.StatusCode, string(body))
	}
	err = res.Body.Close()
	if err != nil {
		return []RolesListResponse{{}}, err
	}
	var response []RolesListResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return []RolesListResponse{{}}, err
	}
	return response, nil
}
func IamListRoleKeys(WorkspacesName string, accessToken string, roleName string) ([]api.WorkspaceApiKey, error) {
	req, err := http.NewRequest("GET", urls.Url+WorkspacesName+"/auth/api/v1/role/"+roleName+"/keys", nil)
	if err != nil {
		return []api.WorkspaceApiKey{{}}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return []api.WorkspaceApiKey{{}}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return []api.WorkspaceApiKey{{}}, err
	}
	if res.StatusCode != http.StatusOK {
		return []api.WorkspaceApiKey{{}}, fmt.Errorf("[IamListRoleKeys] invalid status code: %d, body : %v", res.StatusCode, string(body))
	}
	err = res.Body.Close()
	if err != nil {
		return []api.WorkspaceApiKey{{}}, err
	}
	var response []api.WorkspaceApiKey
	err = json.Unmarshal(body, &response)
	if err != nil {
		return []api.WorkspaceApiKey{{}}, err
	}
	return response, nil
}
func IamListRoleUsers(WorkspacesName string, accessToken string, roleName string) (api.GetRoleUsersResponse, error) {
	req, err := http.NewRequest("GET", urls.Url+WorkspacesName+"/auth/api/v1/role/"+roleName+"/users", nil)
	if err != nil {
		return api.GetRoleUsersResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return api.GetRoleUsersResponse{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return api.GetRoleUsersResponse{}, err
	}
	if res.StatusCode != http.StatusOK {
		return api.GetRoleUsersResponse{}, fmt.Errorf("[IamListRoleUsers] invalid status code: %d, body : %v", res.StatusCode, string(body))
	}
	err = res.Body.Close()
	if err != nil {
		return api.GetRoleUsersResponse{}, err
	}
	var response api.GetRoleUsersResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return api.GetRoleUsersResponse{}, err
	}
	return response, nil
}

func IamRoleDetails(workspaceName string, roleName string, accessToken string) (api.RoleDetailsResponse, error) {
	req, err := http.NewRequest("GET", urls.Url+workspaceName+"/auth/api/v1/roles/"+roleName, nil)
	if err != nil {
		return api.RoleDetailsResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return api.RoleDetailsResponse{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return api.RoleDetailsResponse{}, err
	}
	if res.StatusCode != http.StatusOK {
		return api.RoleDetailsResponse{}, fmt.Errorf("[IamRoleDetails] invalid status code: %d, body : %v", res.StatusCode, string(body))
	}
	err = res.Body.Close()
	if err != nil {
		return api.RoleDetailsResponse{}, err
	}
	var response api.RoleDetailsResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return api.RoleDetailsResponse{}, err
	}
	return response, nil
}

func IamGetListKeys(workspacesName string, accessToken string) ([]api.WorkspaceApiKey, error) {
	req, err := http.NewRequest("GET", urls.Url+workspacesName+"/auth/api/v1/keys", nil)
	if err != nil {
		return []api.WorkspaceApiKey{{}}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return []api.WorkspaceApiKey{{}}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return []api.WorkspaceApiKey{{}}, err
	}
	if res.StatusCode != http.StatusOK {
		return []api.WorkspaceApiKey{{}}, fmt.Errorf("[IamGetListKeys] invalid status code: %d, body : %v", res.StatusCode, string(body))
	}
	err = res.Body.Close()
	if err != nil {
		return []api.WorkspaceApiKey{{}}, err
	}
	var response []api.WorkspaceApiKey
	err = json.Unmarshal(body, &response)
	if err != nil {
		return []api.WorkspaceApiKey{{}}, err
	}
	return response, nil
}

func IamGetKeyDetails(workspacesName string, accessToken string, id string) (api.WorkspaceApiKey, error) {
	req, err := http.NewRequest("GET", urls.Url+workspacesName+"/auth/api/v1/key/"+id, nil)
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	if res.StatusCode != http.StatusOK {
		return api.WorkspaceApiKey{}, fmt.Errorf("[IamGetKeyDetails] invalid status code: %d, body : %v", res.StatusCode, string(body))
	}
	err = res.Body.Close()
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	var response api.WorkspaceApiKey
	err = json.Unmarshal(body, &response)
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	return response, nil
}

func IamCreateKeys(workspacesName string, accessToken string, keyName string, role string) (api.CreateAPIKeyResponse, error) {
	var request api.CreateAPIKeyRequest
	request.RoleName = api.Role(role)
	request.Name = keyName
	reqBody, err := json.Marshal(request)
	if err != nil {
		return api.CreateAPIKeyResponse{}, err
	}
	req, err := http.NewRequest("POST", urls.Url+workspacesName+"/auth/api/v1/key/create", bytes.NewBuffer(reqBody))
	if err != nil {
		return api.CreateAPIKeyResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return api.CreateAPIKeyResponse{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return api.CreateAPIKeyResponse{}, err
	}
	if res.StatusCode != http.StatusOK {
		return api.CreateAPIKeyResponse{}, fmt.Errorf("[IamCreateKeys] invalid status code: %d, body : %v", res.StatusCode, string(body))
	}
	err = res.Body.Close()
	if err != nil {
		return api.CreateAPIKeyResponse{}, err
	}
	var response api.CreateAPIKeyResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return api.CreateAPIKeyResponse{}, err
	}
	return response, nil
}

func IamUpdateKeyRole(workspacesName string, accessToken string, id uint, role string) (api.WorkspaceApiKey, error) {
	request := api.UpdateKeyRoleRequest{ID: id, RoleName: api.Role(role)}
	reqBody, err := json.Marshal(request)
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	req, err := http.NewRequest("POST", urls.Url+workspacesName+"/auth/api/v1/key/role", bytes.NewBuffer(reqBody))
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	if res.StatusCode != http.StatusOK {
		return api.WorkspaceApiKey{}, fmt.Errorf("[IamUpdateKeyRole] invalid status code: %d, body : %v", res.StatusCode, string(body))
	}
	err = res.Body.Close()
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	response := api.WorkspaceApiKey{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	return response, nil
}

func IamSuspendKey(workspaceName string, accessToken string, id string) (api.WorkspaceApiKey, error) {
	req, err := http.NewRequest("POST", urls.Url+workspaceName+"/auth/api/v1/key/"+id+"/suspend", nil)
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	body, err := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return api.WorkspaceApiKey{}, fmt.Errorf("[IamSuspendKey] invalid status code: %d, body : %v", res.StatusCode, string(body))
	}
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	err = res.Body.Close()
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	var response api.WorkspaceApiKey
	err = json.Unmarshal(body, &response)
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	return response, nil
}

func IamActivateKey(workspaceName string, accessToken string, id string) (api.WorkspaceApiKey, error) {
	req, err := http.NewRequest("POST", urls.Url+workspaceName+"/auth/api/v1/key/"+id+"/activate", nil)
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	if res.StatusCode != http.StatusOK {
		return api.WorkspaceApiKey{}, fmt.Errorf("[IamActivateKey] invalid status code: %d, body : %v", res.StatusCode, string(body))
	}
	err = res.Body.Close()
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	var response api.WorkspaceApiKey
	err = json.Unmarshal(body, &response)
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	return response, nil
}

func IamDeleteKey(workspacesName string, accessToken string, id string) (string, error) {
	req, err := http.NewRequest("DELETE", urls.Url+workspacesName+"/auth/api/v1/key/"+id+"/delete", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	if res.StatusCode == http.StatusOK {
		return "key successfully deleted ", nil
	} else {
		return "deleting key was fail", nil
	}
}

func OnboardCreateAWS(workspaceName string, accessToken string, name string, email string, description string, accessKey string, accessId string, regions []string, secretKey string) (apiOnboard.CreateSourceResponse, error) {
	var bodyRequest apiOnboard.SourceAwsRequest
	bodyRequest.Name = name
	bodyRequest.Email = email
	bodyRequest.Description = description
	bodyRequest.Config.AccessKey = accessKey
	bodyRequest.Config.Regions = regions
	bodyRequest.Config.AccountId = accessId
	bodyRequest.Config.SecretKey = secretKey
	reqBodyEncoded, err := json.Marshal(bodyRequest)
	if err != nil {
		return apiOnboard.CreateSourceResponse{}, err
	}
	req, err := http.NewRequest("POST", urls.Url+workspaceName+"/onboard/api/v1/source/aws", bytes.NewBuffer(reqBodyEncoded))
	if err != nil {
		return apiOnboard.CreateSourceResponse{}, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return apiOnboard.CreateSourceResponse{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return apiOnboard.CreateSourceResponse{}, err
	}
	var response apiOnboard.CreateSourceResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return apiOnboard.CreateSourceResponse{}, err
	}
	if res.StatusCode != http.StatusOK {
		fmt.Println("failed creating AWS source.")
		return apiOnboard.CreateSourceResponse{}, err
	}
	err = res.Body.Close()
	if err != nil {
		return apiOnboard.CreateSourceResponse{}, err
	}
	return response, nil
}

func OnboardCreateAzure(workspaceName string, accessToken string, name string, ObjectId string, description string, clientId string, clientSecret string, subscriptionId string, tenantId string) (ResponseCreateAzure, error) {
	var request apiOnboard.SourceAzureRequest
	request.Name = name
	request.Description = description
	request.Config.ClientId = clientId
	request.Config.ClientSecret = clientSecret
	request.Config.ClientSecret = clientSecret
	request.Config.SubscriptionId = subscriptionId
	request.Config.TenantId = tenantId
	request.Config.ObjectId = ObjectId
	reqBodyEncoded, err := json.Marshal(request)
	if err != nil {
		return ResponseCreateAzure{}, err
	}
	req, err := http.NewRequest("POST", urls.Url+workspaceName+"/onboard/api/v1/source/azure", bytes.NewBuffer(reqBodyEncoded))
	if err != nil {
		return ResponseCreateAzure{}, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return ResponseCreateAzure{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return ResponseCreateAzure{}, err
	}
	if res.StatusCode != http.StatusOK {
		fmt.Println("failed creating AWS source.")
		return ResponseCreateAzure{}, err
	}
	err = res.Body.Close()
	if err != nil {
		return ResponseCreateAzure{}, err
	}
	var response ResponseCreateAzure
	err = json.Unmarshal(body, &response)
	if err != nil {
		return ResponseCreateAzure{}, err
	}
	return response, nil
}

func OnboardCatalogConnectors(workspaceName string, accessToken string, idFilter string, minimumConnectionFilter string, stateFilter string, categoryFilter string) ([]apiOnboard.CatalogConnector, error) {
	req, err := http.NewRequest("GET", urls.Url+workspaceName+"/onboard/api/v1/catalog/connectors", nil)
	if err != nil {
		return []apiOnboard.CatalogConnector{}, err
	}
	query := req.URL.Query()
	query.Set("category", categoryFilter)
	query.Set("state", stateFilter)
	query.Set("minConnection", minimumConnectionFilter)
	query.Set("id", idFilter)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return []apiOnboard.CatalogConnector{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return []apiOnboard.CatalogConnector{}, err
	}
	err = res.Body.Close()
	if err != nil {
		return []apiOnboard.CatalogConnector{}, err
	}
	var response []apiOnboard.CatalogConnector
	err = json.Unmarshal(body, &response)
	if err != nil {
		return []apiOnboard.CatalogConnector{}, err
	}
	return response, nil
}

func OnboardCatalogMetrics(workspaceName string, accessToken string) (apiOnboard.CatalogMetrics, error) {
	req, err := http.NewRequest("GET", urls.Url+workspaceName+"/onboard/api/v1/catalog/metrics", nil)
	if err != nil {
		return apiOnboard.CatalogMetrics{}, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return apiOnboard.CatalogMetrics{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return apiOnboard.CatalogMetrics{}, err
	}
	var response apiOnboard.CatalogMetrics
	err = res.Body.Close()
	if err != nil {
		return apiOnboard.CatalogMetrics{}, err
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return apiOnboard.CatalogMetrics{}, err
	}
	return response, nil
}

func OnboardCountConnections(accessToken string, workspaceName string, connectorsNames []string, health string, state string) (string, error) {
	request := CountConnectionsCLIRequest{
		connectorsNames,
		state,
		health,
	}
	reqEncoded, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", urls.Url+workspaceName+"/onboard/api/v1/connections/count", bytes.NewBuffer(reqEncoded))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	err = res.Body.Close()
	if err != nil {
		return "", err
	}
	return string(body), nil
}
func OnboardGetConnectors(workspaceName string, accessToken string) ([]apiOnboard.ConnectorCount, error) {
	req, err := http.NewRequest("GET", urls.Url+workspaceName+"/onboard/api/v1/connectors", nil)
	if err != nil {
		return []apiOnboard.ConnectorCount{}, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return []apiOnboard.ConnectorCount{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return []apiOnboard.ConnectorCount{}, err
	}
	err = res.Body.Close()
	if err != nil {
		return []apiOnboard.ConnectorCount{}, err
	}
	var response []apiOnboard.ConnectorCount
	err = json.Unmarshal(body, &response)
	if err != nil {
		return []apiOnboard.ConnectorCount{}, err
	}
	return response, nil
}
func OnboardGetConnector(workspaceName string, accessToken string, connectorName string) (apiOnboard.ConnectorCount, error) {
	req, err := http.NewRequest("GET", urls.Url+workspaceName+"/onboard/api/v1/connectors/"+connectorName, nil)
	if err != nil {
		return apiOnboard.ConnectorCount{}, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return apiOnboard.ConnectorCount{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return apiOnboard.ConnectorCount{}, err
	}
	err = res.Body.Close()
	if err != nil {
		return apiOnboard.ConnectorCount{}, err
	}
	var response apiOnboard.ConnectorCount
	err = json.Unmarshal(body, &response)
	if err != nil {
		return apiOnboard.ConnectorCount{}, err
	}
	return response, nil
}
func OnboardGetListCredentialsByFilter(workspacesName string, accessToken string, connectorType string, healthStatus string, pageSize string, pageNumber string) ([]apiOnboard.Credential, error) {
	req, err := http.NewRequest("GET", urls.Url+workspacesName+"/onboard/api/v1/credential", nil)
	if err != nil {
		return []apiOnboard.Credential{}, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	query := req.URL.Query()
	query.Set("connector", connectorType)
	query.Set("health", healthStatus)
	query.Set("pageSize", pageSize)
	query.Set("pageNumber", pageNumber)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return []apiOnboard.Credential{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return []apiOnboard.Credential{}, err
	}
	var response []apiOnboard.Credential
	err = json.Unmarshal(body, &response)
	if err != nil {
		return []apiOnboard.Credential{}, err
	}
	return response, err
}
func OnboardCreateConnectionCredentials(workspaceName string, accessToken string, config string, name string, sourceType string) (apiOnboard.CreateCredentialResponse, error) {
	var request requestCreateConnectionCredentials
	request.Name = name
	request.SourceType = sourceType
	request.Config = config
	reqEncoded, err := json.Marshal(request)
	if err != nil {
		return apiOnboard.CreateCredentialResponse{}, err
	}
	req, err := http.NewRequest("POST", urls.Url+workspaceName+"/onboard/api/v1/credential", bytes.NewBuffer(reqEncoded))
	if err != nil {
		return apiOnboard.CreateCredentialResponse{}, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return apiOnboard.CreateCredentialResponse{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return apiOnboard.CreateCredentialResponse{}, err
	}
	err = res.Body.Close()
	if err != nil {
		return apiOnboard.CreateCredentialResponse{}, err
	}
	var response apiOnboard.CreateCredentialResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return apiOnboard.CreateCredentialResponse{}, err
	}
	return response, nil
}
func OnboardGetCredentialById(workspaceName string, accessToken string, credentialId string) ([]apiOnboard.Credential, error) {
	req, err := http.NewRequest("GET", urls.Url+workspaceName+"/onboard/api/v1/credential/"+credentialId, nil)
	if err != nil {
		return []apiOnboard.Credential{}, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return []apiOnboard.Credential{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return []apiOnboard.Credential{}, err
	}
	err = res.Body.Close()
	if err != nil {
		return []apiOnboard.Credential{}, err
	}
	var response []apiOnboard.Credential
	err = json.Unmarshal(body, &response)
	if err != nil {
		return []apiOnboard.Credential{}, err
	}
	return response, nil
}
func OnboardEditeCredentialById(workspaceName string, accessToken string, config string, connector string, name string, credentialId string) error {
	var request requestEditeCredentialById
	request.Name = name
	request.Config = config
	request.Connector = connector
	reqBody, err := json.Marshal(request)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", urls.Url+workspaceName+"/onboard/api/v1/credential/"+credentialId, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	err = res.Body.Close()
	if err != nil {
		return err
	}
	return nil
}
func OnboardDeleteCredential(workspaceName string, accessToken string, credentialId string) error {
	req, err := http.NewRequest("DELETE", urls.Url+workspaceName+"/onboard/api/v1/credential/"+credentialId, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	err = res.Body.Close()
	if err != nil {
		return err
	}
	return nil
}
func OnboardGetCredentialAvailableConnections(workspaceName string, accessToken string, credentialId string) ([]apiOnboard.Source, error) {
	req, err := http.NewRequest("POST", urls.Url+workspaceName+"/onboard/api/v1/credential/"+credentialId+"/autoonboard", nil)
	if err != nil {
		return []apiOnboard.Source{}, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("Authorization", "bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return []apiOnboard.Source{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return []apiOnboard.Source{}, err
	}
	err = res.Body.Close()
	if err != nil {
		return []apiOnboard.Source{}, err
	}
	var response []apiOnboard.Source
	err = json.Unmarshal(body, &response)
	if err != nil {
		return []apiOnboard.Source{}, err
	}
	return response, nil
}
func OnboardDisableCredential(workspaceName string, accessToken string, credentialId string) error {
	req, err := http.NewRequest("POST", urls.Url+workspaceName+"/onboard/api/v1/credential/"+credentialId+"/disable", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	err = res.Body.Close()
	if err != nil {
		return err
	}
	return nil
}
func OnboardEnableCredential(workspaceName string, accessToken string, credentialId string) error {
	req, err := http.NewRequest("POST", urls.Url+workspaceName+"/onboard/api/v1/credential/"+credentialId+"/enable", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	err = res.Body.Close()
	if err != nil {
		return err
	}
	return nil
}
func OnboardGetLiveCredentialHealth(workspaceName string, accessToken string, credentialId string) error {
	req, err := http.NewRequest("POST", urls.Url+workspaceName+"/onboard/api/v1/credential/"+credentialId+"/healthcheck", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	err = res.Body.Close()
	if err != nil {
		return err
	}
	return nil
}
func OnboardGetProviders(workspaceName string, accessToken string) (apiOnboard.ProvidersResponse, error) {
	req, err := http.NewRequest("GET", urls.Url+workspaceName+"/onboard/api/v1/providers", nil)
	if err != nil {
		return apiOnboard.ProvidersResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return apiOnboard.ProvidersResponse{}, err
	}
	var response apiOnboard.ProvidersResponse
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return apiOnboard.ProvidersResponse{}, err
	}
	err = res.Body.Close()
	if err != nil {
		return apiOnboard.ProvidersResponse{}, err
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return apiOnboard.ProvidersResponse{}, err
	}
	return response, nil
}
func OnboardGetProviderTypes(workspaceName string, accessToken string) (apiOnboard.ProviderTypesResponse, error) {
	req, err := http.NewRequest("GET", urls.Url+workspaceName+"/onboard/api/v1/providers/types", nil)
	if err != nil {
		return apiOnboard.ProviderTypesResponse{}, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return apiOnboard.ProviderTypesResponse{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return apiOnboard.ProviderTypesResponse{}, err
	}
	err = res.Body.Close()
	if err != nil {
		return apiOnboard.ProviderTypesResponse{}, err
	}
	var response apiOnboard.ProviderTypesResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return apiOnboard.ProviderTypesResponse{}, err
	}
	return response, nil
}
func OnboardPutSourceCredential(workspaceName string, accessToken string, sourceId string) error {
	req, err := http.NewRequest("PUT", urls.Url+workspaceName+"/onboard/api/v1/source/"+sourceId+"/credentials", nil)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	statusCode := res.StatusCode
	err = res.Body.Close()
	if err != nil {
		return err
	}
	if statusCode == http.StatusOK {
		return nil
	} else {
		return fmt.Errorf("error with status : %v", statusCode)
	}
}
func OnboardCountSources(workspaceName string, accessToken string, connector string) (string, error) {
	req, err := http.NewRequest("GET", urls.Url+workspaceName+"/onboard/api/v1/sources/count", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("connector", connector)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	err = res.Body.Close()
	if err != nil {
		return "", err
	}
	return string(body), nil
}
func OnboardDeleteSource(workspaceName string, accessToken string, sourceId string) error {
	req, err := http.NewRequest("DELETE", urls.Url+workspaceName+"/onboard/api/v1/source/"+sourceId, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode == http.StatusOK {
		return nil
	} else {
		return fmt.Errorf("exist some error with status code : %v ", res.StatusCode)
	}
}
func OnboardGetSingleSource(workspaceName string, accessToken string, sourceId string) (apiOnboard.Source, error) {
	req, err := http.NewRequest("GET", urls.Url+workspaceName+"/onboard/api/v1/source/"+sourceId, nil)
	if err != nil {
		return apiOnboard.Source{}, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return apiOnboard.Source{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return apiOnboard.Source{}, err
	}
	err = res.Body.Close()
	if err != nil {
		return apiOnboard.Source{}, err
	}
	var response apiOnboard.Source
	err = json.Unmarshal(body, &response)
	if err != nil {
		return apiOnboard.Source{}, err
	}
	return response, nil
}
func OnboardHealthSource(workspaceName string, accessToken string, sourceId string) (apiOnboard.Source, error) {
	req, err := http.NewRequest("POST", urls.Url+workspaceName+"/onboard/api/v1/source/"+sourceId+"/healthcheck", nil)
	if err != nil {
		return apiOnboard.Source{}, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return apiOnboard.Source{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return apiOnboard.Source{}, err
	}
	err = res.Body.Close()
	if err != nil {
		return apiOnboard.Source{}, err
	}
	var response apiOnboard.Source
	err = json.Unmarshal(body, &response)
	if err != nil {
		return apiOnboard.Source{}, err
	}
	return response, nil
}
func OnboardGetListSourcesFilteredById(workspaceName string, accessToken string, sourceIDs []string) ([]apiOnboard.Source, error) {
	var request apiOnboard.GetSourcesRequest
	request.SourceIDs = sourceIDs
	reqEncoded, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", urls.Url+workspaceName+"/onboard/api/v1/sources", bytes.NewBuffer(reqEncoded))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var response []apiOnboard.Source
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}
func OnboardGetSourceCredential(workspaceName string, accessToken string, sourceId string) ([]byte, string, error) {
	req, err := http.NewRequest("POST", urls.Url+workspaceName+"/onboard/api/v1/source/"+sourceId+"/credentials", nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", err
	}
	var fields map[string]interface{}
	err = json.Unmarshal(body, &fields)
	if err != nil {
		return nil, "", err
	}
	if _, ok := fields["accessKey"]; ok {
		return body, "aws", nil
	} else {
		return body, "azure", nil
	}
}
func OnboardEnableSource(workspaceName string, accessToken string, sourceId string) error {
	req, err := http.NewRequest("POST", urls.Url+workspaceName+"/onboard/api/v1/source/"+sourceId+"/enable", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	statusCode := res.StatusCode
	err = res.Body.Close()
	if err != nil {
		return err
	}
	if statusCode == http.StatusOK {
		return nil
	} else {
		return fmt.Errorf("error with status: %v", statusCode)
	}
}
func OnboardDisableSource(workspaceName string, accessToken string, sourceId string) error {
	req, err := http.NewRequest("POST", urls.Url+workspaceName+"/onboard/api/v1/source/"+sourceId+"/disable", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	statusCode := res.StatusCode
	err = res.Body.Close()
	if err != nil {
		return err
	}
	if statusCode == http.StatusOK {
		return nil
	} else {
		return fmt.Errorf("error with status: %v", statusCode)
	}
}
func OnboardGetListOfSource(workspaceName string, accessToken string) ([]apiOnboard.Source, error) {
	req, err := http.NewRequest("GET", urls.Url+workspaceName+"/onboard/api/v1/sources", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	err = res.Body.Close()
	if err != nil {
		return nil, err
	}
	var response []apiOnboard.Source
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}
func OnboardGetListOfSourcesByFilters(workspaceName string, accessToken string, connectorType string, pageSize string, pageNumber string) error {
	//req, err := http.NewRequest("GET", urls.Url+workspaceName+"/onboard/api/v1/credential/sources/list", nil)
	//if err != nil {
	//	return err
	//}
	//req.Header.Add("Content-Type", "application/json")
	//req.Header.Set("Authorization", "Bearer "+accessToken)
	//res, err := http.DefaultClient.Do(req)
	//if err != nil {
	//	return err
	//}
	//io.r
	return nil
}
