/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

// logoutCmd represents the logout command
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "logout ",
	Long:  `this codd do logout from account that was login `,
	Run: func(cmd *cobra.Command, args []string) {
		deleteFile()
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
func deleteFile() {
	home := os.Getenv("HOME")
	errRemove := os.Remove(home + "/.kaytu/auth/accessToken.txt")
	if errRemove != nil {
		errorsRemove := fmt.Sprintf("err belong to remove file : %v ", errRemove)
		panic(errorsRemove)
	}
}
