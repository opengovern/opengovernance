package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	IamCreate.AddCommand(CreateKeyCmd)
	IamCreate.AddCommand(CreateUser)

}

var IamCreate = &cobra.Command{
	Use:   "iam",
	Short: "iam command ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return nil
	},
}
