package onboard

import (
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/cli"
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

var editeCredentialCmd = cobra.Command{
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
		cli.OnboardEditeCredentialById(accessToken)
		return nil
	},
}
var configUpdate string
var credentialIdUpdate string
var nameUpdate string
var connectorUpdate string

func init() {
	Update.AddCommand()
	//	update credential flags :
	editeCredentialCmd.Flags().StringVar()
}
