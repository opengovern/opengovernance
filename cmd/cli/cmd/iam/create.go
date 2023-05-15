package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/cli"
	"log"
	"net/http"
)

var workspacesNameCreate string
var email string
var roleForUser string
var roleName string
var nameKey string
var outputCreate = "table"

// onboard command :
var configCredential string
var nameCredential string
var sourceTypeCredential string
var outputTypeCreate string

// aws variables:
var workspaceNameAWS string
var nameAWS string
var emailAWS string
var descriptionAWS string
var accessIdAWS string
var accessKeyAWS string
var regionsAWS []string
var secretKey string
var outputTypeAWS string

//TODO-saber fix problem workspaceName

// azure variables :
var outputTypeAzure string
var workspaceNameAzure string
var nameAzure string
var descriptionAzure string
var clientIdAzure string
var clientSecretAzure string
var secretIdAzure string
var subscriptionIdAzure string
var tenantIdAzure string
var ObjectId string

//TODO-saber fix problem workspaceName

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

// onboard command :
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
var AwsCmd = &cobra.Command{
	Use:   "aws",
	Short: "onboard command",
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
		response, status, err := cli.OnboardCreateAWS(accessToken, nameAWS, emailAWS, descriptionAWS, accessKeyAWS, accessIdAWS, regionsAWS, secretKey)
		if status != http.StatusOK {
			if err != nil {
				return err
			}
			log.Fatalln("something is wrong")
		}
		if status == http.StatusOK {
			fmt.Println("OK")
		}
		err = cli.PrintOutput(response, "table")
		if err != nil {
			return err
		}
		return nil
	},
}

var AzureCmd = &cobra.Command{
	Use:   "azure",
	Short: "azure command",
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
		response, status, err := cli.OnboardCreateAzure(accessToken, nameAzure, ObjectId, descriptionAzure, clientIdAzure, clientSecretAzure, subscriptionIdAzure, tenantIdAzure)
		if status != http.StatusOK {
			if err != nil {
				return err
			}
			log.Fatalln("something is wrong")
		}
		if status == http.StatusOK {
			fmt.Println("OK")
		}
		err = cli.PrintOutput(response, outputTypeAzure)
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	Create.AddCommand(IamCreate)
	//onboard flags :
	// credential flags :
	credentialCreateCmd.Flags().StringVar(&outputTypeCreate, "output", "table", "it is specifying the output type [table , json][optional]")
	credentialCreateCmd.Flags().StringVar(&configCredential, "config", "", "it is specifying the config credential [mandatory].")
	credentialCreateCmd.Flags().StringVar(&nameCredential, "name", "", "it is specifying the name credential [mandatory].")
	credentialCreateCmd.Flags().StringVar(&sourceTypeCredential, "sourceType", "", "it is specifying the source type credential [mandatory].")
	// aws flags :
	AwsCmd.Flags().StringVar(&outputTypeAWS, "outputType", "", "specifying the output type [optional].")
	AwsCmd.Flags().StringVar(&nameAWS, "name", "", "specifying the name for AWS[mandatory]")
	AwsCmd.Flags().StringVar(&emailAWS, "email", "", "specifying the email for AWS[mandatory]")
	AwsCmd.Flags().StringVar(&descriptionAWS, "description", "", "specifying the name for AWS[mandatory]")
	AwsCmd.Flags().StringVar(&accessKeyAWS, "accessId", "", "specifying the accessId for AWS[mandatory]")
	AwsCmd.Flags().StringVar(&accessIdAWS, "accessKey", "", "specifying the accessKey for AWS[mandatory]")
	AwsCmd.Flags().StringSliceVar(&regionsAWS, "regions", []string{}, "specifying the regions for AWS[optional]")
	AwsCmd.Flags().StringVar(&secretKey, "secretKey", "", "specifying the secretKey for AWS[mandatory]")
	//	azure flags :
	AzureCmd.Flags().StringVar(&outputTypeAzure, "outputType", "", "specifying the output type [optional].")
	AzureCmd.Flags().StringVar(&nameAzure, "name", "", "specifying the name for AZURE[mandatory]")
	AzureCmd.Flags().StringVar(&descriptionAzure, "description", "", "specifying the description for AZURE[mandatory]")
	AzureCmd.Flags().StringVar(&clientSecretAzure, "clientSecret", "", "specifying the clientSecret for AZURE[mandatory]")
	AzureCmd.Flags().StringVar(&clientIdAzure, "clientId", "", "specifying the clientId for AZURE[mandatory]")
	AzureCmd.Flags().StringVar(&secretIdAzure, "secretId", "", "specifying the secretId for AZURE[mandatory]")
	AzureCmd.Flags().StringVar(&subscriptionIdAzure, "subscriptionId", "", "specifying the subscriptionId for AZURE[mandatory]")
	AzureCmd.Flags().StringVar(&tenantIdAzure, "tenantId", "", "specifying the tenantId for AZURE[mandatory]")
	AzureCmd.Flags().StringVar(&ObjectId, "objectId", "", "specifying the ObjectId for AZURE[mandatory]")

}
