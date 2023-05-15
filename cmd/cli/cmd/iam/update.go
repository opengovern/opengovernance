package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/cli"
	"net/http"
)

var Update = &cobra.Command{
	Use:   "update",
	Short: "it is use for update user or key user  ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return nil
	},
}

var editeCredentialCmd = &cobra.Command{
	Use:   "credential",
	Short: "it will update credential by id",
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
		statusCode, err := cli.OnboardEditeCredentialById(accessToken, configUpdate, connectorUpdate, nameUpdate, credentialIdUpdate)
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
var PutSourceCredentialCmd = &cobra.Command{
	Use:   "source",
	Short: "it will update source ",
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := cli.GetConfig()
		if err != nil {
			return err
		}

		aws ,azure ,statusCode, err := cli.OnboardPutSourceCredential(accessToken, sourceIdUpdate)
		if err != nil || statusCode != http.StatusOK {
			return fmt.Errorf("ERROR: status: %v \n %v ", statusCode, err)
		}
		if statusCode == http.StatusOK {
			fmt.Println("OK")
		}
		if aws == {

		}
		return nil
	},
}
var sourceIdUpdate string
var configUpdate string
var credentialIdUpdate string
var nameUpdate string
var connectorUpdate string

func init() {
	Update.AddCommand(IamUpdate)
	Update.AddCommand(editeCredentialCmd)
	Update.AddCommand(editeCredentialCmd)

	//put source credential flag :
	PutSourceCredentialCmd.Flags().StringVar(&sourceIdUpdate, "id", "", "it is specifying the source id.")

	//	update credential flags :
	editeCredentialCmd.Flags().StringVar(&configUpdate, "config", "", "it is specifying the config credential[mandatory].")
	editeCredentialCmd.Flags().StringVar(&nameUpdate, "name", "", "it is specifying the name credential[mandatory].")
	editeCredentialCmd.Flags().StringVar(&connectorUpdate, "connector", "", "it is specifying the connector credential[mandatory].")
	editeCredentialCmd.Flags().StringVar(&credentialIdUpdate, "credentialIdGet", "", "it is specifying the credential id[mandatory].")

}

