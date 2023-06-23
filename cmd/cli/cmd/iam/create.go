package cmd

import (
	"errors"
	"github.com/spf13/cobra"
	"github.com/kaytu-io/kaytu-engine/pkg/cli"
)

// onboard command :
var workspaceNameCreate string
var configCredential string
var nameCredential string
var sourceTypeCredential string
var outputTypeCreate string

// aws variables:
var nameAWS string
var emailAWS string
var descriptionAWS string
var accessIdAWS string
var accessKeyAWS string
var regionsAWS []string
var secretKey string
var outputTypeAWS string

// azure variables :
var outputTypeAzure string
var nameAzure string
var descriptionAzure string
var clientIdAzure string
var clientSecretAzure string
var secretIdAzure string
var subscriptionIdAzure string
var tenantIdAzure string
var ObjectId string

var Create = &cobra.Command{
	Use: "create",
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
			return errors.New("Please enter the config flag. ")
		}
		if cmd.Flags().Lookup("name").Changed {
		} else {
			return errors.New("Please enter the name flag. ")
		}
		if cmd.Flags().Lookup("source-type").Changed {
		} else {
			return errors.New("Please enter the source-type flag. ")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := cli.GetConfig(cmd, true)
		if err != nil {
			return err
		}

		response, err := cli.OnboardCreateConnectionCredentials(cnf.DefaultWorkspace, cnf.AccessToken, configCredential, nameCredential, sourceTypeCredential)
		if err != nil {
			return err
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
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("name").Changed {
		} else {
			return errors.New("Please enter the name flag. ")
		}
		if cmd.Flags().Lookup("email").Changed {
		} else {
			return errors.New("Please enter the email flag. ")
		}
		if cmd.Flags().Lookup("description").Changed {
		} else {
			return errors.New("Please enter the description flag. ")
		}
		if cmd.Flags().Lookup("access-id").Changed {
		} else {
			return errors.New("Please enter the access-id flag. ")
		}
		if cmd.Flags().Lookup("access-key").Changed {
		} else {
			return errors.New("Please enter the access-key flag. ")
		}
		if cmd.Flags().Lookup("regions").Changed {
		} else {
			return errors.New("Please enter the regions flag. ")
		}
		if cmd.Flags().Lookup("secret-key").Changed {
		} else {
			return errors.New("Please enter the secret-key flag. ")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := cli.GetConfig(cmd, true)
		if err != nil {
			return err
		}

		response, err := cli.OnboardCreateAWS(cnf.DefaultWorkspace, cnf.AccessToken, nameAWS, emailAWS, descriptionAWS, accessKeyAWS, accessIdAWS, regionsAWS, secretKey)
		if err != nil {
			return err
		}
		err = cli.PrintOutput(response, outputTypeAWS)
		if err != nil {
			return err
		}
		return nil
	},
}

var AzureCmd = &cobra.Command{
	Use:   "azure",
	Short: "azure command",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("name").Changed {
		} else {
			return errors.New("Please enter the name flag. ")
		}
		if cmd.Flags().Lookup("description").Changed {
		} else {
			return errors.New("Please enter the description flag. ")
		}
		if cmd.Flags().Lookup("client-secret").Changed {
		} else {
			return errors.New("Please enter the client-secret flag. ")
		}
		if cmd.Flags().Lookup("client-id").Changed {
		} else {
			return errors.New("Please enter the client-id flag. ")
		}
		if cmd.Flags().Lookup("subscription-id").Changed {
		} else {
			return errors.New("Please enter the subscription-id flag. ")
		}
		if cmd.Flags().Lookup("object-id").Changed {
		} else {
			return errors.New("Please enter the object-id flag. ")
		}
		if cmd.Flags().Lookup("tenant-id").Changed {
		} else {
			return errors.New("Please enter the tenant-id flag. ")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := cli.GetConfig(cmd, true)
		if err != nil {
			return err
		}

		response, err := cli.OnboardCreateAzure(cnf.DefaultWorkspace, cnf.AccessToken, nameAzure, ObjectId, descriptionAzure, clientIdAzure, clientSecretAzure, subscriptionIdAzure, tenantIdAzure)
		if err != nil {
			return err
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
	credentialCreateCmd.Flags().StringVar(&outputTypeCreate, "output-type", "table", "it is specifying the output type [table , json][optional]")
	credentialCreateCmd.Flags().StringVar(&configCredential, "config", "", "it is specifying the config credential [mandatory].")
	credentialCreateCmd.Flags().StringVar(&nameCredential, "name", "", "it is specifying the name credential [mandatory].")
	credentialCreateCmd.Flags().StringVar(&sourceTypeCredential, "source-type", "", "it is specifying the source type credential [mandatory].")
	credentialCreateCmd.Flags().StringVar(&workspaceNameCreate, "workspace-name", "", "it is specifying the workspaceName [mandatory].")

	// aws flags :
	AwsCmd.Flags().StringVar(&outputTypeAWS, "output-type", "table", "specifying the output type [optional].")
	AwsCmd.Flags().StringVar(&nameAWS, "name", "", "specifying the name for AWS[mandatory]")
	AwsCmd.Flags().StringVar(&emailAWS, "email", "", "specifying the email for AWS[mandatory]")
	AwsCmd.Flags().StringVar(&descriptionAWS, "description", "", "specifying the name for AWS[mandatory]")
	AwsCmd.Flags().StringVar(&accessKeyAWS, "access-id", "", "specifying the accessId for AWS[mandatory]")
	AwsCmd.Flags().StringVar(&accessIdAWS, "access-key", "", "specifying the accessKey for AWS[mandatory]")
	AwsCmd.Flags().StringSliceVar(&regionsAWS, "regions", []string{}, "specifying the regions for AWS[optional]")
	AwsCmd.Flags().StringVar(&secretKey, "secret-key", "", "specifying the secretKey for AWS[mandatory]")
	AwsCmd.Flags().StringVar(&workspaceNameCreate, "workspace-name", "", "it is specifying the workspaceName [mandatory].")

	//	azure flags :
	AzureCmd.Flags().StringVar(&outputTypeAzure, "output-type", "", "specifying the output type [optional].")
	AzureCmd.Flags().StringVar(&nameAzure, "name", "", "specifying the name for AZURE[mandatory]")
	AzureCmd.Flags().StringVar(&descriptionAzure, "description", "", "specifying the description for AZURE[mandatory]")
	AzureCmd.Flags().StringVar(&clientSecretAzure, "client-secret", "", "specifying the clientSecret for AZURE[mandatory]")
	AzureCmd.Flags().StringVar(&clientIdAzure, "client-id", "", "specifying the clientId for AZURE[mandatory]")
	AzureCmd.Flags().StringVar(&secretIdAzure, "secret-id", "", "specifying the secretId for AZURE[mandatory]")
	AzureCmd.Flags().StringVar(&subscriptionIdAzure, "subscription-id", "", "specifying the subscriptionId for AZURE[mandatory]")
	AzureCmd.Flags().StringVar(&tenantIdAzure, "tenant-id", "", "specifying the tenantId for AZURE[mandatory]")
	AzureCmd.Flags().StringVar(&ObjectId, "object-id", "", "specifying the ObjectId for AZURE[mandatory]")
	AzureCmd.Flags().StringVar(&workspaceNameCreate, "workspace-name", "", "it is specifying the workspaceName [mandatory].")

}
