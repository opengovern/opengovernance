package onboard

import (
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/cli"
	"log"
	"net/http"
)

var Get = &cobra.Command{
	Use:   "get",
	Short: "get command",
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
var catalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "catalogCmd",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("metrics").Changed {

		} else if cmd.Flags().Lookup("connectors").Changed {
			if cmd.Flags().Lookup("category").Changed {
			} else {
				fmt.Println("please enter the id flag category ")
				log.Fatalln(cmd.Help())
			}
			if cmd.Flags().Lookup("state").Changed {
			} else {
				fmt.Println("please enter the id flag state ")
				log.Fatalln(cmd.Help())
			}
			if cmd.Flags().Lookup("miniConnection").Changed {
			} else {
				fmt.Println("please enter the id flag miniConnection ")
				log.Fatalln(cmd.Help())
			}
			if cmd.Flags().Lookup("id").Changed {
			} else {
				fmt.Println("please enter the id flag id ")
				log.Fatalln(cmd.Help())
			}
		} else {
			fmt.Println("please enter the output catalog ")
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
		if cmd.Flags().Lookup("connectors").Changed {
			response, statusCode, err := cli.OnboardCatalogConnectors(accessToken, idCatalog, miniConnectionCatalog, stateCatalog, categoryCatalog)
			if err != nil {
				return fmt.Errorf("ERROR : status : %v \n error : %v ", statusCode, err)
			}
			if statusCode == http.StatusOK {
				fmt.Println("OK")
			}
			if outputType == "" {
				outputType = "table"
			}
			err = cli.PrintOutputForTypeArray(response, outputType)
			if err != nil {
				return err
			}
		}
		if cmd.Flags().Lookup("metrics").Changed {
			response, statusCode, err := cli.OnboardCatalogMetrics(accessToken)
			if err != nil {
				return fmt.Errorf("ERROR : status : %v \n error : %v ", statusCode, err)
			}
			if statusCode == http.StatusOK {
				fmt.Println("OK")
			}
			err = cli.PrintOutput(response, outputType)
			if err != nil {
				return err
			}
		}
		return nil
	},
}
var GetConnectorsCmd = &cobra.Command{
	Use:   "connectors",
	Short: "connectors command ",
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
		response, statusCode, err := cli.OnboardGetConnectors(accessToken)
		if err != nil {
			return fmt.Errorf("ERROR : status : %v \n error : %v", statusCode, err)
		}
		if statusCode == http.StatusOK {
			fmt.Println("OK")
		}
		if outputType == "" {
			outputType = "table"
		}
		err = cli.PrintOutputForTypeArray(response, outputType)
		if err != nil {
			return err
		}
		return nil
	},
}
var GetConnectorCmd = &cobra.Command{
	Use:   "connector",
	Short: "connectorCmd",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("name").Changed {
		} else {
			fmt.Println("please enter the name for connector name .")
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
		response, statusCode, err := cli.OnboardGetConnector(accessToken, connectorName)
		if err != nil {
			return fmt.Errorf("ERROR : status : %v \n error : %v ", statusCode, err)
		}
		err = cli.PrintOutput(response, outputType)
		if err != nil {
			return err
		}
		return nil
	},
}
var credentialsGetCmd = cobra.Command{
	Use:   "credential",
	Short: "credential command",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("connector").Changed {
		} else {
			fmt.Println("please enter the name for connector type [AWS or Azure].")
			log.Fatalln(cmd.Help())
		}
		if cmd.Flags().Lookup("health").Changed {
		} else {
			fmt.Println("please enter the name for health status [healthy,unhealthy,initial_discovery] .")
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
		response, statusCode, err := cli.OnboardListCredentials(accessToken, connectorTypeGet, healthGet, pageSizeGet, pageNumberGet)
		if err != nil {
			return fmt.Errorf("ERROR : status : %v \n %v", statusCode, err)
		}
		if statusCode == http.StatusOK {
			fmt.Println("OK")
		}
		err = cli.PrintOutputForTypeArray(response, outputType)
		if err != nil {
			return err
		}
		return nil
	},
}
var connectorTypeGet string
var healthGet string
var pageSizeGet string
var pageNumberGet string

var connectorName string
var connectorsCatalog string
var metricsCatalog string
var categoryCatalog string
var stateCatalog string
var miniConnectionCatalog string
var idCatalog string
var outputType string

func init() {
	Get.AddCommand(catalogCmd)
	Get.AddCommand(GetConnectorCmd)
	Get.AddCommand(GetConnectorsCmd)
	Get.AddCommand(&credentialsGetCmd)
	//credential flag :
	credentialsGetCmd.Flags().StringVar(&connectorTypeGet, "connector", "", "it is use for filter by connector type [AWS or Azure][mandatory].")
	credentialsGetCmd.Flags().StringVar(&healthGet, "health", "", "it is use for filter by health status [healthy,unhealthy,initial_discovery][mandatory] .")
	credentialsGetCmd.Flags().StringVar(&pageSizeGet, "pageSize", "", "it is use for filter by page size,default value is 50 .")
	credentialsGetCmd.Flags().StringVar(&pageNumberGet, "pageNumber", "", "it is use for filter by pageNumber , default value is 1.")
	credentialsGetCmd.Flags().StringVar(&outputType, "output", "", "it is specifying the output type [table , json][optional]")

	//catalog flag :
	catalogCmd.Flags().StringVar(&outputType, "output", "", "it is specifying the output type [table , json][optional]")
	catalogCmd.Flags().StringVar(&metricsCatalog, "metrics", "", "it is specifying the output catalog [metrics , connectors][mandatory]")
	catalogCmd.Flags().StringVar(&connectorsCatalog, "connectors", "", "it is specifying the output catalog [metrics , connectors][mandatory]")
	catalogCmd.Flags().StringVar(&categoryCatalog, "category", "", "it is specifying the category filter [optional]")
	catalogCmd.Flags().StringVar(&stateCatalog, "state", "", "it is specifying the state filter [optional]")
	catalogCmd.Flags().StringVar(&miniConnectionCatalog, "miniConnection", "", "it is specifying the minimum connection filter [optional]")
	catalogCmd.Flags().StringVar(&idCatalog, "id", "", "it is specifying the id filter [optional]")
	//connector flag :
	GetConnectorCmd.Flags().StringVar(&connectorName, "name", "", "it is specifying the connector name [mandatory].")
	GetConnectorCmd.Flags().StringVar(&outputType, "output", "", "it is specifying the output type [table , json][optional]")
	//connectors flag :
	GetConnectorsCmd.Flags().StringVar(&outputType, "output", "", "it is specifying the output type [table , json][optional]")

}
