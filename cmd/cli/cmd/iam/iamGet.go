package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	apis "gitlab.com/keibiengine/keibi-engine/pkg/cli"
	"log"
)

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
	Users.Flags().StringVar(&workspacesNameGet, "workspaceName", "", "specifying the workspace name[mandatory]. ")
	Users.Flags().StringVar(&emailForUser, "userEmail", "", "specifying email user[optional] .")
	Users.Flags().BoolVar(&emailVerified, "userEmailVerified", true, "specifying emailVerification user[optional]. ")
	Users.Flags().StringVar(&roleUser, "userRole", "", "specifying the roles user [optional]. ")
	Users.Flags().StringVar(&outputType, "output", "", "specifying the output type [json, table][optional] .")
	userDetails.Flags().StringVar(&workspacesNameGet, "workspaceName", "", "specifying the workspace name[mandatory].  ")
	userDetails.Flags().StringVar(&userID, "userId", "", "specifying the userID[mandatory].")
	userDetails.Flags().StringVar(&outputType, "output", "", "specifying the output type [json, table][optional] .")
	//roles flags
	roles.Flags().StringVar(&workspacesNameGet, "workspaceName", "", "specifying the workspace name[mandatory].  ")
	roles.Flags().StringVar(&outputType, "output", "", "specifying the output type  [json, table][optional] .")
	roleDetails.Flags().StringVar(&workspacesNameGet, "workspaceName", "", "specifying the workspace name [mandatory]. ")
	roleDetails.Flags().StringVar(&role, "role", "", "specifying the role for details role [mandatory]. ")
	roleDetails.Flags().StringVar(&outputType, "output", "", "specifying the output type [json, table][optional] .")

	listRoleUsers.Flags().StringVar(&role, "role", "", "specifying the role[mandatory].")
	listRoleUsers.Flags().StringVar(&outputType, "output", "", "specifying the output type [json, table][optional] .")
	listRoleUsers.Flags().StringVar(&workspacesNameGet, "workspaceName", "", "specifying the workspace name [mandatory]. ")

	listRoleKeys.Flags().StringVar(&role, "role", "", "specifying the role[mandatory].")
	listRoleKeys.Flags().StringVar(&outputType, "output", "", "specifying the output type [json, table][optional] .")
	listRoleKeys.Flags().StringVar(&workspacesNameGet, "workspaceName", "", "specifying the workspace name [mandatory]. ")

	//keys flags
	KeysCmd.Flags().StringVar(&workspacesNameGet, "workspaceName", "", "specifying the workspace name [mandatory] ")
	KeysCmd.Flags().StringVar(&outputType, "output", "", "specifying the output type [json, table][optional] .")
	KeyDetailsCmd.Flags().StringVar(&workspacesNameGet, "workspaceName", "", "specifying the workspace name[mandatory]")
	KeyDetailsCmd.Flags().StringVar(&KeyID, "keyID", "", "specifying the keyID [mandatory]+")
	KeyDetailsCmd.Flags().StringVar(&outputType, "output", "", "specifying the output type [json, table][optional] .")

}

var KeyID string
var role string
var workspacesNameGet string
var userID string

var emailVerified bool
var roleUser string
var emailForUser string
var outputType = "table"

var IamGet = &cobra.Command{
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

var userDetails = &cobra.Command{
	Use:   "user-details",
	Short: "print the Specifications of a user  ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("workspaceName").Changed {
		} else {
			fmt.Println("please enter the workspaceName flag .")
			log.Fatalln(cmd.Help())
		}
		if cmd.Flags().Lookup("userId").Changed {
		} else {
			fmt.Println("please enter the userId flag .")
			log.Fatalln(cmd.Help())
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
		response, err := apis.IamGetUserDetails(accessToken, workspacesNameGet, userID)
		if err != nil {
			return err
		}
		err = apis.PrintOutput(response, outputType)
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
		if cmd.Flags().Lookup("workspaceName").Changed {
		} else {
			fmt.Println("please enter the workspaceName flag .")
			log.Fatalln(cmd.Help())
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
		users, err := apis.IamGetUsers(workspacesNameGet, accessToken, emailForUser, emailVerified, roleUser)
		if err != nil {
			return err
		}
		err = apis.PrintOutputForTypeArray(users, outputType)
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
		if cmd.Flags().Lookup("workspaceName").Changed {
		} else {
			fmt.Println("please enter the workspaceName flag .")
			log.Fatalln(cmd.Help())
		}
		if cmd.Flags().Lookup("role").Changed {
		} else {
			fmt.Println("please enter the role flag .")
			log.Fatalln(cmd.Help())
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

		response, err := apis.IamRoleDetails(workspacesNameGet, role, accessToken)
		if err != nil {
			return err
		}
		err = apis.PrintOutput(response, outputType)
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
		if cmd.Flags().Lookup("workspaceName").Changed {
		} else {
			fmt.Println("please enter the workspaceName flag ")
			log.Fatalln(cmd.Help())
		}
		if cmd.Flags().Lookup("role").Changed {
		} else {
			fmt.Println("please enter the role flag ")
			log.Fatalln(cmd.Help())
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

		response, err := apis.IamListRoleUsers(workspacesNameGet, accessToken, role)
		if err != nil {
			return err
		}
		err = apis.PrintOutput(response, outputType)
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
		if cmd.Flags().Lookup("workspaceName").Changed {
		} else {
			fmt.Println("please enter the workspaceName flag ")
			log.Fatalln(cmd.Help())
		}
		if cmd.Flags().Lookup("role").Changed {
		} else {
			fmt.Println("please enter the role flag ")
			log.Fatalln(cmd.Help())
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

		response, err := apis.IamListRoleKeys(workspacesNameGet, accessToken, role)
		if err != nil {
			return err
		}
		err = apis.PrintOutputForTypeArray(response, outputType)
		if err != nil {
			return err
		}
		return nil
	},
}

var roles = &cobra.Command{
	Use:   "roles",
	Short: "Print the roles ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("workspaceName").Changed {
		} else {
			fmt.Println("please enter the workspaceName flag ")
			log.Fatalln(cmd.Help())
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

		response, err := apis.IamListRoles(workspacesNameGet, accessToken)
		if err != nil {
			return err
		}
		err = apis.PrintOutputForTypeArray(response, outputType)
		if err != nil {
			return err
		}
		return nil
	},
}
var KeysCmd = &cobra.Command{
	Use:   "keys",
	Short: "print the keys ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("workspaceName").Changed {
		} else {
			fmt.Println("please enter the workspaceName flag .")
			log.Fatalln(cmd.Help())
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
		response, err := apis.IamGetListKeys(workspacesNameGet, accessToken)
		if err != nil {
			return err
		}

		err = apis.PrintOutputForTypeArray(response, outputType)
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
		if cmd.Flags().Lookup("workspaceName").Changed {
		} else {
			fmt.Println("please enter the workspaceName flag .")
			log.Fatalln(cmd.Help())
		}
		if cmd.Flags().Lookup("keyId").Changed {
		} else {
			fmt.Println("please enter the key id flag .")
			log.Fatalln(cmd.Help())
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
		response, err := apis.IamGetKeyDetails(workspacesNameGet, accessToken, KeyID)
		if err != nil {
			return err
		}

		err = apis.PrintOutput(response, outputType)
		if err != nil {
			return err
		}
		return nil
	},
}
