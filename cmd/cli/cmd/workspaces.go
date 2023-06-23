package cmd

import (
	"errors"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/cli"
	"github.com/spf13/cobra"
)

// workspacesCmd represents the workspaces command
var workspacesCmd = &cobra.Command{
	Use: "workspaces",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().ParseErrorsWhitelist.UnknownFlags {
			return errors.New("please enter right flag .")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := cli.GetConfig(cmd, false)
		if err != nil {
			return fmt.Errorf("[workspaces] : %v", err)
		}

		response, err := cli.RequestWorkspaces(cnf.AccessToken)
		if err != nil {
			return fmt.Errorf("[workspaces] : %v", err)
		}

		err = cli.PrintOutputForTypeArray(response, outputTypeWorkspaces)
		if err != nil {
			return fmt.Errorf("[workspaces] : %v", err)
		}
		return nil
	},
}
var outputTypeWorkspaces string

func init() {
	rootCmd.AddCommand(workspacesCmd)
	workspacesCmd.Flags().StringVar(&outputTypeWorkspaces, "output", "table", "specifying output type [json, table]")
}
