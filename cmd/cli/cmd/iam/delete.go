package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	apis "gitlab.com/keibiengine/keibi-engine/pkg/cli"
)

var Delete = &cobra.Command{
	Use:   "delete",
	Short: "for delete something",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("delete")
	},
}

var DeleteUser = &cobra.Command{
	Use:   "user",
	Short: "delete user",
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := apis.GetConfig()
		if err != nil {
			return err
		}
		response, err := apis.DeleteIamUser(workspacesNameForDelete, accessToken, UserIdForDelete)
		if err != nil {
			return err
		}
		fmt.Println(response)
		return nil
	},
}

var DeleteKey = &cobra.Command{
	Use:   "user",
	Short: "delete key",
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := apis.GetConfig()
		if err != nil {
			return err
		}
		response, err := apis.DeleteKey(workspacesNameForDelete, accessToken, KeyIdForDelete)
		if err != nil {
			return err
		}
		fmt.Println(response)
		return nil
	},
}

var workspacesNameForDelete string
var UserIdForDelete string
var KeyIdForDelete string

func init() {
	Delete.AddCommand(IamDelete)

	DeleteUser.Flags().StringVar(&workspacesNameForDelete, "workspacesName", "", "specifying the workspaceName user ")
	DeleteUser.Flags().StringVar(&UserIdForDelete, "userId", "", "specifying the userId ")

	DeleteKey.Flags().StringVar(&workspacesNameForDelete, "workspacesName", "", "specifying the workspaceName user")
	DeleteKey.Flags().StringVar(&KeyIdForDelete, "keyId", "", "specifying the keyID ")

}
