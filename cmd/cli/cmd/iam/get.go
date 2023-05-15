package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/cli"
	apis "gitlab.com/keibiengine/keibi-engine/pkg/cli"
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

// variables onboard :
var sourceIdGet string
var connectorTypeGet string
var healthGet string
var pageSizeGet string
var pageNumberGet string
var credentialIdGet string
var connectorNameGet string
var connectorsCatalogGet string
var metricsCatalogGet string
var categoryCatalogGet string
var stateCatalogGet string
var miniConnectionCatalogGet string
var idCatalogGet string
var outputTypeGet string
var healthCredentialGet string
var allAvailableCredentialGet string
var displayCredentialGet string
var enableCredentialGet string
var typeProviderGet string
var disableSourceGet string
var enableSourceGet string
var healthSourceGet string
var credentialSourceGet string
var sourceIds []string

// onboard command :
var credentialsGetCmd = &cobra.Command{
	Use:   "credentials",
	Short: "it is return a list of credentials",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("connectorGet").Changed {
		} else {
			fmt.Println("Please enter the name for connectorGet type [AWS or Azure].")
			log.Fatalln(cmd.Help())
		}
		if cmd.Flags().Lookup("health").Changed {
		} else {
			fmt.Println("Please enter the name for health status [healthy,unhealthy,initial_discovery] .")
			log.Fatalln(cmd.Help())
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := apis.GetConfig()
		if err != nil {
			return err
		}
		checkEXP, err := apis.CheckExpirationTime(accessToken)
		if err != nil {
			return err
		}
		if checkEXP == true {
			fmt.Println("your access token was expire please login again ")
			return nil
		}
		response, statusCode, err := apis.OnboardGetListCredentialsByFilter(accessToken, connectorTypeGet, healthGet, pageSizeGet, pageNumberGet)
		if err != nil {
			return fmt.Errorf("ERROR : status : %v \n %v", statusCode, err)
		}
		if statusCode == http.StatusOK {
			fmt.Println("OK")
		}
		err = apis.PrintOutputForTypeArray(response, outputTypeGet)
		if err != nil {
			return err
		}
		return nil
	},
}
var credentialCmd = &cobra.Command{
	Use:   "credential",
	Short: "credential command",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("credentialIdGet").Changed {
		} else {
			fmt.Println("please enter the credential id.")
			log.Fatalln(cmd.Help())
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := apis.GetConfig()
		if err != nil {
			return err
		}
		checkEXP, err := apis.CheckExpirationTime(accessToken)
		if err != nil {
			return err
		}
		if checkEXP == true {
			fmt.Println("your access token was expire please login again ")
			return nil
		}
		if cmd.Flags().Lookup("all-available").Changed {
			response, statusCode, err := apis.OnboardGetCredentialAvailableConnections(accessToken, credentialIdGet)
			if err != nil {
				return fmt.Errorf("ERROR[all-available]: status :%v \n %v ", statusCode, err)
			}
			err = apis.PrintOutputForTypeArray(response, outputTypeGet)
			if err != nil {
				return err
			}
			return nil
		} else if cmd.Flags().Lookup("display").Changed {
			statusCode, err := apis.OnboardDisableCredential(accessToken, credentialIdGet)
			if err != nil {
				return fmt.Errorf("ERROR[display]: status: %v \n %v ", statusCode, err)
			}
			if statusCode == http.StatusOK {
				fmt.Println("OK")
				return nil
			}
			return fmt.Errorf("status: %v ", statusCode)
		} else if cmd.Flags().Lookup("enable").Changed {
			statusCode, err := apis.OnboardEnableCredential(accessToken, credentialIdGet)
			if err != nil {
				return fmt.Errorf("ERROR[enable]: status: %v \n %v", statusCode, err)
			}
			if statusCode == http.StatusOK {
				fmt.Println("OK")
				return nil
			}
			return fmt.Errorf("status : %v", err)
		} else if cmd.Flags().Lookup("health").Changed {
			statusCode, err := apis.OnboardGetLiveCredentialHealth(accessToken, credentialIdGet)
			if err != nil {
				return fmt.Errorf("ERROR[health]: status : %v \n %v", statusCode, err)
			}
			if statusCode == http.StatusOK {
				fmt.Println("OK")
			}
			return fmt.Errorf("status: %v", statusCode)
		} else {
			response, statusCode, err := apis.OnboardGetCredentialById(accessToken, credentialIdGet)
			if err != nil {
				return fmt.Errorf("ERROR : status : %v \n %v ", statusCode, err)
			}
			err = apis.PrintOutputForTypeArray(response, outputTypeGet)
			if err != nil {
				return err
			}
			return nil

		}
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
		accessToken, err := apis.GetConfig()
		if err != nil {
			return err
		}
		checkEXP, err := apis.CheckExpirationTime(accessToken)
		if err != nil {
			return err
		}
		if checkEXP == true {
			fmt.Println("your access token was expire please login again ")
			return nil
		}
		if cmd.Flags().Lookup("connectors").Changed {
			response, statusCode, err := apis.OnboardCatalogConnectors(accessToken, idCatalogGet, miniConnectionCatalogGet, stateCatalogGet, categoryCatalogGet)
			if err != nil {
				return fmt.Errorf("ERROR : status : %v \n error : %v ", statusCode, err)
			}
			if statusCode == http.StatusOK {
				fmt.Println("OK")
			}
			if outputTypeGet == "" {
				outputTypeGet = "table"
			}
			err = apis.PrintOutputForTypeArray(response, outputTypeGet)
			if err != nil {
				return err
			}
		}
		if cmd.Flags().Lookup("metrics").Changed {
			response, statusCode, err := apis.OnboardCatalogMetrics(accessToken)
			if err != nil {
				return fmt.Errorf("ERROR : status : %v \n error : %v ", statusCode, err)
			}
			if statusCode == http.StatusOK {
				fmt.Println("OK")
			}
			err = apis.PrintOutput(response, outputTypeGet)
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
		accessToken, err := apis.GetConfig()
		if err != nil {
			return err
		}
		checkEXP, err := apis.CheckExpirationTime(accessToken)
		if err != nil {
			return err
		}
		if checkEXP == true {
			fmt.Println("your access token was expire please login again ")
			return nil
		}
		response, statusCode, err := apis.OnboardGetConnectors(accessToken)
		if err != nil {
			return fmt.Errorf("ERROR : status : %v \n error : %v", statusCode, err)
		}
		if statusCode == http.StatusOK {
			fmt.Println("OK")
		}
		if outputTypeGet == "" {
			outputTypeGet = "table"
		}
		err = apis.PrintOutputForTypeArray(response, outputTypeGet)
		if err != nil {
			return err
		}
		return nil
	},
}
var GetConnectorCmd = &cobra.Command{
	Use:   "connectorGet",
	Short: "connectorCmd",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("name").Changed {
		} else {
			fmt.Println("Please enter the name for connectorGet name .")
			log.Fatalln(cmd.Help())
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := apis.GetConfig()
		if err != nil {
			return err
		}
		checkEXP, err := apis.CheckExpirationTime(accessToken)
		if err != nil {
			return err
		}
		if checkEXP == true {
			fmt.Println("Your access token was expire please login again ")
			return nil
		}
		response, statusCode, err := apis.OnboardGetConnector(accessToken, connectorNameGet)
		if err != nil {
			return fmt.Errorf("ERROR : status : %v \n error : %v ", statusCode, err)
		}
		err = apis.PrintOutput(response, outputTypeGet)
		if err != nil {
			return err
		}
		return nil
	},
}
var ProvidersCmd = &cobra.Command{
	Use:   "provider",
	Short: "Get providers",
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := apis.GetConfig()
		if err != nil {
			return err
		}
		checkEXP, err := apis.CheckExpirationTime(accessToken)
		if err != nil {
			return err
		}
		if checkEXP == true {
			fmt.Println("Your access token was expire please login again ")
			return nil
		}
		if cmd.Flags().Lookup("type").Changed {
			respone, statusCode, err := apis.OnboardGetProviderTypes(accessToken)
			if err != nil {
				return fmt.Errorf("ERROR[providerType]: status: %v \n %v ", statusCode, err)
			}
			err = apis.PrintOutputForTypeArray(respone, outputTypeGet)
			if err != nil {
				return err
			}
		} else {
			response, statusCode, err := apis.OnboardGetProviders(accessToken)
			if err != nil {
				return fmt.Errorf("ERROR[provider]: status: %v \n %v", statusCode, err)
			}
			err = apis.PrintOutputForTypeArray(response, outputTypeGet)
			if err != nil {
				return err
			}
		}
		return nil
	},
}
var GetSourceCmd = &cobra.Command{
	Use:   "source",
	Short: "get a single source ",
	PreRunE: func(cmd *cobra.Command, args []string) error {

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := apis.GetConfig()
		if err != nil {
			return err
		}
		checkEXP, err := apis.CheckExpirationTime(accessToken)
		if err != nil {
			return err
		}
		if checkEXP == true {
			fmt.Println("Your access token was expire please login again ")
			return nil
		}
		if cmd.Flags().Lookup("Id").Changed {
			response, statusCode, err := apis.OnboardGetSingleSource(accessToken, sourceIdGet)
			if err != nil {
				return fmt.Errorf("ERROR[source]: status: %v \n %v", statusCode, err)
			}
			err = apis.PrintOutput(response, outputTypeGet)
			if err != nil {
				return err
			}
		} else if cmd.Flags().Lookup("credential").Changed {

		} else if cmd.Flags().Lookup("ids").Changed {
			response, statusCode, err := cli.OnboardGetSourcesByFilter(accessToken, connectorTypeGet, sourceIds)
			if err != nil {
				return fmt.Errorf("ERROR: status: %v \n %v ", statusCode, err)
			}
			err = cli.PrintOutputForTypeArray(response, outputTypeGet)
			if err != nil {
				return err
			}
		} else if cmd.Flags().Lookup("disable").Changed {
			statusCode, err := apis.OnboardDisableSource(accessToken, sourceIdUpdate)
			if err != nil {
				return fmt.Errorf("ERROR: status: %v \n %v ", statusCode, err)
			}
			if statusCode == http.StatusOK {
				fmt.Println("OK")
			}
		} else if cmd.Flags().Lookup("enable").Changed {
			statusCode, err := apis.OnboardEnableSource(accessToken, sourceIdGet)
			if err != nil {
				return fmt.Errorf("ERROR: status: %v \n %v ", statusCode, err)
			}
			if statusCode == http.StatusOK {
				fmt.Println("OK")
			}
		} else if cmd.Flags().Lookup("health").Changed {
			response, statusCode, err := apis.OnboardHealthSource(accessToken, sourceIdGet)
			if err != nil {
				return fmt.Errorf("ERROR: status: %v \n %v ", statusCode, err)
			}
			err = apis.PrintOutput(response, outputTypeGet)
			if err != nil {
				return err
			}
			return nil
		} else {
			response, statusCode, err := apis.OnboardGetListSources(accessToken, connectorTypeGet)
			if err != nil {
				return fmt.Errorf("ERROR: status : %v \n %v ", statusCode, err)
			}
			err = apis.PrintOutputForTypeArray(response, outputTypeGet)
			if err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	Get.AddCommand(IamGet)
	//	onboard flags :
	Get.AddCommand(catalogCmd)
	Get.AddCommand(GetConnectorCmd)
	Get.AddCommand(GetConnectorsCmd)
	Get.AddCommand(credentialsGetCmd)
	Get.AddCommand(GetSourceCmd)
	//get source flag :
	GetSourceCmd.Flags().StringSliceVar(&sourceIds, "id", []string{}, "it is use for specifying the source ids.")
	GetSourceCmd.Flags().StringVar(&sourceIdGet, "id", "", "it is use for specifying the source id.")
	GetSourceCmd.Flags().StringVar(&disableSourceGet, "disable", "", "it is use for disable source.")
	GetSourceCmd.Flags().StringVar(&enableSourceGet, "enable", "", "it is use for enable source.")
	GetSourceCmd.Flags().StringVar(&healthSourceGet, "health", "", "it is use for check health source.")
	GetSourceCmd.Flags().StringVar(&credentialSourceGet, "credential", "", "it is use for get credential source.")
	GetSourceCmd.Flags().StringVar(&connectorTypeGet, "connector", "", "it is use for filter by connectorGet type [AWS or Azure].")
	GetSourceCmd.Flags().StringVar(&outputTypeGet, "output", "table", "it is specifying the output type [table , json][optional]")

	// provider flag :
	ProvidersCmd.Flags().StringVar(&outputTypeGet, "output", "table", "it is specifying the output type [table , json][optional]")
	ProvidersCmd.Flags().StringVar(&typeProviderGet, "type", "", "it is specifying the type for provider.")
	//credential flag :
	credentialCmd.Flags().StringVar(&outputTypeGet, "output", "table", "it is specifying the output type [table , json][optional]")
	credentialCmd.Flags().StringVar(&credentialIdGet, "credentialIdGet", "", "it is use for get credential ")
	credentialCmd.Flags().StringVar(&allAvailableCredentialGet, "all-available", "", "it is return all available credential ")
	credentialCmd.Flags().StringVar(&displayCredentialGet, "display", "", "it will display the credential ")
	credentialCmd.Flags().StringVar(&enableCredentialGet, "enable", "", "it will enable the credential ")
	credentialCmd.Flags().StringVar(&healthCredentialGet, "health", "", "Get live credential health status")
	//credentials flag :
	credentialsGetCmd.Flags().StringVar(&connectorTypeGet, "connector", "", "it is use for filter by connectorGet type [AWS or Azure].")
	credentialsGetCmd.Flags().StringVar(&healthGet, "health", "", "it is use for filter by health status [healthy,unhealthy,initial_discovery][mandatory] .")
	credentialsGetCmd.Flags().StringVar(&pageSizeGet, "pageSize", "", "it is use for filter by page size,default value is 50 .")
	credentialsGetCmd.Flags().StringVar(&pageNumberGet, "pageNumber", "", "it is use for filter by pageNumber , default value is 1.")
	credentialsGetCmd.Flags().StringVar(&outputTypeGet, "output", "table", "it is specifying the output type [table , json][optional]")

	//catalog flag :
	catalogCmd.Flags().StringVar(&outputTypeGet, "output", "table", "it is specifying the output type [table , json][optional]")
	catalogCmd.Flags().StringVar(&metricsCatalogGet, "metrics", "", "it is specifying the output catalog [metrics , connectors][mandatory]")
	catalogCmd.Flags().StringVar(&connectorsCatalogGet, "connectors", "", "it is specifying the output catalog [metrics , connectors][mandatory]")
	catalogCmd.Flags().StringVar(&categoryCatalogGet, "category", "", "it is specifying the category filter [optional]")
	catalogCmd.Flags().StringVar(&stateCatalogGet, "state", "", "it is specifying the state filter [optional]")
	catalogCmd.Flags().StringVar(&miniConnectionCatalogGet, "miniConnection", "", "it is specifying the minimum connection filter [optional]")
	catalogCmd.Flags().StringVar(&idCatalogGet, "id", "", "it is specifying the id filter [optional]")
	//connectorGet flag :
	GetConnectorCmd.Flags().StringVar(&connectorNameGet, "name", "", "it is specifying the connectorGet name [mandatory].")
	GetConnectorCmd.Flags().StringVar(&outputTypeGet, "output", "table", "it is specifying the output type [table , json][optional]")
	//connectors flag :
	GetConnectorsCmd.Flags().StringVar(&outputTypeGet, "output", "table", "it is specifying the output type [table , json][optional]")

}
