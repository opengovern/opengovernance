package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	apis "gitlab.com/keibiengine/keibi-engine/pkg/cli"
)

var Delete = &cobra.Command{
	Use:   "delete",
	Short: "for delete something",
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
		fmt.Println("delete")
	},
}

var DeleteUser = &cobra.Command{
	Use:   "user",
	Short: "delete user",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			err := cmd.Help()
			if err != nil {
				return err
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := apis.GetConfig()
		if err != nil {
			return err
		}
		if cmd.Flags().Lookup("workspaceName").Changed {
		} else {
			fmt.Println("please enter the workspaceName flag .")
			return nil
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
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			err := cmd.Help()
			if err != nil {
				return err
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := apis.GetConfig()
		if err != nil {
			return err
		}
		if cmd.Flags().Lookup("workspaceName").Changed {
		} else {
			fmt.Println("please enter the workspaceName flag .")
			return nil
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

	DeleteUser.Flags().StringVar(&workspacesNameForDelete, "workspaceName", "", "specifying the workspaceName user ")
	DeleteUser.Flags().StringVar(&UserIdForDelete, "userId", "", "specifying the userId ")

	DeleteKey.Flags().StringVar(&workspacesNameForDelete, "workspaceName", "", "specifying the workspaceName user")
	DeleteKey.Flags().StringVar(&KeyIdForDelete, "keyId", "", "specifying the keyID ")

}
