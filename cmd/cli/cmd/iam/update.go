package cmd

import (
	"errors"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/cli"
	"github.com/spf13/cobra"
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
		if cmd.Flags().Lookup("id").Changed {
		} else {
			return errors.New("please enter id flag.")
		}
		if cmd.Flags().Lookup("name").Changed {
		} else {
			return errors.New("please enter name flag for name credential.")
		}
		if cmd.Flags().Lookup("config").Changed {
		} else {
			return errors.New("please enter config flag.")
		}
		if cmd.Flags().Lookup("connector").Changed {
		} else {
			return errors.New("please enter connector flag for connector type.")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := cli.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		err = cli.OnboardEditeCredentialById(cnf.DefaultWorkspace, cnf.AccessToken, configUpdate, connectorUpdate, nameUpdate, credentialIdUpdate)
		if err != nil {
			return err
		} else {
			fmt.Println("Your change has been successfully changed.")
			return nil
		}
	},
}
var credentialUpdateEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("id").Changed {
		} else {
			return errors.New("please enter id flag.")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := cli.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		err = cli.OnboardEnableCredential(cnf.DefaultWorkspace, cnf.AccessToken, credentialIdUpdate)
		if err != nil {
			return err
		}
		fmt.Println("credential enabled successfully.")
		return nil
	},
}
var credentialUpdateDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("id").Changed {
		} else {
			return errors.New("please enter id flag.")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := cli.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		err = cli.OnboardDisableCredential(cnf.DefaultWorkspace, cnf.AccessToken, credentialIdUpdate)
		if err != nil {
			return err
		}
		fmt.Println("credential disabled successfully.")
		return nil
	},
}
var sourceUpdateCmd = &cobra.Command{
	Use: "source",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
var SourceUpdateCredentialCmd = &cobra.Command{
	Use:   "credential",
	Short: "it is use for credential source.",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("id").Changed {
		} else {
			return errors.New("please enter id flag.")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := cli.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		err = cli.OnboardPutSourceCredential(cnf.DefaultWorkspace, cnf.AccessToken, sourceIdUpdate)
		if err != nil {
			return err
		}
		fmt.Println("put source credential")
		return nil
	},
}

var SourceUpdateEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "it is use for enable source.",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("id").Changed {
		} else {
			return errors.New("please enter id flag.")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := cli.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		err = cli.OnboardEnableSource(cnf.DefaultWorkspace, cnf.AccessToken, sourceIdUpdate)
		if err != nil {
			return err
		}
		fmt.Println("source enabled.")
		return nil
	},
}
var SourceUpdateDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "it is use for disable source.",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("id").Changed {
		} else {
			return errors.New("please enter id flag.")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := cli.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		err = cli.OnboardDisableSource(cnf.DefaultWorkspace, cnf.AccessToken, sourceIdUpdate)
		if err != nil {
			return err
		}
		fmt.Println("source disabled.")
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

func init() {
	Update.AddCommand(IamUpdate)

	Update.AddCommand(credentialUpdateCmd)
	credentialUpdateCmd.AddCommand(credentialUpdateEnableCmd)
	credentialUpdateCmd.AddCommand(credentialUpdateDisableCmd)

	Update.AddCommand(sourceUpdateCmd)
	sourceUpdateCmd.AddCommand(SourceUpdateDisableCmd)
	sourceUpdateCmd.AddCommand(SourceUpdateEnableCmd)
	sourceUpdateCmd.AddCommand(SourceUpdateCredentialCmd)

	//put source credential flag :
	SourceUpdateDisableCmd.Flags().StringVar(&sourceIdUpdate, "id", "", "it is specifying the source id[mandatory].")
	SourceUpdateDisableCmd.Flags().StringVar(&defaultOutput, "output-type", "table", "it is specifying the output type[table,json][optional]")
	SourceUpdateDisableCmd.Flags().StringVar(&workspacesName, "workspace-name", "", "it is specifying the workspaceName [mandatory].")

	SourceUpdateEnableCmd.Flags().StringVar(&sourceIdUpdate, "id", "", "it is specifying the source id[mandatory].")
	SourceUpdateEnableCmd.Flags().StringVar(&defaultOutput, "output-type", "table", "it is specifying the output type[table,json][optional]")
	SourceUpdateEnableCmd.Flags().StringVar(&workspacesName, "workspace-name", "", "it is specifying the workspaceName [mandatory].")

	SourceUpdateCredentialCmd.Flags().StringVar(&sourceIdUpdate, "id", "", "it is specifying the source id[mandatory].")
	SourceUpdateCredentialCmd.Flags().StringVar(&defaultOutput, "output-type", "table", "it is specifying the output type[table,json][optional]")
	SourceUpdateCredentialCmd.Flags().StringVar(&workspacesName, "workspace-name", "", "it is specifying the workspaceName [mandatory].")

	//	update credential flags :
	credentialUpdateCmd.Flags().StringVar(&configUpdate, "config", "", "it is specifying the config credential[mandatory].")
	credentialUpdateCmd.Flags().StringVar(&nameUpdate, "name", "", "it is specifying the name credential[mandatory].")
	credentialUpdateCmd.Flags().StringVar(&connectorUpdate, "connector", "", "it is specifying the connector credential[mandatory].")
	credentialUpdateCmd.Flags().StringVar(&credentialIdUpdate, "id", "", "it is specifying the credential id[mandatory].")
	credentialUpdateCmd.Flags().StringVar(&workspacesName, "workspace-name", "", "it is specifying the workspaceName [mandatory].")

	credentialUpdateEnableCmd.Flags().StringVar(&workspacesName, "workspace-name", "", "it is specifying the workspaceName [mandatory].")
	credentialUpdateEnableCmd.Flags().StringVar(&credentialIdUpdate, "id", "", "it is specifying the credential id[mandatory].")

	credentialUpdateDisableCmd.Flags().StringVar(&workspacesName, "workspace-name", "", "it is specifying the workspaceName [mandatory].")
	credentialUpdateDisableCmd.Flags().StringVar(&credentialIdUpdate, "id", "", "it is specifying the credential id[mandatory].")

}
