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
	"gitlab.com/keibiengine/keibi-engine/pkg/workspace/api"
	"io/ioutil"
	"net/http"
	"os"
)

// workspacesCmd represents the workspaces command
var workspacesCmd = &cobra.Command{
	Use:   "workspaces",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		home := os.Getenv("HOME")
		AT, errFile := os.ReadFile(home + "/.kaytu/auth/accessToken.txt")
		if errFile != nil {
			fmt.Println("error relate to reading	 accessToken file in workspaces: ")
			panic(errFile)
		}
		var dataAccessToken DataStoredInFile
		errJm := json.Unmarshal(AT, &dataAccessToken)
		if errJm != nil {
			panic(errJm)
		}
		err, bodyResponse := RequestWorkspaces(dataAccessToken.AccessToken)
		if err != nil {
			panic(err)
		}
		var responseUnmarshal []api.WorkspaceResponse
		errJson := json.Unmarshal(bodyResponse, &responseUnmarshal)
		if errJson != nil {
			fmt.Println("error relate to jsonUnmarshal in workspace: ")
			panic(errJson)
		}
		typeOutput, errFlag := cmd.Flags().GetString("output")
		if errFlag != nil {
			panic(errFlag)
		}
		if typeOutput == "json" {
			fmt.Println(string(bodyResponse))
		} else {
			for _, value := range responseUnmarshal {
				tableWorkspaces := table.NewWriter()
				tableWorkspaces.SetOutputMirror(os.Stdout)
				tableWorkspaces.AppendHeader(table.Row{"", "Workspaces Name", "ID", "Workspaces State", "Workspaces creation time", "workspaces Version"})
				tableWorkspaces.AppendRows([]table.Row{
					{"", value.Name, value.ID, value.Status, value.CreatedAt, value.Version},
				})
				tableWorkspaces.AppendSeparator()
				tableWorkspaces.Render()
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(workspacesCmd)
	workspacesCmd.PersistentFlags().String("output", "", "this flag use for specify the output type .")
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
