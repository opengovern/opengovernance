/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
	iam "github.com/kaytu-io/kaytu-engine/cmd/cli/cmd/iam"
	"os"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ktucli",
	Short: "Kaytu cli",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {

	rootCmd.AddCommand(iam.Get)
	rootCmd.AddCommand(iam.Delete)
	rootCmd.AddCommand(iam.Create)
	rootCmd.AddCommand(iam.Update)
	rootCmd.AddCommand(iam.Count)

}
