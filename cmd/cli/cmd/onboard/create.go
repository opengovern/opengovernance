package onboard

import (
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/cli"
	"log"
)

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
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
var credentialCreateCmd = cobra.Command{
	Use:   "credential",
	Short: "create credential",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("config").Changed {
		} else {
			fmt.Println("Please enter the name for config credential.")
			log.Fatalln(cmd.Help())
		}
		if cmd.Flags().Lookup("name").Changed {
		} else {
			fmt.Println("Please enter the name credential.")
			log.Fatalln(cmd.Help())
		}
		if cmd.Flags().Lookup("sourceType").Changed {
		} else {
			fmt.Println("Please enter the source type credential.")
			log.Fatalln(cmd.Help())
		}
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
			fmt.Println("Your access token was expire please login again.")
			return nil
		}
		response, statusCode, err := cli.OnboardCreateConnectionCredentials(accessToken, configCredential, nameCredential, sourceTypeCredential)
		if err != nil {
			return fmt.Errorf("ERROR : status : %v \n %v", statusCode, err)
		}
		err = cli.PrintOutput(response, outputTypeCreate)
		if err != nil {
			return err
		}
		return nil
	},
}
var configCredential string
var nameCredential string
var sourceTypeCredential string
var outputTypeCreate string

func init() {
	Create.AddCommand()
	//	credential flags :
	credentialCreateCmd.Flags().StringVar(&outputTypeCreate, "output", "", "it is specifying the output type [table , json][optional]")
	credentialCreateCmd.Flags().StringVar(&configCredential, "config", "", "it is specifying the config credential [mandatory].")
	credentialCreateCmd.Flags().StringVar(&nameCredential, "name", "", "it is specifying the name credential [mandatory].")
	credentialCreateCmd.Flags().StringVar(&sourceTypeCredential, "sourceType", "", "it is specifying the source type credential [mandatory].")
}
