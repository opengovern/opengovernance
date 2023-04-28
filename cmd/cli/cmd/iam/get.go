package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	apis "gitlab.com/keibiengine/keibi-engine/pkg/cli"
)

var KeyID string
var role string
var workspacesNameGet string
var userID string
var emailVerified bool
var roleUser string
var emailForUser string
var outputType string

var Get = &cobra.Command{
	Use:   "get",
	Short: "get command",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return nil
	},
}

var userDetails = &cobra.Command{
	Use:   "user-details",
	Short: "print the Specifications of a user  ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("workspaceName").Changed {
		} else {
			fmt.Println("please enter the workspaceName flag .")
			return cmd.Help()
		}

		if cmd.Flags().ParseErrorsWhitelist.UnknownFlags != false {
			fmt.Println("please enter right  flag .")
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
		if cmd.Flags().ParseErrorsWhitelist.UnknownFlags != false {
			fmt.Println("please enter right flag .")
			return cmd.Help()
		}
		if cmd.Flags().Lookup("workspaceName").Changed {
		} else {
			fmt.Println("please enter the workspaceName flag .")
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
			return nil
		}
		if cmd.Flags().ParseErrorsWhitelist.UnknownFlags != false {
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

var roles = &cobra.Command{
	Use:   "roles",
	Short: "Print the roles ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("workspaceName").Changed {
		} else {
			fmt.Println("please enter the workspaceName flag ")
			return cmd.Help()
		}
		if cmd.Flags().ParseErrorsWhitelist.UnknownFlags {
			fmt.Println("please enter right flag ")
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
			return nil
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

func init() {
	Get.AddCommand(IamGet)
	// users flags
	Users.Flags().StringVar(&workspacesNameGet, "workspaceName", "", "specifying the workspace name ")
	Users.Flags().StringVar(&emailForUser, "userEmail", "", "specifying email user")
	Users.Flags().BoolVar(&emailVerified, "userEmailVerified", true, "specifying emailVerification user")
	Users.Flags().StringVar(&roleUser, "userRole", "", "specifying the roles user ")
	Users.Flags().StringVar(&outputType, "output", "", "specifying the output type [json, table].")
	userDetails.Flags().StringVar(&workspacesNameGet, "workspaceName", "", "specifying the workspace name  ")
	userDetails.Flags().StringVar(&userID, "userId", "", "specifying the userID")
	userDetails.Flags().StringVar(&outputType, "output", "", "specifying the output type [json, table].")
	//roles flags
	roles.Flags().StringVar(&workspacesNameGet, "workspaceName", "", "specifying the workspace name  ")
	roles.Flags().StringVar(&outputType, "output", "", "specifying the output type  [json, table].")
	roleDetails.Flags().StringVar(&workspacesNameGet, "workspaceName", "", "specifying the workspace name  ")
	roleDetails.Flags().StringVar(&role, "role", "", "specifying the role for details role ")
	roleDetails.Flags().StringVar(&outputType, "output", "", "specifying the output type [json, table].")
	//keys flags
	KeysCmd.Flags().StringVar(&workspacesNameGet, "workspaceName", "", "specifying the workspace name  ")
	KeysCmd.Flags().StringVar(&outputType, "output", "", "specifying the output type [json, table].")
	KeyDetailsCmd.Flags().StringVar(&workspacesNameGet, "workspaceName", "", "specifying the workspace name")
	KeyDetailsCmd.Flags().StringVar(&KeyID, "keyID", "", "specifying the keyID ")
	KeyDetailsCmd.Flags().StringVar(&outputType, "output", "", "specifying the output type [json, table].")

}
