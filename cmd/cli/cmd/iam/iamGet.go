package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	apis "gitlab.com/keibiengine/keibi-engine/pkg/cli"
	"log"
)

var KeyID string
var role string
var workspacesNameGet string
var userID string
var emailVerified bool
var roleUser string
var emailForUser string
var outputTypeIamGet string

var IamGet = &cobra.Command{
	Use: "iam",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var userDetails = &cobra.Command{
	Use:   "user-details",
	Short: "print the Specifications of a user  ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("user-id").Changed {
		} else {
			fmt.Println("please enter the userId flag .")
			return cmd.Help()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}

		response, err := apis.IamGetUserDetails(cnf.AccessToken, cnf.DefaultWorkspace, userID)
		if err != nil {
			return err
		}
		err = apis.PrintOutput(response, outputTypeIamGet)
		if err != nil {
			return err
		}
		return nil
	},
}

var Users = &cobra.Command{
	Use:   "users",
	Short: "print the users",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("email").Changed {
		} else {
			fmt.Println("please enter the email flag .")
			return cmd.Help()
		}
		if cmd.Flags().Lookup("email-verified").Changed {
		} else {
			fmt.Println("please enter the email-verified flag .")
			return cmd.Help()
		}
		if cmd.Flags().Lookup("role-name").Changed {
		} else {
			fmt.Println("please enter the role-name flag .")
			return cmd.Help()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}

		users, err := apis.IamGetUsers(cnf.DefaultWorkspace, cnf.AccessToken, emailForUser, emailVerified, roleUser)
		if err != nil {
			return err
		}
		err = apis.PrintOutputForTypeArray(users, outputTypeIamGet)
		if err != nil {
			return err
		}
		return nil
	},
}

var roleDetails = &cobra.Command{
	Use:   "role-details",
	Short: "Print the specification of a role ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("role").Changed {
		} else {
			fmt.Println("please enter the role flag .")
			return cmd.Help()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}

		response, err := apis.IamRoleDetails(cnf.DefaultWorkspace, role, cnf.AccessToken)
		if err != nil {
			return err
		}

		err = apis.PrintOutput(response, outputTypeIamGet)
		if err != nil {
			return err
		}
		return nil
	},
}

var listRoleUsers = &cobra.Command{
	Use:   "role-users",
	Short: "Print the role users",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("role").Changed {
		} else {
			fmt.Println("please enter the role flag ")
			return cmd.Help()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}

		response, err := apis.IamListRoleUsers(cnf.DefaultWorkspace, cnf.AccessToken, role)
		if err != nil {
			return err
		}

		err = apis.PrintOutput(response, outputTypeIamGet)
		if err != nil {
			return err
		}
		return nil
	},
}
var listRoleKeys = &cobra.Command{
	Use:   "role-keys",
	Short: "Print the role keys",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("role").Changed {
		} else {
			fmt.Println("please enter the role flag ")
			log.Fatalln(cmd.Help())
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}

		response, err := apis.IamListRoleKeys(cnf.DefaultWorkspace, cnf.AccessToken, role)
		if err != nil {
			return err
		}

		err = apis.PrintOutputForTypeArray(response, outputTypeIamGet)
		if err != nil {
			return err
		}
		return nil
	},
}

var roles = &cobra.Command{
	Use:   "roles",
	Short: "Print the roles ",
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		response, err := apis.IamListRoles(cnf.DefaultWorkspace, cnf.AccessToken)
		if err != nil {
			return err
		}
		err = apis.PrintOutputForTypeArray(response, outputTypeIamGet)
		if err != nil {
			return err
		}
		return nil
	},
}
var KeysCmd = &cobra.Command{
	Use:   "keys",
	Short: "print the keys ",
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}

		response, err := apis.IamGetListKeys(cnf.DefaultWorkspace, cnf.AccessToken)
		if err != nil {
			return err
		}

		err = apis.PrintOutputForTypeArray(response, outputTypeIamGet)
		if err != nil {
			return err
		}
		return nil
	},
}

var KeyDetailsCmd = &cobra.Command{
	Use:   "key-details",
	Short: "Print the specification of a key ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("key-id").Changed {
		} else {
			fmt.Println("please enter the key-id flag .")
			log.Fatalln(cmd.Help())
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}

		response, err := apis.IamGetKeyDetails(cnf.DefaultWorkspace, cnf.AccessToken, KeyID)
		if err != nil {
			return err
		}

		err = apis.PrintOutput(response, outputTypeIamGet)
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	IamGet.AddCommand(roles)
	IamGet.AddCommand(roleDetails)
	IamGet.AddCommand(listRoleUsers)
	IamGet.AddCommand(listRoleKeys)
	IamGet.AddCommand(KeysCmd)
	IamGet.AddCommand(KeyDetailsCmd)
	IamGet.AddCommand(userDetails)
	IamGet.AddCommand(Users)
	// users flags
	Users.Flags().StringVar(&workspacesNameGet, "workspace-name", "", "specifying the workspace name[mandatory]. ")
	Users.Flags().StringVar(&emailForUser, "email", "", "specifying email user[optional] .")
	Users.Flags().BoolVar(&emailVerified, "email-verified", true, "specifying emailVerification user[optional]. ")
	Users.Flags().StringVar(&roleUser, "role-name", "", "specifying the roles user [optional]. ")
	Users.Flags().StringVar(&outputTypeIamGet, "output-type", "table", "specifying the output type [json, table][optional] .")
	userDetails.Flags().StringVar(&workspacesNameGet, "workspace-name", "", "specifying the workspace name[mandatory].  ")
	userDetails.Flags().StringVar(&userID, "user-id", "", "specifying the userID[mandatory].")
	userDetails.Flags().StringVar(&outputTypeIamGet, "output-type", "table", "specifying the output type [json, table][optional] .")
	//roles flags
	roles.Flags().StringVar(&workspacesNameGet, "workspace-name", "", "specifying the workspace name[mandatory].  ")
	roles.Flags().StringVar(&outputTypeIamGet, "output-type", "table", "specifying the output type  [json, table][optional] .")
	roleDetails.Flags().StringVar(&workspacesNameGet, "workspace-name", "", "specifying the workspace name [mandatory]. ")
	roleDetails.Flags().StringVar(&role, "role", "", "specifying the role for details role [mandatory]. ")
	roleDetails.Flags().StringVar(&outputTypeIamGet, "output-type", "", "specifying the output type [json, table][optional] .")

	listRoleUsers.Flags().StringVar(&role, "role", "", "specifying the role[mandatory].")
	listRoleUsers.Flags().StringVar(&outputTypeIamGet, "output-type", "table", "specifying the output type [json, table][optional] .")
	listRoleUsers.Flags().StringVar(&workspacesNameGet, "workspace-name", "", "specifying the workspace name [mandatory]. ")

	listRoleKeys.Flags().StringVar(&role, "role", "", "specifying the role[mandatory].")
	listRoleKeys.Flags().StringVar(&outputTypeIamGet, "output-type", "table", "specifying the output type [json, table][optional] .")
	listRoleKeys.Flags().StringVar(&workspacesNameGet, "workspace-name", "", "specifying the workspace name [mandatory]. ")

	//keys flags
	KeysCmd.Flags().StringVar(&workspacesNameGet, "workspace-name", "", "specifying the workspace name [mandatory] ")
	KeysCmd.Flags().StringVar(&outputTypeIamGet, "output-type", "table", "specifying the output type [json, table][optional] .")
	KeyDetailsCmd.Flags().StringVar(&workspacesNameGet, "workspace-name", "", "specifying the workspace name[mandatory]")
	KeyDetailsCmd.Flags().StringVar(&KeyID, "key-id", "", "specifying the keyID [mandatory]+")
	KeyDetailsCmd.Flags().StringVar(&outputTypeIamGet, "output-type", "table", "specifying the output type [json, table][optional] .")
}
