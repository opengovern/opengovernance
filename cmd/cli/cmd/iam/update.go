package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	apis "gitlab.com/keibiengine/keibi-engine/pkg/cli"
)

var userIdForKey uint
var workspacesNameUpdate string
var userId string
var roleUpdate string
var Update = &cobra.Command{
	Use:   "update",
	Short: "update something ",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("update")
	},
}
var UserUpdate = &cobra.Command{
	Use:   "user-access",
	Short: "update the account user ",
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := apis.GetConfig()
		if err != nil {
			return err
		}
		response, err := apis.UpdateUser(workspacesNameUpdate, accessToken, roleUpdate, userId)
		if err != nil {
			return err
		}
		fmt.Println(response)
		return nil
	},
}

var UpdateKeyRole = &cobra.Command{
	Use:   "user-role",
	Short: "update Key-role",
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := apis.GetConfig()
		if err != nil {
			return err
		}
		response, err := apis.UpdateKeyRole(workspacesNameUpdate, accessToken, userIdForKey, role)
		if err != nil {
			return err
		}
		fmt.Println(response)
		return nil
	},
}

//	var UpdateKeyState = &cobra.Command{
//		Use:   "user-access",
//		Short: "Print the version number of Hugo",
//		Long:  `All software has versions. This is Hugo's`,
//		RunE: func(cmd *cobra.Command, args []string) error {
//			accessToken, err := apis.GetConfig()
//			if err != nil {
//				return err
//			}
//			err = apis.UpdateKeyState(workspaceName, accessToken, IdForWorkspaceKeys)
//			if err != nil {
//				return err
//			}
//			return nil
//		},
//	}
func init() {
	Update.AddCommand(IamUpdate)

	UpdateKeyRole.Flags().StringVar(&workspacesNameUpdate, "workspacesName", "", "specifying the workspacesName ")
	UpdateKeyRole.Flags().UintVar(&userIdForKey, "userId", 0, "specifying the userID")
	UpdateKeyRole.Flags().StringVar(&roleUpdate, "role", "", "specifying the roles user ")

	UserUpdate.Flags().StringVar(&workspacesNameUpdate, "workspacesName", "", "specifying the workspacesName user  ")
	UserUpdate.Flags().StringVar(&userId, "userId", "", "specifying the userID")
	UserUpdate.Flags().StringVar(&roleUpdate, "role", "", "specifying the roles user ")
}
