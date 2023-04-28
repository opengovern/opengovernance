package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	apis "gitlab.com/keibiengine/keibi-engine/pkg/cli"
)

var Get = &cobra.Command{
	Use:   "get",
	Short: "get command",
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
		fmt.Println("get command")
	},
}

var userDetails = &cobra.Command{
	Use:   "user-details",
	Short: "print the Specifications of a user  ",
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
		userDetails, err := apis.GetIamUserDetails(accessToken, workspacesNameGet, userID)
		if err != nil {
			return err
		}
		err = apis.PrintOutput(userDetails, outputType)
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
		if cmd.Flags().ParseErrorsWhitelist.UnknownFlags != false {
			fmt.Println("please enter right  flag .")
		}
		if cmd.Flags().Lookup("workspaceName").Changed {
		} else {
			fmt.Println("please enter the workspaceName flag .")
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
		response, err := apis.RoleDetail(workspacesNameGet, role, accessToken)
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
		response, err := apis.ListRoles(workspacesNameGet, accessToken)
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
		response, err := apis.GetListKeys(workspacesNameGet, accessToken)
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
		response, err := apis.GetKeyDetails(workspacesNameGet, accessToken, KeyID)
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

var KeyID string
var role string
var workspacesNameGet string
var userID string
var emailVerified bool
var roleUser string
var emailForUser string
var outputType string

func init() {
	Get.AddCommand(IamGet)

	Users.Flags().StringVar(&workspacesNameGet, "workspaceName", "", "specifying the workspace name ")
	Users.Flags().StringVar(&emailForUser, "userEmail", "", "specifying email user")
	Users.Flags().BoolVar(&emailVerified, "userEmailVerified", true, "specifying emailVerification user")
	Users.Flags().StringVar(&roleUser, "userRole", "", "specifying the roles user ")
	Users.Flags().StringVar(&outputType, "output", "", "specifying the output type .")

	userDetails.Flags().StringVar(&workspacesNameGet, "workspaceName", "", "specifying the workspace name  ")
	userDetails.Flags().StringVar(&userID, "userId", "", "specifying the userID")
	userDetails.Flags().StringVar(&outputType, "output", "", "specifying the output type .")

	roles.Flags().StringVar(&workspacesNameGet, "workspaceName", "", "specifying the workspace name  ")
	roles.Flags().StringVar(&outputType, "output", "", "specifying the output type .")
	roleDetails.Flags().StringVar(&workspacesNameGet, "workspaceName", "", "specifying the workspace name  ")
	roleDetails.Flags().StringVar(&role, "role", "", "specifying the role for details the role ")
	roleDetails.Flags().StringVar(&outputType, "output", "", "specifying the output type .")

	KeysCmd.Flags().StringVar(&workspacesNameGet, "workspaceName", "", "specifying the workspace name  ")
	KeysCmd.Flags().StringVar(&outputType, "output", "", "specifying the output type .")
	KeyDetailsCmd.Flags().StringVar(&workspacesNameGet, "workspaceName", "", "specifying the workspace name")
	KeyDetailsCmd.Flags().StringVar(&KeyID, "keyID", "", "specifying the keyID ")
	KeyDetailsCmd.Flags().StringVar(&outputType, "output", "", "specifying the output type .")

}
