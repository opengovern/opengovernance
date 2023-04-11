/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/cli"
	"gitlab.com/keibiengine/keibi-engine/pkg/workspace/api"
	"os"
)

// workspacesCmd represents the workspaces command
var workspacesCmd = &cobra.Command{
	Use:   "workspaces",
	Short: "A brief description of your command",
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, errAC := cli.GetConfig()
		if errAC != nil {
			return errAC
		}

		var dataAccessToken cli.DataStoredInFile
		errJm := json.Unmarshal(accessToken, &dataAccessToken)
		if errJm != nil {
			return errJm
		}

		err, bodyResponse := cli.RequestWorkspaces(dataAccessToken.AccessToken)
		if err != nil {
			panic(err)
		}

		var responseUnmarshal []api.WorkspaceResponse
		errJson := json.Unmarshal(bodyResponse, &responseUnmarshal)
		if errJson != nil {
			fmt.Println("error relate to jsonUnmarshal in workspace: ")
			return errJson
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
		return nil
	},
}

func init() {
	rootCmd.AddCommand(workspacesCmd)
	workspacesCmd.PersistentFlags().String("output", "", "this flag use for specify the output type .")
}
