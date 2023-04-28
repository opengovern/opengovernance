package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

func init() {

	IamUpdate.AddCommand(UpdateKeyRole)
	IamUpdate.AddCommand(UserUpdate)
	IamUpdate.AddCommand(StateWorkspaceKey)

}

var IamUpdate = &cobra.Command{
	Use:   "iam",
	Short: "iam command ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			err := cmd.Help()
			if err != nil {
				return err
			}
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("iam Get")
	},
}
