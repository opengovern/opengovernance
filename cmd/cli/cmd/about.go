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
		errFunc, bodyResponse := RequestAbout(dataAccessToken.AccessToken)
		if errFunc != nil {
			panic(errFunc)
		}
		response := ResponseAbout{}
		errJson := json.Unmarshal(bodyResponse, &response)
		if errJson != nil {
			panic(errJson)
		}
		typeOutput, err := cmd.Flags().GetString("output")
		if err != nil {
			panic(err)
		}
		if typeOutput == "json" {
			fmt.Println(string(bodyResponse))
		} else {
			tableAbout := table.NewWriter()
			tableAbout.SetOutputMirror(os.Stdout)
			tableAbout.AppendHeader(table.Row{"", "email", "email_verified", "sub"})
			tableAbout.AppendRows([]table.Row{
				{"", response.Email, response.EmailVerified, response.Sub},
			})
			tableAbout.AppendSeparator()
			tableAbout.Render()
		}
	},
}

func init() {
	rootCmd.AddCommand(aboutCmd)
	aboutCmd.PersistentFlags().String("output", "", "can use this flag for specify the output type .")

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
