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
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().ParseErrorsWhitelist.UnknownFlags {
			fmt.Println("please enter right flag .")
			return cmd.Help()
		}
		if len(args) == 0 {
			return cmd.Help()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := cli.GetConfig()
		if err != nil {
			return fmt.Errorf("[workspaces] : %v", err)
		}
		checkEXP, err := cli.CheckExpirationTime(accessToken)
		if err != nil {
			return err
		}
		if checkEXP == true {
			fmt.Println("your access token was expire please login again ")
			return nil
		}

		response, err := cli.RequestWorkspaces(accessToken)
		if err != nil {
			return fmt.Errorf("[workspaces] : %v", err)
		}

		err = cli.PrintOutputForTypeArray(response, OutputType)
		if err != nil {
			return fmt.Errorf("[workspaces] : %v", err)
		}
		return nil
	},
}
var OutputType string

func init() {
	rootCmd.AddCommand(workspacesCmd)
	workspacesCmd.Flags().StringVar(&OutputType, "output", "", "specifying output type [json, table]")
}
