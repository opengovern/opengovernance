package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	urls "gitlab.com/keibiengine/keibi-engine/pkg/cli/consts"
	"gitlab.com/keibiengine/keibi-engine/pkg/workspace/api"

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
		return "", fmt.Errorf("[getConfig] : %v", err)
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
	err := os.RemoveAll(home + "/.kaytu/config.json")
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
func RequestAbout(accessToken string) ([]byte, error) {
	req, err := http.NewRequest("GET", urls.UrlAbout, nil)
	if err != nil {
		return nil, fmt.Errorf("[requestAbout] : %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[requestAbout] : %v", err)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("[requestAbout] : %v", err)
	}
	return body, nil
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
	defer res.Body.Close()
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

func RequestWorkspaces(accessToken string) (interface{}, error) {
	req, err := http.NewRequest("GET", urls.UrlWorkspace, nil)
	if err != nil {
		return nil, fmt.Errorf("[RequestWorkspaces] : %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[RequestWorkspaces] : %v", err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("[RequestWorkspaces] : %v", err)
	}

	defer res.Body.Close()

	var responseUnmarshal []api.WorkspaceResponse
	err = json.Unmarshal(body, &responseUnmarshal)
	if err != nil {
		return nil, fmt.Errorf("[RequestWorkspaces] : %v", err)
	}

	return responseUnmarshal, nil
}
