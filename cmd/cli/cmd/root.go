/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
	iam "gitlab.com/keibiengine/keibi-engine/cmd/cli/cmd/iam"
	"os"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "keibi",
	Short: "Keibi is a program for keeping the company's data without interruption and keeping the data in the most optimal form.",
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
	rootCmd.AddCommand(iam.Count)

}
