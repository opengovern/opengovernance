package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

func init() {
	IamCreate.AddCommand(CreateKeyCmd)
	IamCreate.AddCommand(UserCreate)

}

var IamCreate = &cobra.Command{
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
