package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/cli"
	apis "gitlab.com/keibiengine/keibi-engine/pkg/cli"
	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"
	"log"
)

var Get = &cobra.Command{
	Use: "get",
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
var healthSourceGet string
var credentialSourceGet string
var sourceIds []string

// onboard command :

var CredentialGetCmd = &cobra.Command{
	Use:   "credential",
	Short: "credential command",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("id").Changed || cmd.Flags().Lookup("health").Changed || cmd.Flags().Lookup("all-available").Changed {
			if cmd.Flags().Lookup("id").Changed {
			} else {
				fmt.Println("please enter the credential id.")
				return cmd.Help()
			}
		} else {
			if cmd.Flags().Lookup("connector").Changed {
			} else {
				fmt.Println("Please enter the name for connectorGet type [AWS or Azure].")
				log.Fatalln(cmd.Help())
			}
			if cmd.Flags().Lookup("health").Changed {
			} else {
				fmt.Println("Please enter the name for health status [healthy,unhealthy,initial_discovery] .")
				log.Fatalln(cmd.Help())
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		if cmd.Flags().Lookup("all-available").Changed {
			response, err := apis.OnboardGetCredentialAvailableConnections(cnf.DefaultWorkspace, cnf.AccessToken, credentialIdGet)
			if err != nil {
				return err
			}
			err = apis.PrintOutputForTypeArray(response, outputTypeGet)
			if err != nil {
				return err
			}
			return nil
		} else if cmd.Flags().Lookup("health").Changed {
			err := apis.OnboardGetLiveCredentialHealth(cnf.DefaultWorkspace, cnf.AccessToken, credentialIdGet)
			if err != nil {
				return err
			}
			fmt.Println("credential is healthy")
			return nil
		} else if cmd.Flags().Lookup("id").Changed {
			response, err := apis.OnboardGetCredentialById(cnf.DefaultWorkspace, cnf.AccessToken, credentialIdGet)
			if err != nil {
				return err
			}
			err = apis.PrintOutputForTypeArray(response, outputTypeGet)
			if err != nil {
				return err
			}
			return nil
		} else {
			response, err := apis.OnboardGetListCredentialsByFilter(cnf.DefaultWorkspace, cnf.AccessToken, connectorTypeGet, healthGet, pageSizeGet, pageNumberGet)
			if err != nil {
				return err
			}
			err = apis.PrintOutputForTypeArray(response, outputTypeGet)
			if err != nil {
				return err
			}
			return nil
		}
	},
}

var CatalogGetCmd = &cobra.Command{
	Use:   "catalog",
	Short: "catalog command",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("metrics").Changed {
		} else {
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
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}

		if cmd.Flags().Lookup("metrics").Changed {
			response, err := apis.OnboardCatalogMetrics(cnf.DefaultWorkspace, cnf.AccessToken)
			if err != nil {
				return err
			}
			err = apis.PrintOutput(response, outputTypeGet)
			if err != nil {
				return err
			}
		} else {
			response, err := apis.OnboardCatalogConnectors(cnf.DefaultWorkspace, cnf.AccessToken, idCatalogGet, miniConnectionCatalogGet, stateCatalogGet, categoryCatalogGet)
			if err != nil {
				return err
			}
			err = apis.PrintOutputForTypeArray(response, outputTypeGet)
			if err != nil {
				return err
			}
		}
		return nil
	},
}

var ConnectorGetCmd = &cobra.Command{
	Use:   "connector",
	Short: "connectorCmd",
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		if cmd.Flags().Lookup("name").Changed {
			response, err := apis.OnboardGetConnector(cnf.DefaultWorkspace, cnf.AccessToken, connectorNameGet)
			if err != nil {
				return err
			}
			err = apis.PrintOutput(response, outputTypeGet)
			if err != nil {
				return err
			}
			return nil
		} else {
			response, err := apis.OnboardGetConnectors(cnf.DefaultWorkspace, cnf.AccessToken)
			if err != nil {
				return err
			}
			err = apis.PrintOutputForTypeArray(response, outputTypeGet)
			if err != nil {
				return err
			}
			return nil
		}
	},
}

var ProvidersCmd = &cobra.Command{
	Use:   "provider",
	Short: "Get providers",
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}

		if cmd.Flags().Lookup("type").Changed {
			response, err := apis.OnboardGetProviderTypes(cnf.DefaultWorkspace, cnf.AccessToken)
			if err != nil {
				return err
			}
			err = apis.PrintOutputForTypeArray(response, outputTypeGet)
			if err != nil {
				return err
			}
		} else {
			response, err := apis.OnboardGetProviders(cnf.DefaultWorkspace, cnf.AccessToken)
			if err != nil {
				return err
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
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		if cmd.Flags().Lookup("Id").Changed {
			response, err := apis.OnboardGetSingleSource(cnf.DefaultWorkspace, cnf.AccessToken, sourceIdGet)
			if err != nil {
				return err
			}
			err = apis.PrintOutput(response, outputTypeGet)
			if err != nil {
				return err
			}
		} else if cmd.Flags().Lookup("credential").Changed {
			body, outputType, err := apis.OnboardGetSourceCredential(cnf.DefaultWorkspace, cnf.AccessToken, sourceIdGet)
			if err != nil {
				return err
			}
			if outputType == "aws" {
				var response api.AWSCredential
				err = json.Unmarshal(body, &response)
				if err != nil {
					return err
				}
				err = apis.PrintOutput(response, outputTypeGet)
				if err != nil {
					return err
				}
			} else {
				var response api.AzureCredential
				err = json.Unmarshal(body, &response)
				if err != nil {
					return err
				}
				err = apis.PrintOutput(response, outputTypeGet)
				if err != nil {
					return err
				}
			}
		} else if cmd.Flags().Lookup("source-ids").Changed {
			response, err := cli.OnboardGetListSourcesFilteredById(cnf.DefaultWorkspace, cnf.AccessToken, sourceIds)
			if err != nil {
				return err
			}
			err = cli.PrintOutputForTypeArray(response, outputTypeGet)
			if err != nil {
				return err
			}
		} else if cmd.Flags().Lookup("health").Changed {
			response, err := apis.OnboardHealthSource(cnf.DefaultWorkspace, cnf.AccessToken, sourceIdGet)
			if err != nil {
				return err
			}
			err = apis.PrintOutput(response, outputTypeGet)
			if err != nil {
				return err
			}
			return nil
		} else if cmd.Flags().Lookup("source-type").Changed {
			response, err := apis.OnboardGetListOfSource(cnf.DefaultWorkspace, cnf.AccessToken)
			if err != nil {
				return err
			}
			err = apis.PrintOutputForTypeArray(response, outputTypeGet)
			if err != nil {
				return err
			}
		} else {
			response, err := apis.OnboardGetListOfSourcesByFilters(cnf.DefaultWorkspace, cnf.AccessToken, connectorTypeGet, pageSizeGet, pageNumberGet)
			if err != nil {
				return err
			}
			err = cli.PrintOutputForTypeArray(response, outputTypeGet)
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
	Get.AddCommand(CatalogGetCmd)
	Get.AddCommand(ConnectorGetCmd)
	Get.AddCommand(CredentialGetCmd)
	Get.AddCommand(GetSourceCmd)
	//get source flag :
	GetSourceCmd.Flags().StringSliceVar(&sourceIds, "ids", []string{}, "It is used to specifying the source ids.")
	GetSourceCmd.Flags().StringVar(&sourceIdGet, "id", "", "It is used to specifying the source id.")
	GetSourceCmd.Flags().StringVar(&healthSourceGet, "health", "", "It is used to check health source.")
	GetSourceCmd.Flags().StringVar(&credentialSourceGet, "credential", "", "It is used to get credential source.")
	GetSourceCmd.Flags().StringVar(&connectorTypeGet, "connector", "", "It is used to filter by connectorGet type [AWS or Azure].")
	GetSourceCmd.Flags().StringVar(&outputTypeGet, "output-type", "table", "Specifies the output type [table , json][optional].")
	GetSourceCmd.Flags().StringVar(&connectorTypeGet, "connector", "", "It is used to filter by connectorGet type [AWS or Azure][mandatory].")
	GetSourceCmd.Flags().StringVar(&pageSizeGet, "page-size", "", "It is used to filter based on page size, default value is 50.")
	GetSourceCmd.Flags().StringVar(&pageNumberGet, "page-number", "", "Used to filter by page number, default value is 1")

	// provider flag :
	ProvidersCmd.Flags().StringVar(&outputTypeGet, "output-type", "table", "Specifies the output type[table , json][optional]")
	ProvidersCmd.Flags().StringVar(&typeProviderGet, "type", "", "Specifies the type for provider.")
	//credential flag :
	CredentialGetCmd.Flags().StringVar(&outputTypeGet, "output-type", "table", "Specifies the output type [table , json][optional]")
	CredentialGetCmd.Flags().StringVar(&credentialIdGet, "id", "", "Used for get a credential by source id.")
	CredentialGetCmd.Flags().StringVar(&allAvailableCredentialGet, "all-available", "", "Used to get all available credential. ")
	CredentialGetCmd.Flags().StringVar(&healthCredentialGet, "health", "", "Get live credential health status.")
	CredentialGetCmd.Flags().StringVar(&connectorTypeGet, "connector", "", "Used to filter by connector type [AWS or Azure][mandatory].")
	CredentialGetCmd.Flags().StringVar(&pageSizeGet, "page-size", "", "Specifies page size for using in filter based on page size, default value is 50.")
	CredentialGetCmd.Flags().StringVar(&pageNumberGet, "page-number", "", "Specifies page number for using in filter base on page number, default value is 1.")
	//catalog flag :
	CatalogGetCmd.Flags().StringVar(&outputTypeGet, "output-type", "table", "Specifies the output type[table , json][optional].")
	CatalogGetCmd.Flags().StringVar(&metricsCatalogGet, "metrics", "", "Specifies the output catalog[metrics , connectors][mandatory].")
	CatalogGetCmd.Flags().StringVar(&categoryCatalogGet, "category", "", "Specifies the category filter[optional].")
	CatalogGetCmd.Flags().StringVar(&stateCatalogGet, "state", "", "Specifies the state filter[optional].")
	CatalogGetCmd.Flags().StringVar(&miniConnectionCatalogGet, "miniConnection", "", "Specifies the minimum connection filter[optional].")
	CatalogGetCmd.Flags().StringVar(&idCatalogGet, "id", "", "Specifies the id filter [optional]")
	//connectorGet flag :
	ConnectorGetCmd.Flags().StringVar(&connectorNameGet, "name", "", "Specifying the connectorGet name [mandatory].")
	ConnectorGetCmd.Flags().StringVar(&outputTypeGet, "output-type", "table", "Specifies the output type[table , json][optional]")

}
