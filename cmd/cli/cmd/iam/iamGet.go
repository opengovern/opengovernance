package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	IamGet.AddCommand(roles)
	IamGet.AddCommand(roleDetails)
	IamGet.AddCommand(listRoleUsers)
	IamGet.AddCommand(listRoleKeys)
	IamGet.AddCommand(KeysCmd)
	IamGet.AddCommand(KeyDetailsCmd)
	IamGet.AddCommand(userDetails)
	IamGet.AddCommand(Users)

}

var IamGet = &cobra.Command{
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
