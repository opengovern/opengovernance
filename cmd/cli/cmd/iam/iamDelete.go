package cmd

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	apis "gitlab.com/keibiengine/keibi-engine/pkg/cli"
)

var workspacesNameForDelete string
var UserIdForDelete string
var KeyIdForDelete string

func init() {
	IamDelete.AddCommand(DeleteUserInvite)
	IamDelete.AddCommand(DeleteUserAccess)
	IamDelete.AddCommand(DeleteKey)
	//flags delete user
	DeleteUserInvite.Flags().StringVar(&workspacesNameForDelete, "workspace-name", "", "specifying the workspace name[mandatory] ")
	DeleteUserInvite.Flags().StringVar(&UserIdForDelete, "user-id", "", "specifying the userId [mandatory]")
	DeleteUserAccess.Flags().StringVar(&workspacesNameForDelete, "workspace-name", "", "specifying the workspace name[mandatory] ")
	DeleteUserAccess.Flags().StringVar(&UserIdForDelete, "user-id", "", "specifying the userId [mandatory]")

	//flags delete key
	DeleteKey.Flags().StringVar(&workspacesNameForDelete, "workspace-name", "", "specifying the workspace name[mandatory] ")
	DeleteKey.Flags().StringVar(&KeyIdForDelete, "key-id", "", "specifying the keyID [mandatory]")

}

var IamDelete = &cobra.Command{
	Use: "iam",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
var DeleteUserAccess = &cobra.Command{
	Use:   "user-Access",
	Short: "delete user access",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("user-id").Changed {
		} else {
			return errors.New("please enter the userId flag. ")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}

		response, err := apis.IamDeleteUserAccess(cnf.DefaultWorkspace, cnf.AccessToken, UserIdForDelete)
		if err != nil {
			return err
		}
		fmt.Println(response)
		return nil
	},
}
var DeleteUserInvite = &cobra.Command{
	Use:   "user-invite",
	Short: "delete user invite ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("user-id").Changed {
		} else {
			return errors.New("please enter the userId flag. ")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}

		response, err := apis.IamDeleteUserInvite(cnf.DefaultWorkspace, cnf.AccessToken, UserIdForDelete)
		if err != nil {
			return err
		}
		fmt.Println(response)
		return nil
	},
}

var DeleteKey = &cobra.Command{
	Use:   "key",
	Short: "delete key",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("key-id").Changed {
		} else {
			return errors.New("please enter the keyId flag. ")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}

		response, err := apis.IamDeleteKey(cnf.DefaultWorkspace, cnf.AccessToken, KeyIdForDelete)
		if err != nil {
			return err
		}
		fmt.Println(response)
		return nil
	},
}
