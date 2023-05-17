package cmd

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	apies "gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	apis "gitlab.com/keibiengine/keibi-engine/pkg/cli"
)

var IamUpdate = &cobra.Command{
	Use: "iam",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
var UpdateUser = &cobra.Command{
	Use:   "user-access",
	Short: "update the account user ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("user-id").Changed {
		} else {
			return errors.New("please enter the userId flag. ")
		}
		if cmd.Flags().Lookup("role").Changed {
		} else {
			return errors.New("please enter the role flag. ")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}

		response, err := apis.IamUpdateUser(cnf.DefaultWorkspace, cnf.AccessToken, roleUpdate, userId)
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
		if cmd.Flags().Lookup("state").Changed {
		} else {
			return errors.New("please enter the state flag. ")
		}
		if cmd.Flags().Lookup("id").Changed {
		} else {
			return errors.New("please enter the id flag. ")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}

		var response apies.WorkspaceApiKey
		if state == "active " {
			response, err = apis.IamActivateKey(cnf.DefaultWorkspace, cnf.AccessToken, idForSuspendAndActive)
		} else if state == "suspend" {
			response, err = apis.IamSuspendKey(cnf.DefaultWorkspace, cnf.AccessToken, idForSuspendAndActive)
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
		if cmd.Flags().Lookup("user-id").Changed {
		} else {
			return errors.New("please enter the userId flag. ")
		}
		if cmd.Flags().Lookup("role").Changed {
		} else {
			return errors.New("please enter the role flag. ")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}

		response, err := apis.IamUpdateKeyRole(cnf.DefaultWorkspace, cnf.AccessToken, userIdForKey, role)
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

var idForSuspendAndActive string
var userIdForKey uint
var workspacesNameUpdate string
var userId string
var roleUpdate string
var state string
var outputUpdate string

func init() {

	IamUpdate.AddCommand(UpdateKeyRole)
	IamUpdate.AddCommand(UpdateUser)
	IamUpdate.AddCommand(StateWorkspaceKey)
	//update key role flag
	UpdateKeyRole.Flags().StringVar(&workspacesNameUpdate, "workspace-name", "", "specifying the workspaces name ")
	UpdateKeyRole.Flags().UintVar(&userIdForKey, "user-id", 0, "specifying the userID")
	UpdateKeyRole.Flags().StringVar(&roleUpdate, "role", "", "specifying the role name ")
	UpdateKeyRole.Flags().StringVar(&outputUpdate, "output-type", "table", "specifying the output type  [json, table]")

	//update workspace key flag
	StateWorkspaceKey.Flags().StringVar(&idForSuspendAndActive, "id", "", "specifying the id for suspend and active key")
	StateWorkspaceKey.Flags().StringVar(&workspacesNameUpdate, "workspace-name", "", "specifying the workspaces name ")
	StateWorkspaceKey.Flags().StringVar(&state, "state", "", "specifying the state workspace key ")
	StateWorkspaceKey.Flags().StringVar(&outputUpdate, "output-type", "table", "specifying the output type  [json, table]")

	//update user flags
	UpdateUser.Flags().StringVar(&workspacesNameUpdate, "workspace-name", "", "specifying the workspaces name")
	UpdateUser.Flags().StringVar(&userId, "user-id", "", "specifying the userID")
	UpdateUser.Flags().StringVar(&roleUpdate, "role", "", "specifying the role name")
	UpdateUser.Flags().StringVar(&outputUpdate, "output-type", "table", "specifying the output type  [json, table] ")

}
