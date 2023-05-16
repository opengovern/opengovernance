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
			//TODO implement ListSourcesByCredentials
			fmt.Println("Please enter what you want to get from the sources.")
			return cmd.Help()
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
	GetSourceCmd.Flags().StringSliceVar(&sourceIds, "ids", []string{}, "it is use for specifying the source ids.")
	GetSourceCmd.Flags().StringVar(&sourceIdGet, "id", "", "it is use for specifying the source id.")
	GetSourceCmd.Flags().StringVar(&healthSourceGet, "health", "", "it is use for check health source.")
	GetSourceCmd.Flags().StringVar(&credentialSourceGet, "credential", "", "it is use for get credential source.")
	GetSourceCmd.Flags().StringVar(&connectorTypeGet, "connector", "", "it is use for filter by connectorGet type [AWS or Azure].")
	GetSourceCmd.Flags().StringVar(&outputTypeGet, "output-type", "table", "it is specifying the output type [table , json][optional]")

	// provider flag :
	ProvidersCmd.Flags().StringVar(&outputTypeGet, "output-type", "table", "it is specifying the output type [table , json][optional]")
	ProvidersCmd.Flags().StringVar(&typeProviderGet, "type", "", "it is specifying the type for provider.")
	//credential flag :
	CredentialGetCmd.Flags().StringVar(&outputTypeGet, "output-type", "table", "it is specifying the output type [table , json][optional]")
	CredentialGetCmd.Flags().StringVar(&credentialIdGet, "id", "", "it is use for get credential ")
	CredentialGetCmd.Flags().StringVar(&allAvailableCredentialGet, "all-available", "", "it is return all available credential ")
	CredentialGetCmd.Flags().StringVar(&displayCredentialGet, "display", "", "it will display the credential ")
	CredentialGetCmd.Flags().StringVar(&enableCredentialGet, "enable", "", "it will enable the credential ")
	CredentialGetCmd.Flags().StringVar(&healthCredentialGet, "health", "", "Get live credential health status")
	CredentialGetCmd.Flags().StringVar(&connectorTypeGet, "connector", "", "it is use for filter by connectorGet type [AWS or Azure].")
	CredentialGetCmd.Flags().StringVar(&pageSizeGet, "page-size", "", "it is use for filter by page size,default value is 50 .")
	CredentialGetCmd.Flags().StringVar(&pageNumberGet, "page-number", "", "it is use for filter by pageNumber , default value is 1.")

	//catalog flag :
	CatalogGetCmd.Flags().StringVar(&outputTypeGet, "output-type", "table", "it is specifying the output type [table , json][optional]")
	CatalogGetCmd.Flags().StringVar(&metricsCatalogGet, "metrics", "", "it is specifying the output catalog [metrics , connectors][mandatory]")
	CatalogGetCmd.Flags().StringVar(&categoryCatalogGet, "category", "", "it is specifying the category filter [optional]")
	CatalogGetCmd.Flags().StringVar(&stateCatalogGet, "state", "", "it is specifying the state filter [optional]")
	CatalogGetCmd.Flags().StringVar(&miniConnectionCatalogGet, "miniConnection", "", "it is specifying the minimum connection filter [optional]")
	CatalogGetCmd.Flags().StringVar(&idCatalogGet, "id", "", "it is specifying the id filter [optional]")
	//connectorGet flag :
	ConnectorGetCmd.Flags().StringVar(&connectorNameGet, "name", "", "it is specifying the connectorGet name [mandatory].")
	ConnectorGetCmd.Flags().StringVar(&outputTypeGet, "output-type", "table", "it is specifying the output type [table , json][optional]")

}
