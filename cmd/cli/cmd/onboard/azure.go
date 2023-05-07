package onboard

import (
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/cli"
	"log"
	"net/http"
)

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
		response, status, err := cli.OnboardCreateAzure(accessToken, workspaceNameAzure, nameAzure, ObjectId, descriptionAzure, clientIdAzure, clientSecretAzure, subscriptionIdAzure, tenantIdAzure)
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
	AzureCmd.Flags().StringVar(&outputTypeAzure, "outputType", "", "specifying the output type [optional].")
	AzureCmd.Flags().StringVar(&workspaceNameAzure, "workspaceName", "", "specifying the workspaceName [mandatory]")
	AzureCmd.Flags().StringVar(&nameAzure, "name", "", "specifying the name for AZURE[mandatory]")
	AzureCmd.Flags().StringVar(&descriptionAzure, "description", "", "specifying the description for AZURE[mandatory]")
	AzureCmd.Flags().StringVar(&clientSecretAzure, "clientSecret", "", "specifying the clientSecret for AZURE[mandatory]")
	AzureCmd.Flags().StringVar(&clientIdAzure, "clientId", "", "specifying the clientId for AZURE[mandatory]")
	AzureCmd.Flags().StringVar(&secretIdAzure, "secretId", "", "specifying the secretId for AZURE[mandatory]")
	AzureCmd.Flags().StringVar(&subscriptionIdAzure, "subscriptionId", "", "specifying the subscriptionId for AZURE[mandatory]")
	AzureCmd.Flags().StringVar(&tenantIdAzure, "tenantId", "", "specifying the tenantId for AZURE[mandatory]")
	AzureCmd.Flags().StringVar(&ObjectId, "objectId", "", "specifying the ObjectId for AZURE[mandatory]")

}
