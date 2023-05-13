package onboard

import (
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/cli"
	"log"
	"net/http"
)

var workspaceNameAWS string
var nameAWS string
var emailAWS string
var descriptionAWS string
var accessIdAWS string
var accessKeyAWS string
var regionsAWS []string
var secretKey string
var outputTypeAWS string

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

func init() {
	AwsCmd.Flags().StringVar(&outputTypeAWS, "outputType", "", "specifying the output type [optional].")
	AwsCmd.Flags().StringVar(&nameAWS, "name", "", "specifying the name for AWS[mandatory]")
	AwsCmd.Flags().StringVar(&emailAWS, "email", "", "specifying the email for AWS[mandatory]")
	AwsCmd.Flags().StringVar(&descriptionAWS, "description", "", "specifying the name for AWS[mandatory]")
	AwsCmd.Flags().StringVar(&accessKeyAWS, "accessId", "", "specifying the accessId for AWS[mandatory]")
	AwsCmd.Flags().StringVar(&accessIdAWS, "accessKey", "", "specifying the accessKey for AWS[mandatory]")
	AwsCmd.Flags().StringSliceVar(&regionsAWS, "regions", []string{}, "specifying the regions for AWS[optional]")
	AwsCmd.Flags().StringVar(&secretKey, "secretKey", "", "specifying the secretKey for AWS[mandatory]")
}
