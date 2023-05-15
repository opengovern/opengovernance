package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/cli"
	"net/http"
)

var Delete = &cobra.Command{
	Use:   "delete",
	Short: "it is use for deleting user or key ",
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

var deleteCredentialCmd = &cobra.Command{
	Use:   "credential",
	Short: "it is remove credential",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := cli.GetConfig()
		if err != nil {
			return err
		}
		checkEXP, err := cli.CheckExpirationTime(accessToken)
		if err != nil {
			return err
		}
		if checkEXP == true {
			fmt.Println("your access token was expire please login again ")
			return nil
		}
		statusCode, err := cli.OnboardDeleteCredential(accessToken, credentialIdGet)
		if err != nil {
			return fmt.Errorf("ERROR : status : %v \n %v ", statusCode, err)
		}
		if statusCode == http.StatusOK {
			fmt.Println("OK")
			return nil
		}
		return nil
	},
}
var deleteSourceCmd = &cobra.Command{
	Use:   "source",
	Short: "it will delete source ",
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := cli.GetConfig()
		if err != nil {
			return err
		}
		checkEXP, err := cli.CheckExpirationTime(accessToken)
		if err != nil {
			return err
		}
		if checkEXP == true {
			fmt.Println("your access token was expire please login again ")
			return nil
		}
		statusCode, err := cli.OnboardDeleteSource(accessToken, sourceIdDelete)
		if err != nil {
			return fmt.Errorf("ERROR: status: %v \n %v ", statusCode, err)
		}
		if statusCode == http.StatusOK {
			fmt.Println("OK")
		}
		return nil
	},
}
var sourceIdDelete string
var credentialIdDelete string

func init() {
	Delete.AddCommand(IamDelete)
	Delete.AddCommand(deleteCredentialCmd)
	Delete.AddCommand(deleteSourceCmd)
	//delete source flag :
	deleteSourceCmd.Flags().StringVar(&sourceIdDelete, "id", "", "it is specifying the source id. ")
	//delete credential :
	deleteCredentialCmd.Flags().StringVar(&credentialIdDelete, "id", "", "it is specifying the credentialIdGet[mandatory].")
}
