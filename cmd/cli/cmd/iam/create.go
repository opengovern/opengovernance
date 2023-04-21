package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	apis "gitlab.com/keibiengine/keibi-engine/pkg/cli"
)

var Create = &cobra.Command{
	Use:   "create",
	Short: "this use for create something",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("create")
	},
}

var UserCreate = &cobra.Command{
	Use:   "user",
	Short: "create a profile for user ",
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := apis.GetConfig()
		if err != nil {
			return err
		}
		response, err := apis.CreateUser(workspacesNameCreate, accessToken, email, roleForUser)
		if err != nil {
			return err
		}
		fmt.Println(response)
		return nil
	},
}

var CreateKeyCmd = &cobra.Command{
	Use:   "keys",
	Short: "create Key user ",
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := apis.GetConfig()
		if err != nil {
			return err
		}
		response, err := apis.CreateKeys(workspacesNameCreate, accessToken, nameKey, api.Role(roleName))
		if err != nil {
			return err
		}
		fmt.Println(response)
		return nil
	},
}
var workspacesNameCreate string
var email string
var roleForUser string
var roleName string
var nameKey string

func init() {
	Create.AddCommand(IamCreate)

	UserCreate.Flags().StringVar(&workspacesNameCreate, "workspaceName", "", "specifying the workspaces name user")
	UserCreate.Flags().StringVar(&email, "email", "", "specifying the user email ")
	UserCreate.Flags().StringVar(&roleForUser, "role", "", "specifying the user role ")

	CreateKeyCmd.Flags().StringVar(&workspacesNameCreate, "workspaceName", "", "specifying the roles user ")
	CreateKeyCmd.Flags().StringVar(&roleName, "roleName", "", "")
	CreateKeyCmd.Flags().StringVar(&nameKey, "keyName", "", "specifying the roles user ")
}
