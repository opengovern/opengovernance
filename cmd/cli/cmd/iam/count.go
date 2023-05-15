package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/cli"
	"log"
	"net/http"
)

var Count = &cobra.Command{
	Use:   "count",
	Short: "show number ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return nil
	},
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
			log.Fatalln(cmd.Help())
		}
		if cmd.Flags().Lookup("health").Changed {
		} else {
			fmt.Println("Please enter health flag. ")
			log.Fatalln(cmd.Help())
		}
		if cmd.Flags().Lookup("state").Changed {
		} else {
			fmt.Println("Please enter state flag. ")
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
			fmt.Println("Your access token was expire please login again. ")
			return nil
		}
		response, statusCode, err := cli.OnboardCountConnections(accessToken, connectorsNamesCountConnection, healthCountConnection, stateCountConnection)
		if err != nil {
			return fmt.Errorf("ERROR : status : %v \n error : %v ", statusCode, err)
		}
		if statusCode == http.StatusOK {
			fmt.Printf("OK \n count connections : %v", response)
		}
		return nil
	},
}
var countSourceCmd = &cobra.Command{
	Use:   "source",
	Short: "it will return a count of sources ",
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
			fmt.Println("Your access token was expire please login again. ")
			return nil
		}
		count, statusCode, err := cli.OnboardCountSources(accessToken, connectorCount)
		if err != nil {
			return fmt.Errorf("ERROR: status: %v \n %v ", statusCode, err)
		}
		if statusCode == http.StatusOK {
			fmt.Printf("OK\n%v", count)
		}
		return nil
	},
}
var connectorsNamesCountConnection []string
var healthCountConnection string
var stateCountConnection string
var connectorCount string

func init() {
	Count.AddCommand(CountConnectionsCmd)
	Count.AddCommand(countSourceCmd)
	//count source flag :
	countSourceCmd.Flags().StringVar(&connectorCount, "connector", "", "with it you can filter count with connector.")
	//count connections flag :
	CountConnectionsCmd.Flags().StringSliceVar(&connectorsNamesCountConnection, "connectorsNames", []string{}, "it is use for specifying the connectors names [mandatory] .")
	CountConnectionsCmd.Flags().StringVar(&healthCountConnection, "health", "", "it is use for specifying the health [mandatory] .")
	CountConnectionsCmd.Flags().StringVar(&stateCountConnection, "state", "", "it is use for specifying the state [mandatory] .")
}
