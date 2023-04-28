package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	apis "gitlab.com/keibiengine/keibi-engine/pkg/cli"
)

var workspacesNameCreate string
var email string
var roleForUser string
var roleName string
var nameKey string
var outputCreate string

var Create = &cobra.Command{
	Use:   "create",
	Short: "this use for create user or key",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			err := cmd.Help()
			if err != nil {
				return err
			}
		}
		return nil
	},
}

var CreateUser = &cobra.Command{
	Use:   "user",
	Short: "create a user ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("workspaceName").Changed {
		} else {
			fmt.Println("please enter the workspaceName flag .")
		}

		if cmd.Flags().ParseErrorsWhitelist.UnknownFlags != false {
			fmt.Println("please enter right flag .")
		}

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
		checkEXP, err := apis.CheckExpirationTime(accessToken)
		if err != nil {
			return err
		}
		if checkEXP == true {
			fmt.Println("your access token was expire please login again ")
			return nil
		}

		response, err := apis.IamCreateUser(workspacesNameCreate, accessToken, email, roleForUser)
		if err != nil {
			return err
		}
		fmt.Println(response)
		return nil
	},
}

var CreateKeyCmd = &cobra.Command{
	Use:   "keys",
	Short: "create a Key for user ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("workspaceName").Changed {
		} else {
			fmt.Println("please enter the workspaceName flag .")
		}
		if cmd.Flags().ParseErrorsWhitelist.UnknownFlags != false {
			fmt.Println("please enter right  flag .")
		}
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
		checkEXP, err := apis.CheckExpirationTime(accessToken)
		if err != nil {
			return err
		}
		if checkEXP == true {
			fmt.Println("your access token was expire please login again ")
			return nil
		}
		response, err := apis.IamCreateKeys(workspacesNameCreate, accessToken, nameKey, roleName)
		if err != nil {
			return err
		}
		err = apis.PrintOutput(response, outputCreate)
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	Create.AddCommand(IamCreate)
	//flags create user :
	CreateUser.Flags().StringVar(&workspacesNameCreate, "workspaceName", "", "specifying the workspaces name ")
	CreateUser.Flags().StringVar(&email, "email", "", "specifying the user email ")
	CreateUser.Flags().StringVar(&roleForUser, "role", "", "specifying the user role ")
	//flags create keys :
	CreateKeyCmd.Flags().StringVar(&workspacesNameCreate, "workspaceName", "", "specifying the workspace name ")
	CreateKeyCmd.Flags().StringVar(&roleName, "roleName", "", "specifying the role name ")
	CreateKeyCmd.Flags().StringVar(&nameKey, "keyName", "", "specifying the key name ")
	CreateKeyCmd.Flags().StringVar(&outputCreate, "output", "", "specifying the output type [json, table].")
}
