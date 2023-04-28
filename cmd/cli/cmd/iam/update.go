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
var outputUpdate string
var Update = &cobra.Command{
	Use:   "update",
	Short: "it is use for update user or key user  ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return nil
	},
}
var UpdateUser = &cobra.Command{
	Use:   "user-access",
	Short: "update the account user ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("workspaceName").Changed {
		} else {
			fmt.Println("please enter the workspaceName flag .")
			return cmd.Help()
		}
		if cmd.Flags().ParseErrorsWhitelist.UnknownFlags {
			fmt.Println("please enter right flag .")
			return cmd.Help()
		}
		if len(args) == 0 {
			return cmd.Help()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := apis.GetConfig()
		if err != nil {
			return err
		}
		checkEXP, err := apis.CheckExpirationTime(accessToken)
		if err != nil {
			return err
		}
		if checkEXP == true {
			fmt.Println("your access token was expire please login again ")
			return nil
		}

		response, err := apis.IamUpdateUser(workspacesNameUpdate, accessToken, roleUpdate, userId)
		if err != nil {
			return err
		}
		fmt.Println(response)
		return nil
	},
}

var StateWorkspaceKey = &cobra.Command{
	Use:   "key-state",
	Short: "update the state key user ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("workspaceName").Changed {
		} else {
			fmt.Println("please enter the workspaceName flag .")
			return cmd.Help()
		}
		if cmd.Flags().ParseErrorsWhitelist.UnknownFlags {
			fmt.Println("please enter right flag .")
			return cmd.Help()
		}
		if len(args) == 0 {
			return cmd.Help()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := apis.GetConfig()
		if err != nil {
			return err
		}
		checkEXP, err := apis.CheckExpirationTime(accessToken)
		if err != nil {
			return err
		}
		if checkEXP == true {
			fmt.Println("your access token was expire please login again ")
			return nil
		}
		var response apies.WorkspaceApiKey
		if state == "active " {
			response, err = apis.IamActivateKey(workspacesNameUpdate, accessToken, idForSuspendAndActive)
		} else if state == "suspend" {
			response, err = apis.IamSuspendKey(workspacesNameUpdate, accessToken, idForSuspendAndActive)
			if err != nil {
				return err
			}
		} else {
			fmt.Println("the state you are looking for is wrong please just choose one of these tow : \n 1. active \n 2. suspend ")
		}
		err = apis.PrintOutput(response, outputUpdate)
		if err != nil {
			return err
		}
		return nil
	},
}

var UpdateKeyRole = &cobra.Command{
	Use:   "key-role",
	Short: "update key role",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("workspaceName").Changed {
		} else {
			fmt.Println("please enter the workspaceName flag .")
			return cmd.Help()
		}
		if cmd.Flags().ParseErrorsWhitelist.UnknownFlags {
			fmt.Println("please enter right flag .")
			return cmd.Help()
		}
		if len(args) == 0 {
			return cmd.Help()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := apis.GetConfig()
		if err != nil {
			return err
		}
		checkEXP, err := apis.CheckExpirationTime(accessToken)
		if err != nil {
			return err
		}
		if checkEXP == true {
			fmt.Println("your access token was expire please login again ")
			return nil
		}
		response, err := apis.IamUpdateKeyRole(workspacesNameUpdate, accessToken, userIdForKey, role)
		if err != nil {
			return err
		}
		err = apis.PrintOutput(response, outputUpdate)
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	Update.AddCommand(IamUpdate)
	//update key role flag
	UpdateKeyRole.Flags().StringVar(&workspacesNameUpdate, "workspaceName", "", "specifying the workspaces name ")
	UpdateKeyRole.Flags().UintVar(&userIdForKey, "userId", 0, "specifying the userID")
	UpdateKeyRole.Flags().StringVar(&roleUpdate, "role", "", "specifying the role name ")
	UpdateKeyRole.Flags().StringVar(&outputUpdate, "output", "", "specifying the output type  [json, table]")
	//update workspace key flag
	StateWorkspaceKey.Flags().StringVar(&idForSuspendAndActive, "id", "", "specifying the id for suspend and active key")
	StateWorkspaceKey.Flags().StringVar(&workspacesNameUpdate, "workspaceName", "", "specifying the workspaces name ")
	StateWorkspaceKey.Flags().StringVar(&state, "state", "", "specifying the state workspace key ")
	StateWorkspaceKey.Flags().StringVar(&outputUpdate, "output", "", "specifying the output type  [json, table]")

	//update user flags
	UpdateUser.Flags().StringVar(&workspacesNameUpdate, "workspaceName", "", "specifying the workspaces name")
	UpdateUser.Flags().StringVar(&userId, "userId", "", "specifying the userID")
	UpdateUser.Flags().StringVar(&roleUpdate, "role", "", "specifying the role name")
	UpdateUser.Flags().StringVar(&outputUpdate, "output", "", "specifying the output type  [json, table] ")

}
