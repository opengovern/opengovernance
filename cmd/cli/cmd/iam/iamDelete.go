package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	IamDelete.AddCommand(DeleteUserInvite)
	IamDelete.AddCommand(DeleteUserAccess)
	IamDelete.AddCommand(DeleteKey)
}

var IamDelete = &cobra.Command{
	Use:   "iam",
	Short: "iam command ",
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
