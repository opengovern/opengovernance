/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

// aboutCmd represents the about command
var aboutCmd = &cobra.Command{
	Use:   "about",
	Short: "About user",
	Run: func(cmd *cobra.Command, args []string) {
		home := os.Getenv("HOME")
		accessToken, errRead := os.ReadFile(home + "/.kaytu/auth/accessToken.txt")
		if errRead != nil {
			fmt.Println("error relate to reading file accessToken: ")
			panic(errRead)
		}
		err := RequestAbout(string(accessToken))
		if err != nil {
			panic(err)
		}
	},
}

const urlAbout string = "https://dev-ywhyatwt.us.auth0.com/userinfo"

func init() {
	rootCmd.AddCommand(aboutCmd)
}

type resposeAbout struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	emailVerified bool   `json:"email_verified"`
}

func RequestAbout(accessToken string) error {
	req, errReq := http.NewRequest("GET", urlAbout, nil)
	if errReq != nil {
		fmt.Println("error relate to request for about :")
		return errReq
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, errRes := http.DefaultClient.Do(req)
	if errRes != nil {
		fmt.Println("error relate to response in about : ")
		return errRes
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("error relate to reading response in about : ")
		return err
	}
	response := resposeAbout{}
	errJson := json.Unmarshal(body, &response)
	if errJson != nil {
		fmt.Println("error belong to unmarshal response about : ")
		return errJson
	}

	fmt.Printf("sub : %v\n", response.Sub)
	fmt.Printf("email: %v \n", response.Email)
	fmt.Printf("email verified: %v \n", response.emailVerified)

	return nil
}
