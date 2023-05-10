package onboard

import "github.com/spf13/cobra"

var Delete = &cobra.Command{
	Use:   "delete",
	Short: "it is use for deleting user or key ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	Delete.AddCommand()
}
