package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

func init() {
	IamDelete.AddCommand(DeleteUser)

}

var IamDelete = &cobra.Command{
	Use:   "iam",
	Short: "iam command ",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("iam Get")
	},
}
