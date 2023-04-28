package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	urls "gitlab.com/keibiengine/keibi-engine/pkg/cli/consts"
	workspace "gitlab.com/keibiengine/keibi-engine/pkg/workspace/api"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

func GetConfig() (string, error) {
	home := os.Getenv("HOME")
	accessTokenByte, err := os.ReadFile(home + "/.kaytu/config.json")
	if err != nil {
		return "", fmt.Errorf("[getConfig] : please firs login")
	}

	var config Config
	err = json.Unmarshal(accessTokenByte, &config)
	if err != nil {
		return "", fmt.Errorf("[getConfig] : %v", err)
	}
	return config.AccessToken, nil
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
func CheckExpirationTime(accessToken string) (bool, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(accessToken, jwt.MapClaims{})
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

func IamGetUsers(workspaceName string, accessToken string, email string, emailVerified bool, role string) ([]api.GetUserResponse, error) {
	roleTypeRole := api.Role(role)
	bodyRequest := RequestGetIamUsers{
		email,
		emailVerified,
		roleTypeRole,
	}
	bodyEncoded, err := json.Marshal(bodyRequest)
	if err != nil {
		return []api.GetUserResponse{{}}, err
	}
	req, err := http.NewRequest("GET", urls.Url+workspaceName+"/auth/api/v1/users", bytes.NewBuffer(bodyEncoded))
	if err != nil {
		return []api.GetUserResponse{{}}, err
	}
	req.Header.Add("content-type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return []api.GetUserResponse{{}}, err
	}
	bodyResponse, err := io.ReadAll(res.Body)
	if err != nil {
		return []api.GetUserResponse{{}}, err
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

func GetIamUserDetails(accessToken string, workspaceName string, userId string) (ResponseUserDetails, error) {
	req, err := http.NewRequest("GET", urls.Url+workspaceName+"/auth/api/v1/user/:"+userId, nil)
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

func DeleteIamUser(workspacesName string, accessToken string, userId string) (string, error) {
	req, err := http.NewRequest("DELETE", urls.Url+workspacesName+"/auth/api/v1/user/invite", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Add("content-type", "application/json")
	query := req.URL.Query()
	query.Set("userId", userId)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	if res.StatusCode == http.StatusOK {
		return "user deleted ", nil
	} else {
		return "deleting user was fail", nil
	}
}

func ListRoles(WorkspacesName string, accessToken string) ([]api.RolesListResponse, error) {
	req, err := http.NewRequest("GET", urls.UrlListRoles+WorkspacesName+"/auth/api/v1/roles", nil)
	if err != nil {
		return []api.RolesListResponse{{}}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Add("content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return []api.RolesListResponse{{}}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return []api.RolesListResponse{{}}, err
	}
	err = res.Body.Close()
	if err != nil {
		return []api.RolesListResponse{{}}, err
	}
	var response []api.RolesListResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return []api.RolesListResponse{{}}, err
	}
	return response, nil
}

func RoleDetail(workspaceName string, roleName string, accessToken string) (api.RoleDetailsResponse, error) {
	req, err := http.NewRequest("GET", urls.Url+workspaceName+"/auth/api/v1/roles/:"+roleName, nil)
	if err != nil {
		return api.RoleDetailsResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Add("content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return api.RoleDetailsResponse{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return api.RoleDetailsResponse{}, err
	}
	err = res.Body.Close()
	if err != nil {
		return api.RoleDetailsResponse{}, err
	}
	var response api.RoleDetailsResponse
	fmt.Println(string(body))
	err = json.Unmarshal(body, &response)
	if err != nil {
		return api.RoleDetailsResponse{}, err
	}
	return response, nil
}

func CreateUser(workspaceName string, accessToken string, email string, role string) (string, error) {
	request := RequestCreateUser{
		Email: email,
		Role:  role,
	}
	reqBody, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("GET", urls.Url+workspaceName+"/auth/api/v1/user/invite", bytes.NewBuffer(reqBody))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Add("content-type", "application/json")
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

func UpdateUser(workspaceName string, accessToken string, role string, userID string) (string, error) {
	request := api.PutRoleBindingRequest{Role: api.Role(role), UserID: userID}
	reqBody, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("PUT", urls.Url+workspaceName+"/auth/api/v1/user/role/binding", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Add("content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	if res.StatusCode == http.StatusOK {
		return "user updated successfully ", nil
	} else {
		return "updating user was fail", nil
	}
}

func GetListKeys(workspacesName string, accessToken string) ([]api.WorkspaceApiKey, error) {
	req, err := http.NewRequest("GET", urls.Url+workspacesName+"/auth/api/v1/keys", nil)
	if err != nil {
		return []api.WorkspaceApiKey{{}}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Add("content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return []api.WorkspaceApiKey{{}}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return []api.WorkspaceApiKey{{}}, err
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
func GetKeyDetails(workspacesName string, accessToken string, id string) (api.WorkspaceApiKey, error) {
	req, err := http.NewRequest("GET", urls.Url+workspacesName+"/auth/api/v1/key/:"+id, nil)
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Add("content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	body, err := io.ReadAll(res.Body)
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

func CreateKeys(workspacesName string, accessToken string, keyName string, role string) (api.CreateAPIKeyResponse, error) {
	var request RequestCreateAPIKey
	request.RoleName = role
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
	req.Header.Add("content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return api.CreateAPIKeyResponse{}, err
	}
	body, err := io.ReadAll(res.Body)
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
func UpdateKeyRole(workspacesName string, accessToken string, id uint, role string) (api.WorkspaceApiKey, error) {
	request := api.UpdateKeyRoleRequest{ID: id, Role: api.Role(role)}
	reqBody, err := json.Marshal(request)
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	req, err := http.NewRequest("POST", urls.Url+workspacesName+"/auth/api/v1/key/role", bytes.NewBuffer(reqBody))
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Add("content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	body, err := io.ReadAll(res.Body)
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

func SuspendKey(workspaceName string, accessToken string, id string) (api.WorkspaceApiKey, error) {
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

func ActiveKey(workspaceName string, accessToken string, id string) (api.WorkspaceApiKey, error) {
	req, err := http.NewRequest("POST", urls.Url+workspaceName+"/auth/api/v1/key/"+id+"/activate", nil)
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	req.Header.Add("Authorization", "bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return api.WorkspaceApiKey{}, err
	}
	body, err := io.ReadAll(res.Body)
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
func DeleteKey(workspacesName string, accessToken string, id string) (string, error) {
	req, err := http.NewRequest("DELETE", urls.Url+workspacesName+"/auth/api/v1/key/:"+id+"/delete", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Add("content-type", "application/json")
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

//	func UpdateKeyState(workspacesName string, accessToken string, id string) error {
//		req , err := http.NewRequest("POST", urls.Url+workspacesName+"", nil)
//		if err!=nil{
//			return err
//		}
//		req .Header.Set("Authorization","Bearer "+ accessToken)
//		res , err := http.DefaultClient.Do(req)
//		if err != nil{
//			return err
//		}
//		body , err :=io.ReadAll(res.Body)
//		if err!= nil{
//			return err
//		}
//		err = res.Body.Close()
//		if err!=nil{
//			return err
//		}
//		fmt.Println(string(body))
//		return nil
//	}
