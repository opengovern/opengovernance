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

var Delete = &cobra.Command{
	Use:   "delete",
	Short: "it is use for deleting user or key ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return nil
	},
}

var DeleteUser = &cobra.Command{
	Use:   "user",
	Short: "delete user",
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

		response, err := apis.IamDeleteUser(workspacesNameForDelete, accessToken, UserIdForDelete)
		if err != nil {
			return err
		}
		fmt.Println(response)
		return nil
	},
}

var DeleteKey = &cobra.Command{
	Use:   "user",
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

func init() {
	Delete.AddCommand(IamDelete)
	//flags delete user
	DeleteUser.Flags().StringVar(&workspacesNameForDelete, "workspaceName", "", "specifying the workspace name ")
	DeleteUser.Flags().StringVar(&UserIdForDelete, "userId", "", "specifying the userId ")
	//flags delete key
	DeleteKey.Flags().StringVar(&workspacesNameForDelete, "workspaceName", "", "specifying the workspace name ")
	DeleteKey.Flags().StringVar(&KeyIdForDelete, "keyId", "", "specifying the keyID ")

}
