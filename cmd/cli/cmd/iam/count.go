package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/cli"
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
		if cmd.Flags().Lookup("connectorsNames").Changed {
		} else {
			fmt.Println("Please enter connectorsNames flag. ")
			return cmd.Help()
		}
		if cmd.Flags().Lookup("health").Changed {
		} else {
			fmt.Println("Please enter health flag. ")
			return cmd.Help()
		}
		if cmd.Flags().Lookup("state").Changed {
		} else {
			fmt.Println("Please enter state flag. ")
			return cmd.Help()
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
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := cli.GetConfig(cmd, true)
		if err != nil {
			return err
		}

		count, err := cli.OnboardCountSources(cnf.DefaultWorkspace, cnf.AccessToken, connectorCount)
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
var connectorCount string
var workspaceNameCount string

func init() {
	Count.AddCommand(CountConnectionsCmd)
	Count.AddCommand(countSourceCmd)
	//count source flag :
	countSourceCmd.Flags().StringVar(&connectorCount, "connector", "", "with it you can filter count with connector type [AWS ,Azure][optional].")
	countSourceCmd.Flags().StringVar(&workspaceNameCount, "workspace-name", "", "it is specifying the workspaceName [mandatory].")

	//count connections flag :
	CountConnectionsCmd.Flags().StringSliceVar(&connectorsNamesCountConnection, "connectorsNames", []string{}, "it is use for specifying the connectors names [mandatory] .")
	CountConnectionsCmd.Flags().StringVar(&healthCountConnection, "health", "", "it is use for specifying the health [mandatory] .")
	CountConnectionsCmd.Flags().StringVar(&stateCountConnection, "state", "", "it is use for specifying the state [mandatory] .")
	CountConnectionsCmd.Flags().StringVar(&workspaceNameCount, "workspace-name", "", "it is specifying the workspaceName [mandatory].")

}
