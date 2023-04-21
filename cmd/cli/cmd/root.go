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
	Use:   "kctl",
	Short: "Kaytu cli",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(iam.Get)
	//rootCmd.AddCommand(role.GetRole)
	//rootCmd.AddCommand(users.GetUsers)
	rootCmd.AddCommand(iam.Delete)
	rootCmd.AddCommand(iam.Create)
	rootCmd.AddCommand(iam.Update)

}
