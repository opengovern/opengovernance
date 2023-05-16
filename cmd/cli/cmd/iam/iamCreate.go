package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	apis "gitlab.com/keibiengine/keibi-engine/pkg/cli"
	"log"
)

var workspacesNameCreate string
var email string
var roleForUser string
var roleName string
var nameKey string
var outputIamCreate string

var IamCreate = &cobra.Command{
	Use:   "iam",
	Short: "iam command ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var CreateUser = &cobra.Command{
	Use:   "user",
	Short: "create a user ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("role").Changed {
		} else {
			fmt.Println("please enter the role flag .")
			log.Fatalln(cmd.Help())
		}
		if cmd.Flags().Lookup("email").Changed {
		} else {
			fmt.Println("please enter the email flag .")
			log.Fatalln(cmd.Help())
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}

		response, err := apis.IamCreateUser(cnf.DefaultWorkspace, cnf.AccessToken, email, roleForUser)
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
		if cmd.Flags().Lookup("role-name").Changed {
		} else {
			fmt.Println("please enter the roleName flag .")
			log.Fatalln(cmd.Help())
		}
		if cmd.Flags().Lookup("key-name").Changed {
		} else {
			fmt.Println("please enter the keyName flag .")
			log.Fatalln(cmd.Help())
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}

		response, err := apis.IamCreateKeys(cnf.DefaultWorkspace, cnf.AccessToken, nameKey, roleName)
		if err != nil {
			return err
		}

		err = apis.PrintOutput(response, outputIamCreate)
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	IamCreate.AddCommand(CreateKeyCmd)
	IamCreate.AddCommand(CreateUser)
	//flags create user :
	CreateUser.Flags().StringVar(&workspacesNameCreate, "workspace-name", "", "specifying the workspaces name [mandatory] .")
	CreateUser.Flags().StringVar(&email, "email", "", "specifying the user email [mandatory]")
	CreateUser.Flags().StringVar(&roleForUser, "role", "", "specifying the user role[mandatory] ")
	//flags create keys :
	CreateKeyCmd.Flags().StringVar(&workspacesNameCreate, "workspace-name", "", "specifying the workspace name [mandatory].")
	CreateKeyCmd.Flags().StringVar(&roleName, "role-name", "", "specifying the role name [mandatory].")
	CreateKeyCmd.Flags().StringVar(&nameKey, "key-name", "", "specifying the key name[mandatory] .")
	CreateKeyCmd.Flags().StringVar(&outputIamCreate, "output-type", "table", "specifying the output type [json, table][optional].")

}
