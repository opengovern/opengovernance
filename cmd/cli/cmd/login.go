package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Logging in into kaytu",
	Long:  `Logging into kaytu using device authentication mechanism`,
	Run: func(cmd *cobra.Command, args []string) {
		deviceCode, err := RequestDeviceCode()
		if err != nil {
			panic(err)
		}
		accessToken, errAccessToken := accessToken(deviceCode)
		if errAccessToken != nil {
			panic(errAccessToken)
		}
		//save accessToken to the file :
		home := os.Getenv("HOME")
		if _, errStat := os.Stat(home + "/.kaytu/auth/accessToken.txt"); errStat != nil {
			file, errFil := os.Create(home + "/.kaytu/auth/accessToken.txt")
			if errFil != nil {
				errorsFile := fmt.Sprintf("error belong to created file :%v", errFil)
				panic(errorsFile)
			}
			_, errWrite := file.WriteString(accessToken)
			if errWrite != nil {
				fmt.Println("error belong to writing accessToken into file : ")
				panic(errWrite)
			}
		} else {
			errRemove := os.Remove(home + "/.kaytu/auth/accessToken.txt")
			if errRemove != nil {
				fmt.Println("error relate to removing file accessToken: ")
				panic(errRemove)
			}
			file, errFil := os.Create(home + "/.kaytu/auth/accessToken.txt")
			if errFil != nil {
				errorsFile := fmt.Sprintf("error belong to created file :%v", errFil)
				panic(errorsFile)
			}
			_, errWrite := file.WriteString(accessToken)
			if errWrite != nil {
				fmt.Println("error belong to writing accessToken into file : ")
				panic(errWrite)
			}
		}
	},
}

const domain string = "https://dev-ywhyatwt.us.auth0.com"
const clientID string = "6P7NtO3D9bQaw9DbdJ2pICBY82nLGmBg"

func init() {
	rootCmd.AddCommand(loginCmd)
}

type DeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationUrl         string `json:"verification_uri"`
	VerificationUrlComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type DeviceCodeRequest struct {
	ClientId string `json:"client_id"`
	Scope    string `json:"scope"`
	Audience string `json:"audience"`
}

func RequestDeviceCode() (string, error) {

	payload := DeviceCodeRequest{
		ClientId: clientID,
		Scope:    "openid profil email api:read",
		Audience: "https://app.keibi.io",
	}
	payloadEncode, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("error belong to jsonMarshal in deviceCode request : ")
		return "", err
	}
	req, err := http.NewRequest("POST", domain+"/oauth/device/code", bytes.NewBuffer(payloadEncode))
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

type responseAccessToken struct {
	AccessToken string `json:"access_token"`
	scope       string `json:"scope"`
	IdToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
	ExpireIn    string `json:"expire_in"`
}
type requestAccessToken struct {
	GrantType  string `json:"grant_type"`
	DeviceCode string `json:"device_code"`
	ClientId   string `json:"client_id"`
}

func accessToken(deviceCode string) (string, error) {
	payload := requestAccessToken{
		GrantType:  "urn:ietf:params:oauth:grant-type:device_code",
		DeviceCode: deviceCode,
		ClientId:   clientID,
	}

	var res *http.Response
	for {
		requestEncoded, errJM := json.Marshal(payload)
		if errJM != nil {
			fmt.Printf("error into jsonMarshal in request accessToken : %v", requestEncoded)
			time.Sleep(5)
			continue
		}
		req, err := http.NewRequest("POST", domain+"/oauth/token", bytes.NewBuffer(requestEncoded))
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
		response := responseAccessToken{}
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
			return response.AccessToken, nil
		} else {
			time.Sleep(5)
			continue
		}
	}
}
