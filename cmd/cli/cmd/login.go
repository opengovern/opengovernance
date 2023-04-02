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
	Short: "part login ",
	Long: `in this part we sent two request the first is for confirming device and
the second request is for give access token `,
	Run: func(cmd *cobra.Command, args []string) {
		RequestDeviceCode()
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

type responseFirstRequest struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationUrl         string `json:"verification_uri"`
	VerificationUrlComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

func RequestDeviceCode() {
	url := "https://dev-ywhyatwt.us.auth0.com/oauth/device/code"
	var payload = []byte(`{"client_id": "6P7NtO3D9bQaw9DbdJ2pICBY82nLGmBg", "scope": "openid profil email api:read", "audience": "https://dev-ywhyatwt.us.auth0.com/userinfo"}`)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		errors := fmt.Sprintf("error into handeling first request : %v", err)
		panic(errors)
	}
	req.Header.Add("content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		errors := fmt.Sprintf("error into first requesting  : %v", err)
		panic(errors)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		errors := fmt.Sprintf("error into reading first request : %v", err)
		panic(errors)
	}
	response := responseFirstRequest{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println("test")
		panic(err)
	}
	fmt.Printf("to active this program :\n1. on your computer or mobile device , go to: %v \n", response.VerificationUrlComplete)
	fmt.Printf("2. enter the following code: %v", response.UserCode)
	accessToken(response.DeviceCode, "6P7NtO3D9bQaw9DbdJ2pICBY82nLGmBg")
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

//urn:ietf:params:oauth:grant-type:device_code
func accessToken(deviceCode string, clientId string) {
	url := "https://dev-ywhyatwt.us.auth0.com/oauth/token"
	payload := requestAccessToken{
		GrantType:  "urn:ietf:params:oauth:grant-type:device_code",
		DeviceCode: deviceCode,
		ClientId:   clientId,
	}
	var res *http.Response
	for {
		requestEncode, errJM := json.Marshal(payload)
		if errJM != nil {
			fmt.Sprintf("error is inside json marshal : %v", errJM)
		}
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestEncode))
		if err != nil {
			errors := fmt.Sprintf("error into information request : %v", err)
			panic(errors)
		}
		req.Header.Add("content-type", "application/json")
		res, err = http.DefaultClient.Do(req)
		if err != nil {
			time.Sleep(5)
			continue
		}
		response := responseAccessToken{}
		body, _ := ioutil.ReadAll(res.Body)
		res.Body.Close()
		errJson := json.Unmarshal(body, &response)
		if errJson != nil {
			panic(errJson)
		}
		if response.AccessToken != "" {
			color.Red("\naccessToken equal to :")
			//save accessToken to a file :
			fmt.Println(response.AccessToken)
			home := os.Getenv("HOME")
			file, errFil := os.Create(home + "/.kaytu/auth/accessToken.txt")
			if errFil != nil {
				errorsFile := fmt.Sprintf("error belong to created file :%v", errFil)
				panic(errorsFile)
			}
			file.WriteString(response.AccessToken)
			break
		} else {
			time.Sleep(5)
			continue
		}
	}
}
