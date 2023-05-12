/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
	iam "gitlab.com/keibiengine/keibi-engine/cmd/cli/cmd/iam"
	onboard "gitlab.com/keibiengine/keibi-engine/cmd/cli/cmd/onboard"
	"os"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "keibi",
	Short: "Keibi is a program for keeping the companies data without interruption and keep the data the most optimal form .",
	RunE: func(cmd *cobra.Command, args []string) error {
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
	rootCmd.AddCommand(onboard.Get)
	rootCmd.AddCommand(onboard.Create)
	rootCmd.AddCommand(onboard.Count)
	rootCmd.AddCommand(onboard.Update)
	rootCmd.AddCommand(onboard.Delete)

}
