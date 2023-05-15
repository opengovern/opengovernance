package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	apis "gitlab.com/keibiengine/keibi-engine/pkg/cli"
	"log"
)

var workspacesNameForDelete string
var UserIdForDelete string
var KeyIdForDelete string

func init() {
	IamDelete.AddCommand(DeleteUserInvite)
	IamDelete.AddCommand(DeleteUserAccess)
	IamDelete.AddCommand(DeleteKey)
	//flags delete user
	DeleteUserInvite.Flags().StringVar(&workspacesNameForDelete, "workspaceName", "", "specifying the workspace name[mandatory] ")
	DeleteUserInvite.Flags().StringVar(&UserIdForDelete, "userId", "", "specifying the userId [mandatory]")
	DeleteUserAccess.Flags().StringVar(&workspacesNameForDelete, "workspaceName", "", "specifying the workspace name[mandatory] ")
	DeleteUserAccess.Flags().StringVar(&UserIdForDelete, "userId", "", "specifying the userId [mandatory]")

	//flags delete key
	DeleteKey.Flags().StringVar(&workspacesNameForDelete, "workspaceName", "", "specifying the workspace name[mandatory] ")
	DeleteKey.Flags().StringVar(&KeyIdForDelete, "keyId", "", "specifying the keyID [mandatory]")

}

var IamDelete = &cobra.Command{
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
var DeleteUserAccess = &cobra.Command{
	Use:   "user-Access",
	Short: "delete user access",
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

		response, err := apis.IamDeleteUserAccess(workspacesNameForDelete, accessToken, UserIdForDelete)
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

		response, err := apis.IamDeleteUserInvite(workspacesNameForDelete, accessToken, UserIdForDelete)
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
		if cmd.Flags().Lookup("workspaceName").Changed {
		} else {
			fmt.Println("please enter the workspaceName flag .")
			log.Fatalln(cmd.Help())
		}
		if cmd.Flags().Lookup("keyId").Changed {
		} else {
			fmt.Println("please enter the keyId flag .")
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

		response, err := apis.IamDeleteKey(workspacesNameForDelete, accessToken, KeyIdForDelete)
		if err != nil {
			return err
		}
		fmt.Println(response)
		return nil
	},
}
