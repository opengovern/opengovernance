package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/cli"
)

// workspacesCmd represents the workspaces command
var workspacesCmd = &cobra.Command{
	Use:   "workspaces",
	Short: "A brief description of your command",
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := cli.GetConfig()
		if err != nil {
			return fmt.Errorf("[workspaces] : %v", err)
		}

		response, err := cli.RequestWorkspaces(accessToken)
		if err != nil {
			return fmt.Errorf("[workspaces] : %v", err)
		}

		err = cli.PrintOutputForWorkspaces(response, OutputType)
		if err != nil {
			return fmt.Errorf("[workspaces] : %v", err)
		}
		return nil
	},
}
var OutputType string
var workspacesId int64

func init() {
	rootCmd.AddCommand(workspacesCmd)
	workspacesCmd.Flags().Int64Var(&workspacesId, "workspace", 0, "specifying the workspacesId")
	workspacesCmd.Flags().StringVar(&OutputType, "output", "", "specifying output type [json, table]")
}
