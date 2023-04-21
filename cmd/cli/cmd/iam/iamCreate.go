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
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("iam Get")
	},
}
