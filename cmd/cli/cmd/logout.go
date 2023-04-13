package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/cli"
)

// logoutCmd represents the logout command
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logging out from kaytu",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := cli.RemoveConfig()
		if err != nil {
			return fmt.Errorf("[logout] : %v", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
