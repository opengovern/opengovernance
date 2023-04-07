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
		err := RequestWorkspaces()
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(workspacesCmd)
}

func RequestWorkspaces() error {
	req, err := http.NewRequest("GET", urls.UrlWorkspace, nil)
	if err != nil {
		fmt.Println("error related to request in workspaces: ")
		return err
	}
	home := os.Getenv("HOME")
	AT, errFile := os.ReadFile(home + "/.kaytu/auth/accessToken.txt")
	if errFile != nil {
		fmt.Println("error relate to reading	 accessToken file in workspaces: ")
		return errFile
	}
	var dataAccessToken DataStoredInFile
	errJm := json.Unmarshal(AT, &dataAccessToken)
	if err != nil {
		return errJm
	}
	req.Header.Set("Authorization", "Bearer "+dataAccessToken.AccessToken)
	res, errRead := http.DefaultClient.Do(req)
	if errRead != nil {
		fmt.Println("error relate to response in workspaces:")
		return errRead
	}
	body, errBody := ioutil.ReadAll(res.Body)
	if errBody != nil {
		fmt.Println("error relate to reading response in workspaces:")
		return errBody
	}
	defer res.Body.Close()
	var response []api.WorkspaceResponse
	errJson := json.Unmarshal(body, &response)
	if errJson != nil {
		fmt.Println("error relate to jsonUnmarshal in workspace: ")
		return errJson
	}
	for _, value := range response {
		tableWorkspaces := table.NewWriter()
		tableWorkspaces.SetOutputMirror(os.Stdout)
		tableWorkspaces.AppendHeader(table.Row{"", "Workspaces Name", "ID", "Workspaces State", "Workspaces creation time", "workspaces Version"})
		tableWorkspaces.AppendRows([]table.Row{
			{"", value.Name, value.ID, value.Status, value.CreatedAt, value.Version},
		})
		tableWorkspaces.AppendSeparator()
		tableWorkspaces.Render()
	}
	return nil
}
