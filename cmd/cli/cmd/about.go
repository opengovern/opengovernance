package cmd

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/kaytu-io/kaytu-engine/pkg/cli"
)

var outputAbout string

// aboutCmd represents the about command
var aboutCmd = &cobra.Command{
	Use: "about",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().ParseErrorsWhitelist.UnknownFlags {
			return errors.New("please enter right flag .")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := cli.GetConfig(cmd, false)
		if err != nil {
			return fmt.Errorf("[about]: %v", err)
		}

		bodyResponse, err := cli.RequestAbout(cnf.AccessToken)
		if err != nil {
			return fmt.Errorf("[about]: %v", err)
		}
		err = cli.PrintOutput(bodyResponse, outputAbout)
		if err != nil {
			return fmt.Errorf("[about]: %v", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(aboutCmd)
	aboutCmd.Flags().StringVar(&outputAbout, "output", "table", "specifying output type [json, table]")
}
