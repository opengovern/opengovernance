package onboard

import (
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/cli"
	_ "gitlab.com/keibiengine/keibi-engine/pkg/onboard"
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
var CountConnections = &cobra.Command{
	Use:   "connections",
	Short: "show the count connections ",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("connectorsNames").Changed {
		} else {
			fmt.Println("please enter connectorsNames flag ")
			log.Fatalln(cmd.Help())
		}
		if cmd.Flags().Lookup("health").Changed {
		} else {
			fmt.Println("please enter health flag ")
			log.Fatalln(cmd.Help())
		}
		if cmd.Flags().Lookup("state").Changed {
		} else {
			fmt.Println("please enter state flag ")
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
			fmt.Println("your access token was expire please login again ")
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
var connectorsNamesCountConnection []string
var healthCountConnection string
var stateCountConnection string

func init() {
	Count.AddCommand(CountConnections)
	CountConnections.Flags().StringSliceVar(&connectorsNamesCountConnection, "connectorsNames", []string{}, "it is use for specifying the connectors names [mandatory] .")
	CountConnections.Flags().StringVar(&healthCountConnection, "health", "", "it is use for specifying the health [mandatory] .")
	CountConnections.Flags().StringVar(&stateCountConnection, "state", "", "it is use for specifying the state [mandatory] .")
}
