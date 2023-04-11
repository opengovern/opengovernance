package cli

import (
	"bytes"
	"fmt"
	"github.com/fatih/color"
	urls "gitlab.com/keibiengine/keibi-engine/pkg/cli/consts"
	"io"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
	"os"
	"time"
)

func GetConfig() ([]byte, error) {
	home := os.Getenv("HOME")
	AC, errRead := os.ReadFile(home + "/.kaytu/config.json")
	if errRead != nil {
		fmt.Println("error relate to reading file accessToken: ")
		return nil, errRead
	}
	return AC, nil
}

func RequestAbout(accessToken string) (error, []byte) {
	req, errReq := http.NewRequest("GET", urls.UrlAbout, nil)
	if errReq != nil {
		fmt.Println("error relate to request for about :")
		return errReq, nil
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, errRes := http.DefaultClient.Do(req)
	if errRes != nil {
		fmt.Println("error relate to response in about : ")
		return errRes, nil
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("error relate to reading response in about : ")
		return err, nil
	}
	return nil, body
}

func RequestDeviceCode() (string, error) {

	payload := DeviceCodeRequest{
		ClientId: ClientID,
		Scope:    "openid profil email api:read",
		Audience: "https://app.keibi.io",
	}
	payloadEncode, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("error belong to jsonMarshal in deviceCode request : ")
		return "", err
	}
	req, err := http.NewRequest("POST", urls.UrlLogin+"/oauth/device/code", bytes.NewBuffer(payloadEncode))
	if err != nil {
		fmt.Println("error belong to handle first request : ")
		return "", err
	}
	req.Header.Add("content-type", "application/JSON")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("error into first requesting  : ")
		return "", err
	}
	body, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		fmt.Println("error into reading first request :")
		return "", err
	}
	response := DeviceCodeResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println("error belong to jsonMarshal : ")
		return "", err
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

	var res *http.Response
	for {
		requestEncoded, errJM := json.Marshal(payload)
		if errJM != nil {

			fmt.Printf("error into jsonMarshal in request accessToken : %v", requestEncoded)
			time.Sleep(5)
			continue
		}
		req, err := http.NewRequest("POST", urls.UrlLogin+"/oauth/token", bytes.NewBuffer(requestEncoded))
		if err != nil {
			fmt.Printf("error into information request : %v ", err)
			time.Sleep(5)
			continue
		}
		req.Header.Add("content-type", "application/JSON")
		res, err = http.DefaultClient.Do(req)
		if err != nil {
			time.Sleep(5)
			continue
		}
		response := ResponseAccessToken{}
		body, errRead := ioutil.ReadAll(res.Body)
		if errRead != nil {
			fmt.Printf("error relate to reading body response : %v ", errRead)
			time.Sleep(5)
			continue
		}
		err = res.Body.Close()
		if err != nil {
			fmt.Printf("error belong to close body response : %v ", err)
			time.Sleep(5)
			continue
		}
		errJson := json.Unmarshal(body, &response)
		if errJson != nil {
			fmt.Printf("error belong to jsonUnmarshal accessToken : %v", errJson)
			time.Sleep(5)
			continue
		}
		if response.AccessToken != "" {
			color.Red("\naccessToken equal to :")
			fmt.Println(response.AccessToken)
			responseAccessToken := response.AccessToken
			return responseAccessToken, nil
		} else {
			time.Sleep(5)
			continue
		}
	}
}

func RemoveConfigFile() error {
	home := os.Getenv("HOME")
	errRemove := os.Remove(home + "/.kaytu/config.json")
	if errRemove != nil {
		return errRemove
	}
	fmt.Println("successfully logout from your account. ")
	return nil
}

func RequestWorkspaces(accessToken string) (error, []byte) {
	req, err := http.NewRequest("GET", urls.UrlWorkspace, nil)
	if err != nil {
		fmt.Println("error related to request in workspaces: ")
		return err, nil
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, errRead := http.DefaultClient.Do(req)
	if errRead != nil {
		fmt.Println("error relate to response in workspaces:")
		return errRead, nil
	}
	body, errBody := ioutil.ReadAll(res.Body)
	if errBody != nil {
		fmt.Println("error relate to reading response in workspaces:")
		return errBody, nil
	}
	defer res.Body.Close()

	return nil, body
}
