package cmd

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/kaytu-io/kaytu-engine/pkg/cli"
)

var Count = &cobra.Command{
	Use: "count",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
var CountConnectionsCmd = &cobra.Command{
	Use:   "connections",
	Short: "show the count connections ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("connector-type").Changed {
		} else {
			return errors.New("Please enter connectorsNames flag. ")
		}
		if cmd.Flags().Lookup("health").Changed {
		} else {
			return errors.New("Please enter health flag. ")
		}
		if cmd.Flags().Lookup("state").Changed {
		} else {
			return errors.New("Please enter state flag. ")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := cli.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		response, err := cli.OnboardCountConnections(cnf.DefaultWorkspace, cnf.AccessToken, connectorsNamesCountConnection, healthCountConnection, stateCountConnection)
		if err != nil {
			return err
		}
		fmt.Printf("count connections : %v", response)
		return nil
	},
}
var countSourceCmd = &cobra.Command{
	Use:   "source",
	Short: "it will return a count of sources ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("count-type").Changed {
		} else {
			return errors.New("please enter count-type flag")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := cli.GetConfig(cmd, true)
		if err != nil {
			return err
		}

		count, err := cli.OnboardCountSources(cnf.DefaultWorkspace, cnf.AccessToken, connectorType)
		if err != nil {
			return err
		}
		fmt.Printf("count source : %v", count)
		return nil
	},
}
var connectorsNamesCountConnection []string
var healthCountConnection string
var stateCountConnection string
var connectorType string
var workspaceNameCount string

func init() {
	Count.AddCommand(CountConnectionsCmd)
	Count.AddCommand(countSourceCmd)

	//count source flag :
	countSourceCmd.Flags().StringVar(&connectorType, "connector-type", "", "with it you can filter count with connector type [AWS ,Azure][optional].")
	countSourceCmd.Flags().StringVar(&workspaceNameCount, "workspace-name", "", "it is specifying the workspaceName [mandatory].")

	//count connections flag :
	CountConnectionsCmd.Flags().StringSliceVar(&connectorsNamesCountConnection, "connector-type", []string{}, "it is use for specifying the connectors names [mandatory] .")
	CountConnectionsCmd.Flags().StringVar(&healthCountConnection, "health", "", "it is use for specifying the health [mandatory] .")
	CountConnectionsCmd.Flags().StringVar(&stateCountConnection, "state", "", "it is use for specifying the state [mandatory] .")
	CountConnectionsCmd.Flags().StringVar(&workspaceNameCount, "workspace-name", "", "it is specifying the workspaceName [mandatory].")

}
