/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	urls "gitlab.com/keibiengine/keibi-engine/pkg/cli/consts"
	"io"
	"net/http"
	"os"
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
		var dataAccessToken DataStoredInFile
		errJm := json.Unmarshal(accessToken, &dataAccessToken)
		if errJm != nil {
			panic(errJm)
		}
		err := RequestAbout(dataAccessToken.AccessToken)
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(aboutCmd)
}

func RequestAbout(accessToken string) error {
	req, errReq := http.NewRequest("GET", urls.UrlAbout, nil)
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
	fmt.Println(string(body))
	response := ResponseAbout{}
	errJson := json.Unmarshal(body, &response)
	if errJson != nil {
		fmt.Println("error belong to unmarshal response about : ")
		return errJson
	}

	tableAbout := table.NewWriter()
	tableAbout.SetOutputMirror(os.Stdout)
	tableAbout.AppendHeader(table.Row{"", "email", "email_verified", "sub"})
	tableAbout.AppendRows([]table.Row{
		{"", response.Email, response.EmailVerified, response.Sub},
	})
	tableAbout.AppendSeparator()
	tableAbout.Render()
	return nil
}
