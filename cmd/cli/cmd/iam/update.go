package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/cli"
)

var Update = &cobra.Command{
	Use: "update",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var credentialUpdateCmd = &cobra.Command{
	Use: "credential",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("disable").Changed {
			if cmd.Flags().Lookup("id").Changed {
			} else {
				fmt.Println("please enter the credential id.")
				return cmd.Help()
			}
		} else if cmd.Flags().Lookup("enable").Changed {
			if cmd.Flags().Lookup("id").Changed {
			} else {
				fmt.Println("please enter the credential id.")
				return cmd.Help()
			}
		} else {
			if cmd.Flags().Lookup("id").Changed {
			} else {
				fmt.Println("please enter id flag.")
				return cmd.Help()
			}
			if cmd.Flags().Lookup("name").Changed {
			} else {
				fmt.Println("please enter name flag for name credential.")
				return cmd.Help()
			}
			if cmd.Flags().Lookup("config").Changed {
			} else {
				fmt.Println("please enter config flag.")
				return cmd.Help()
			}
			if cmd.Flags().Lookup("connector").Changed {
			} else {
				fmt.Println("please enter connector flag for connector type.")
				return cmd.Help()
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := cli.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		if cmd.Flags().Lookup("disable").Changed {
			err := cli.OnboardDisableCredential(cnf.DefaultWorkspace, cnf.AccessToken, credentialIdGet)
			if err != nil {
				return err
			}
			fmt.Println("credential disabled successfully.")
			return nil
		} else if cmd.Flags().Lookup("enable").Changed {
			err := cli.OnboardEnableCredential(cnf.DefaultWorkspace, cnf.AccessToken, credentialIdGet)
			if err != nil {
				return err
			}
			fmt.Println("credential enabled successfully.")
			return nil
		} else {
			err = cli.OnboardEditeCredentialById(cnf.DefaultWorkspace, cnf.AccessToken, configUpdate, connectorUpdate, nameUpdate, credentialIdUpdate)
			if err != nil {
				return err
			} else {
				fmt.Println("Your change has been successfully changed.")
				return nil
			}
		}
	},
}
var sourceUpdateCmd = &cobra.Command{
	Use: "source",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("enable").Changed {

		} else if cmd.Flags().Lookup("disable").Changed {

		} else {
			if cmd.Flags().Lookup("id").Changed {
			} else {
				fmt.Println("please enter workspace name flag.")
				return nil
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := cli.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		if cmd.Flags().Lookup("enable").Changed {
			err := cli.OnboardEnableSource(cnf.DefaultWorkspace, cnf.AccessToken, sourceIdGet)
			if err != nil {
				return err
			}
			fmt.Println("source enabled.")
			return nil
		} else if cmd.Flags().Lookup("disable").Changed {
			err = cli.OnboardDisableSource(cnf.DefaultWorkspace, cnf.AccessToken, sourceIdUpdate)
			if err != nil {
				return err
			}
			fmt.Println("source disabled.")
			return nil
		} else if cmd.Flags().Lookup("credential").Changed {
			err := cli.OnboardPutSourceCredential(cnf.DefaultWorkspace, cnf.AccessToken, sourceIdUpdate)
			if err != nil {
				return err
			}
			fmt.Println("put source credential")
			return nil
		}
		return nil
	},
}
var sourceIdUpdate string
var configUpdate string
var credentialIdUpdate string
var nameUpdate string
var connectorUpdate string
var workspacesName string
var defaultOutput string
var credentialSourceUpdate string
var enableSourceUpdate string
var disableSourceUpdate string

func init() {
	Update.AddCommand(IamUpdate)
	Update.AddCommand(credentialUpdateCmd)
	Update.AddCommand(credentialUpdateCmd)

	//put source credential flag :
	sourceUpdateCmd.Flags().StringVar(&sourceIdUpdate, "id", "", "it is specifying the source id[mandatory].")
	sourceUpdateCmd.Flags().StringVar(&defaultOutput, "output-type", "table", "it is specifying the output type[table,json][optional]")
	sourceUpdateCmd.Flags().StringVar(&workspacesName, "workspace-name", "", "it is specifying the workspaceName [mandatory].")
	sourceUpdateCmd.Flags().StringVar(&disableSourceUpdate, "disable", "", "it is use for disable source.")
	sourceUpdateCmd.Flags().StringVar(&enableSourceUpdate, "enable", "", "it is use for enable source.")
	sourceUpdateCmd.Flags().StringVar(&credentialSourceUpdate, "credential", "", "it is use for credential source.")

	//	update credential flags :
	credentialUpdateCmd.Flags().StringVar(&configUpdate, "config", "", "it is specifying the config credential[mandatory].")
	credentialUpdateCmd.Flags().StringVar(&nameUpdate, "name", "", "it is specifying the name credential[mandatory].")
	credentialUpdateCmd.Flags().StringVar(&connectorUpdate, "connector", "", "it is specifying the connector credential[mandatory].")
	credentialUpdateCmd.Flags().StringVar(&credentialIdUpdate, "id", "", "it is specifying the credential id[mandatory].")
	credentialUpdateCmd.Flags().StringVar(&workspacesName, "workspace-name", "", "it is specifying the workspaceName [mandatory].")

}
