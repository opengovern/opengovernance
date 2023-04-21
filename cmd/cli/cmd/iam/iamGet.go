package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

func init() {

	IamGet.AddCommand(roles)
	IamGet.AddCommand(roleDetails)
	IamGet.AddCommand(KeysCmd)
	IamGet.AddCommand(KeyDetailsCmd)
	IamGet.AddCommand(userDetails)
	IamGet.AddCommand(Users)

}

var IamGet = &cobra.Command{
	Use:   "iam",
	Short: "iam command ",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("iam Get")
	},
}
