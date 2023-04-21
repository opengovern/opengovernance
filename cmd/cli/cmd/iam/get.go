package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	apis "gitlab.com/keibiengine/keibi-engine/pkg/cli"
)

var Get = &cobra.Command{
	Use:   "get",
	Short: "get command",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("get command")
	},
}

var userDetails = &cobra.Command{
	Use:   "user-details",
	Short: "print the Specifications of a user  ",
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := apis.GetConfig()
		if err != nil {
			return err
		}
		userDetails, err := apis.GetIamUserDetails(accessToken, workspacesName, userID)
		if err != nil {
			return err
		}
		fmt.Println(userDetails)
		return nil
	},
}

var Users = &cobra.Command{
	Use:   "users",
	Short: "print the users",
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := apis.GetConfig()
		if err != nil {
			return err
		}
		users, err := apis.IamGetUsers(workspacesName, accessToken, emailForUser, emailVerified, roleUser)
		if err != nil {
			return err
		}
		fmt.Println(users)
		return nil
	},
}
var roleDetails = &cobra.Command{
	Use:   "role-details",
	Short: "Print the specification of a role ",
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := apis.GetConfig()
		if err != nil {
			return err
		}
		response, err := apis.RoleDetail(workspacesName, role, accessToken)
		if err != nil {
			return err
		}
		fmt.Println(response)
		return nil
	},
}

var roles = &cobra.Command{
	Use:   "roles",
	Short: "Print the roles ",
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := apis.GetConfig()
		if err != nil {
			return err
		}
		response, err := apis.ListRoles(workspacesName, accessToken)
		if err != nil {
			return err
		}
		fmt.Println(response)
		return nil
	},
}
var KeysCmd = &cobra.Command{
	Use:   "keys",
	Short: "print the keys ",
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := apis.GetConfig()
		if err != nil {
			return err
		}
		response, err := apis.GetListKeys(workspacesName, accessToken)
		if err != nil {
			return err
		}
		fmt.Println(response)
		return nil
	},
}
var KeyDetailsCmd = &cobra.Command{
	Use:   "key-details",
	Short: "Print the specification of a key ",
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := apis.GetConfig()
		if err != nil {
			return err
		}
		response, err := apis.GetKeyDetails(workspacesName, accessToken, KeyID)
		if err != nil {
			return err
		}
		fmt.Println(response)
		return nil
	},
}

var KeyID string
var role string
var workspacesName string
var userID string
var emailVerified bool
var roleUser string
var emailForUser string

func init() {
	Get.AddCommand(IamGet)

	Users.Flags().StringVar(&workspacesName, "workspacesName", "", "specifying the workspace name ")
	Users.Flags().StringVar(&emailForUser, "userEmail", "", "specifying email user")
	Users.Flags().BoolVar(&emailVerified, "userEmailVerified", true, "specifying emailVerification user")
	Users.Flags().StringVar(&roleUser, "userRole", "", "specifying the roles user ")

	userDetails.Flags().StringVar(&workspacesName, "workspacesName", "", "specifying the workspace name  ")
	userDetails.Flags().StringVar(&userID, "userId", "", "specifying the userID")

	roles.Flags().StringVar(&workspacesName, "workspacesName", "", "specifying the workspace name  ")
	roleDetails.Flags().StringVar(&workspacesName, "workspacesName", "", "specifying the workspace name  ")
	roleDetails.Flags().StringVar(&role, "role", "", "specifying the role for details the role ")

	KeysCmd.Flags().StringVar(&workspacesName, "workspacesName", "", "specifying the workspace name  ")
	KeyDetailsCmd.Flags().StringVar(&workspacesName, "workspacesName", "", "specifying the workspace name")
	KeyDetailsCmd.Flags().StringVar(&KeyID, "keyID", "", "specifying the keyID ")

}
