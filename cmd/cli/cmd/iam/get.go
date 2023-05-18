package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/cli"
	apis "gitlab.com/keibiengine/keibi-engine/pkg/cli"
	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"
)

var Get = &cobra.Command{
	Use: "get",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// variables onboard :
var workspaceNameGet string
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
var healthSourceGet string
var sourceIds []string
var sourceIdOption string
var sourceTypeOption string

// onboard command :

var GetCredentialsCmd = &cobra.Command{
	Use:   "credential",
	Short: "credential command",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("connector").Changed {
		} else {
			return errors.New("Please enter the name for connectorGet type [AWS or Azure]. ")
		}
		if cmd.Flags().Lookup("health").Changed {
		} else {
			return errors.New("Please enter the name for health status [healthy,unhealthy,initial_discovery]. ")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		response, err := apis.OnboardGetListCredentialsByFilter(cnf.DefaultWorkspace, cnf.AccessToken, connectorTypeGet, healthGet, pageSizeGet, pageNumberGet)
		if err != nil {
			return err
		}
		err = apis.PrintOutputForTypeArray(response, outputTypeGet)
		if err != nil {
			return err
		}
		return nil
	},
}

var GetCredentialAllAvailable = &cobra.Command{
	Use:   "all-available",
	Short: "Used to get all available credential.",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("id").Changed {
		} else {
			return errors.New("please enter the credential id. ")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		response, err := apis.OnboardGetCredentialAvailableConnections(cnf.DefaultWorkspace, cnf.AccessToken, credentialIdGet)
		if err != nil {
			return err
		}
		err = apis.PrintOutputForTypeArray(response, outputTypeGet)
		if err != nil {
			return err
		}
		return nil
	},
}

var GetCredentialByIdCmd = &cobra.Command{
	Use:   "id",
	Short: "Used for get a credential by source id.",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("id").Changed {
		} else {
			return errors.New("please enter the credential id. ")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		response, err := apis.OnboardGetCredentialById(cnf.DefaultWorkspace, cnf.AccessToken, credentialIdGet)
		if err != nil {
			return err
		}
		err = apis.PrintOutputForTypeArray(response, outputTypeGet)
		if err != nil {
			return err
		}
		return nil
	},
}

var credentialHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Get live credential health status.",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("id").Changed {
		} else {
			return errors.New("please enter the credential id. ")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		err = apis.OnboardGetLiveCredentialHealth(cnf.DefaultWorkspace, cnf.AccessToken, credentialIdGet)
		if err != nil {
			return err
		}
		fmt.Println("credential is healthy")
		return nil
	},
}

var CatalogGetCmd = &cobra.Command{
	Use:   "catalog",
	Short: "catalog command",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("category").Changed {
		} else {
			return errors.New("please enter the id flag category. ")
		}
		if cmd.Flags().Lookup("state").Changed {
		} else {
			return errors.New("please enter the id flag state. ")
		}
		if cmd.Flags().Lookup("miniConnection").Changed {
		} else {
			return errors.New("please enter the id flag miniConnection. ")
		}
		if cmd.Flags().Lookup("id").Changed {
		} else {
			return errors.New("please enter the id flag id. ")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		response, err := apis.OnboardCatalogConnectors(cnf.DefaultWorkspace, cnf.AccessToken, idCatalogGet, miniConnectionCatalogGet, stateCatalogGet, categoryCatalogGet)
		if err != nil {
			return err
		}
		err = apis.PrintOutputForTypeArray(response, outputTypeGet)
		if err != nil {
			return err
		}
		return nil
	},
}

var CatalogMetricsCmd = &cobra.Command{
	Use:   "metric",
	Short: "Returns the list of metrics for catalog page.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		response, err := apis.OnboardCatalogMetrics(cnf.DefaultWorkspace, cnf.AccessToken)
		if err != nil {
			return err
		}
		err = apis.PrintOutput(response, outputTypeGet)
		if err != nil {
			return err
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
		response, err := apis.OnboardGetConnectors(cnf.DefaultWorkspace, cnf.AccessToken)
		if err != nil {
			return err
		}
		err = apis.PrintOutputForTypeArray(response, outputTypeGet)
		if err != nil {
			return err
		}
		return nil
	},
}

var ConnectorNameCmd = &cobra.Command{
	Use:   "connector-name",
	Short: "This is the return connector whose name is entered",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("name").Changed {
		} else {
			return errors.New("Please enter the name for connector name. ")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		response, err := apis.OnboardGetConnector(cnf.DefaultWorkspace, cnf.AccessToken, connectorNameGet)
		if err != nil {
			return err
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
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		response, err := apis.OnboardGetProviders(cnf.DefaultWorkspace, cnf.AccessToken)
		if err != nil {
			return err
		}
		err = apis.PrintOutputForTypeArray(response, outputTypeGet)
		if err != nil {
			return err
		}
		return nil
	},
}

var providersTypeCmd = &cobra.Command{
	Use:   "type",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		response, err := apis.OnboardGetProviderTypes(cnf.DefaultWorkspace, cnf.AccessToken)
		if err != nil {
			return err
		}
		err = apis.PrintOutputForTypeArray(response, outputTypeGet)
		if err != nil {
			return err
		}
		return nil
	},
}

var GetSourceCmd = &cobra.Command{
	Use:   "source",
	Short: "return a list of sources and you can use from flags for filter it ",
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		response, err := apis.OnboardGetListOfSourcesByFilters(cnf.DefaultWorkspace, cnf.AccessToken, connectorTypeGet, pageSizeGet, pageNumberGet)
		if err != nil {
			return err
		}
		err = cli.PrintOutputForTypeArray(response, outputTypeGet)
		if err != nil {
			return err
		}
		return nil
	},
}

var GetSourceById = &cobra.Command{
	Use:   "id",
	Short: "",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		response, err := apis.OnboardGetSingleSource(cnf.DefaultWorkspace, cnf.AccessToken, sourceIdGet)
		if err != nil {
			return err
		}
		err = apis.PrintOutput(response, outputTypeGet)
		if err != nil {
			return err
		}
		return nil
	},
}

var GetSourceCredentialCmd = &cobra.Command{
	Use:   "credential",
	Short: "It is used to get credential source",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}
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
		return nil
	},
}

var GetSourceIdCmd = &cobra.Command{
	Use:   "source-ids",
	Short: "It is used to get sources by filter source ids.",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("source-id").Changed {
			return errors.New("please enter id for source id. ")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		response, err := cli.OnboardGetListSourcesFilteredById(cnf.DefaultWorkspace, cnf.AccessToken, sourceIds)
		if err != nil {
			return err
		}
		err = cli.PrintOutputForTypeArray(response, outputTypeGet)
		if err != nil {
			return err
		}
		return nil
	},
}

var GetSourceTypeCmd = &cobra.Command{
	Use:   "source-type",
	Short: "It is used to get sources by filter source type .",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("source-type").Changed {
			return errors.New("please enter source-type flag[AWS or Azure]. ")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		response, err := apis.OnboardGetListOfSource(cnf.DefaultWorkspace, cnf.AccessToken)
		if err != nil {
			return err
		}
		err = apis.PrintOutputForTypeArray(response, outputTypeGet)
		if err != nil {
			return err
		}
		return nil
	},
}

var GetSourceHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Lookup("id").Changed {
		} else {
			fmt.Println("please enter id flag for source id .")
			return cmd.Help()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cnf, err := apis.GetConfig(cmd, true)
		if err != nil {
			return err
		}
		response, err := apis.OnboardHealthSource(cnf.DefaultWorkspace, cnf.AccessToken, sourceIdGet)
		if err != nil {
			return err
		}
		err = apis.PrintOutput(response, outputTypeGet)
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	Get.AddCommand(IamGet)
	//	onboard flags :
	Get.AddCommand(CatalogGetCmd)
	Get.AddCommand(CatalogMetricsCmd)

	Get.AddCommand(ConnectorGetCmd)
	Get.AddCommand(ConnectorNameCmd)

	Get.AddCommand(ProvidersCmd)
	ProvidersCmd.AddCommand(providersTypeCmd)

	Get.AddCommand(GetCredentialsCmd)
	GetCredentialsCmd.AddCommand(credentialHealthCmd)
	GetCredentialsCmd.AddCommand(GetCredentialByIdCmd)
	GetCredentialsCmd.AddCommand(GetCredentialAllAvailable)

	Get.AddCommand(GetSourceCmd)
	GetSourceCmd.AddCommand(GetSourceHealthCmd)
	GetSourceCmd.AddCommand(GetSourceCredentialCmd)
	GetSourceCmd.AddCommand(GetSourceTypeCmd)
	GetSourceCmd.AddCommand(GetSourceIdCmd)
	GetSourceCmd.AddCommand(GetSourceById)

	//get source flag :
	GetSourceCmd.Flags().StringSliceVar(&sourceIds, "ids", []string{}, "It is used to specifying the source ids.")
	GetSourceCmd.Flags().StringVar(&sourceIdGet, "id", "", "It is used to specifying the source id.")
	GetSourceCmd.Flags().StringVar(&healthSourceGet, "health", "", "It is used to check health source.")
	GetSourceCmd.Flags().StringVar(&connectorTypeGet, "connector", "", "It is used to filter by connectorGet type [AWS or Azure].")
	GetSourceCmd.Flags().StringVar(&outputTypeGet, "output-type", "table", "Specifies the output type [table , json][optional].")
	GetSourceCmd.Flags().StringVar(&pageSizeGet, "page-size", "", "It is used to filter based on page size, default value is 50.")
	GetSourceCmd.Flags().StringVar(&pageNumberGet, "page-number", "", "Used to filter by page number, default value is 1")
	GetSourceCmd.Flags().StringVar(&sourceIdOption, "source-id", "", "It is used to get sources by filter source ids.")
	GetSourceCmd.Flags().StringVar(&sourceTypeOption, "source-type", "", "It is used to get sources by filter source type .")
	GetSourceCmd.Flags().StringVar(&workspaceNameGet, "workspace-name", "", "specifies the workspace name[mandatory].")

	// provider flag :
	ProvidersCmd.Flags().StringVar(&outputTypeGet, "output-type", "table", "Specifies the output type[table , json][optional]")
	providersTypeCmd.Flags().StringVar(&outputTypeGet, "output-type", "table", "Specifies the output type[table , json][optional]")
	providersTypeCmd.Flags().StringVar(&workspaceNameGet, "workspace-name", "", "specifies the workspace name[mandatory].")

	//credential flag :
	GetCredentialsCmd.Flags().StringVar(&outputTypeGet, "output-type", "table", "Specifies the output type [table , json][optional]")
	GetCredentialsCmd.Flags().StringVar(&healthCredentialGet, "health", "", "specifying the type healthy[healthy,unhealthy,initial_discovery].")
	GetCredentialsCmd.Flags().StringVar(&connectorTypeGet, "connector", "", "Used to filter by connector type [AWS or Azure][mandatory].")
	GetCredentialsCmd.Flags().StringVar(&pageSizeGet, "page-size", "", "Specifies page size for using in filter based on page size, default value is 50.")
	GetCredentialsCmd.Flags().StringVar(&pageNumberGet, "page-number", "", "Specifies page number for using in filter base on page number, default value is 1.")
	GetCredentialsCmd.Flags().StringVar(&workspaceNameGet, "workspace-name", "", "specifies the workspace name[mandatory].")

	credentialHealthCmd.Flags().StringVar(&credentialIdGet, "id", "", "Used to specifying the credential id.")
	credentialHealthCmd.Flags().StringVar(&outputTypeGet, "output-type", "table", "Specifies the output type [table , json][optional]")
	credentialHealthCmd.Flags().StringVar(&workspaceNameGet, "workspace-name", "", "specifies the workspace name[mandatory].")

	GetCredentialByIdCmd.Flags().StringVar(&credentialIdGet, "id", "", "Used to specifying the credential id.")
	GetCredentialByIdCmd.Flags().StringVar(&outputTypeGet, "output-type", "table", "Specifies the output type [table , json][optional]")
	GetCredentialByIdCmd.Flags().StringVar(&workspaceNameGet, "workspace-name", "", "specifies the workspace name[mandatory].")

	GetCredentialAllAvailable.Flags().StringVar(&credentialIdGet, "id", "", "Used to specifying the credential id.")
	GetCredentialAllAvailable.Flags().StringVar(&outputTypeGet, "output-type", "table", "Specifies the output type [table , json][optional]")
	GetCredentialAllAvailable.Flags().StringVar(&workspaceNameGet, "workspace-name", "", "specifies the workspace name[mandatory].")

	//catalog flag :
	CatalogGetCmd.Flags().StringVar(&outputTypeGet, "output-type", "table", "Specifies the output type[table , json][optional].")
	CatalogGetCmd.Flags().StringVar(&metricsCatalogGet, "metrics", "", "Specifies the output catalog[metrics , connectors][mandatory].")
	CatalogGetCmd.Flags().StringVar(&categoryCatalogGet, "category", "", "Specifies the category filter[optional].")
	CatalogGetCmd.Flags().StringVar(&stateCatalogGet, "state", "", "Specifies the state filter[optional].")
	CatalogGetCmd.Flags().StringVar(&miniConnectionCatalogGet, "miniConnection", "", "Specifies the minimum connection filter[optional].")
	CatalogGetCmd.Flags().StringVar(&idCatalogGet, "id", "", "Specifies the id filter [optional]")
	CatalogGetCmd.Flags().StringVar(&workspaceNameGet, "workspace-name", "", "specifies the workspace name[mandatory].")

	CatalogMetricsCmd.Flags().StringVar(&outputTypeGet, "output-type", "table", "Specifies the output type[table , json][optional].")
	CatalogMetricsCmd.Flags().StringVar(&workspaceNameGet, "workspace-name", "", "specifies the workspace name[mandatory].")

	//connectorGet flag :
	ConnectorGetCmd.Flags().StringVar(&outputTypeGet, "output-type", "table", "Specifies the output type[table , json][optional]")
	ConnectorGetCmd.Flags().StringVar(&workspaceNameGet, "workspace-name", "", "specifies the workspace name[mandatory].")

	ConnectorNameCmd.Flags().StringVar(&connectorNameGet, "name", "", "Specifying the connectorGet name [mandatory].")
	ConnectorNameCmd.Flags().StringVar(&outputTypeGet, "output-type", "table", "Specifies the output type[table , json][optional]")
	ConnectorNameCmd.Flags().StringVar(&workspaceNameGet, "workspace-name", "", "specifies the workspace name[mandatory].")
}
