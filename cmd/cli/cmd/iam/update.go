package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	apies "gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	apis "gitlab.com/keibiengine/keibi-engine/pkg/cli"
)

var idForSuspendAndActive string
var userIdForKey uint
var workspacesNameUpdate string
var userId string
var roleUpdate string
var state string

var Update = &cobra.Command{
	Use:   "update",
	Short: "update something ",
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
		fmt.Println("update")
	},
}
var UserUpdate = &cobra.Command{
	Use:   "user-access",
	Short: "update the account user ",
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
		response, err := apis.UpdateUser(workspacesNameUpdate, accessToken, roleUpdate, userId)
		if err != nil {
			return err
		}
		fmt.Println(response)
		return nil
	},
}

var StateWorkspaceKey = &cobra.Command{
	Use:   "key-state",
	Short: "Print the version number of Hugo",
	Long:  `All software has versions. This is Hugo's`,
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
		var response apies.WorkspaceApiKey
		if state == "active " {
			response, err = apis.ActiveKey(workspacesNameUpdate, accessToken, idForSuspendAndActive)
		} else if state == "suspend" {
			response, err = apis.SuspendKey(workspacesNameUpdate, accessToken, idForSuspendAndActive)
			if err != nil {
				return err
			}
		} else {
			fmt.Println("the state you are looking for is wrong please just choose one of these tow : \n 1. active \n 2. suspend ")
		}
		err = apis.PrintOutput(response, "table")
		if err != nil {
			return err
		}
		return nil
	},
}

var UpdateKeyRole = &cobra.Command{
	Use:   "key-role",
	Short: "Print the version number of Hugo",
	Long:  `All software has versions. This is Hugo's`,
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
		response, err := apis.UpdateKeyRole(workspacesNameUpdate, accessToken, userIdForKey, role)
		if err != nil {
			return err
		}
		err = apis.PrintOutput(response, "table")
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	Update.AddCommand(IamUpdate)

	UpdateKeyRole.Flags().StringVar(&workspacesNameUpdate, "workspaceName", "", "specifying the workspacesName ")
	UpdateKeyRole.Flags().UintVar(&userIdForKey, "userId", 0, "specifying the userID")
	UpdateKeyRole.Flags().StringVar(&roleUpdate, "role", "", "specifying the roles user ")
	//
	StateWorkspaceKey.Flags().StringVar(&idForSuspendAndActive, "id", "", "specifying the id for suspend and active key")
	StateWorkspaceKey.Flags().StringVar(&workspacesNameUpdate, "workspaceName", "", "specifying the workspacesName ")
	StateWorkspaceKey.Flags().StringVar(&state, "state", "", "specifying the state workspace key ")
	//
	UserUpdate.Flags().StringVar(&workspacesNameUpdate, "workspaceName", "", "specifying the workspacesName user  ")
	UserUpdate.Flags().StringVar(&userId, "userId", "", "specifying the userID")
	UserUpdate.Flags().StringVar(&roleUpdate, "role", "", "specifying the roles user ")

}
