package onboard

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	api3 "gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/utils"

	keibiaws "github.com/kaytu-io/kaytu-aws-describer/aws"
	keibiazure "github.com/kaytu-io/kaytu-azure-describer/azure"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe"
	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"
	"gorm.io/gorm"
)

const (
	paramSourceId     = "sourceId"
	paramCredentialId = "credentialId"
)

func (h HttpHandler) Register(r *echo.Echo) {
	v1 := r.Group("/api/v1")

	v1.GET("/sources", httpserver.AuthorizeHandler(h.ListSources, api3.ViewerRole))
	v1.POST("/sources", httpserver.AuthorizeHandler(h.GetSources, api3.ViewerRole))
	v1.GET("/sources/count", httpserver.AuthorizeHandler(h.CountSources, api3.ViewerRole))
	v1.GET("/catalog/metrics", httpserver.AuthorizeHandler(h.CatalogMetrics, api3.ViewerRole))

	connector := v1.Group("/connector")
	connector.GET("", httpserver.AuthorizeHandler(h.ListConnectors, api3.ViewerRole))
	connector.GET("/:connectorId", httpserver.AuthorizeHandler(h.GetConnector, api3.ViewerRole))

	sourceApiGroup := v1.Group("/source")
	sourceApiGroup.POST("/aws", httpserver.AuthorizeHandler(h.PostSourceAws, api3.EditorRole))
	sourceApiGroup.POST("/azure", httpserver.AuthorizeHandler(h.PostSourceAzure, api3.EditorRole))
	sourceApiGroup.GET("/:sourceId", httpserver.AuthorizeHandler(h.GetSource, api3.ViewerRole))
	sourceApiGroup.GET("/:sourceId/healthcheck", httpserver.AuthorizeHandler(h.GetSourceHealth, api3.EditorRole))
	sourceApiGroup.GET("/:sourceId/credentials", httpserver.AuthorizeHandler(h.GetSourceCred, api3.ViewerRole))
	sourceApiGroup.PUT("/:sourceId/credentials", httpserver.AuthorizeHandler(h.PutSourceCred, api3.EditorRole))
	sourceApiGroup.POST("/:sourceId/disable", httpserver.AuthorizeHandler(h.DisableSource, api3.EditorRole))
	sourceApiGroup.POST("/:sourceId/enable", httpserver.AuthorizeHandler(h.EnableSource, api3.EditorRole))
	sourceApiGroup.DELETE("/:sourceId", httpserver.AuthorizeHandler(h.DeleteSource, api3.EditorRole))

	credential := v1.Group("/credential")
	credential.POST("", httpserver.AuthorizeHandler(h.PostCredentials, api3.EditorRole))
	credential.PUT("/:credentialId", httpserver.AuthorizeHandler(h.PutCredentials, api3.EditorRole))
	credential.GET("", httpserver.AuthorizeHandler(h.ListCredentials, api3.ViewerRole))
	credential.GET("/sources/list", httpserver.AuthorizeHandler(h.ListSourcesByCredentials, api3.ViewerRole))
	credential.DELETE("/:credentialId", httpserver.AuthorizeHandler(h.DeleteCredential, api3.EditorRole))
	credential.POST("/:credentialId/disable", httpserver.AuthorizeHandler(h.DisableCredential, api3.EditorRole))
	credential.POST("/:credentialId/enable", httpserver.AuthorizeHandler(h.EnableCredential, api3.EditorRole))
	credential.GET("/:credentialId", httpserver.AuthorizeHandler(h.GetCredential, api3.ViewerRole))
	credential.POST("/:credentialId/autoonboard", httpserver.AuthorizeHandler(h.AutoOnboardCredential, api3.EditorRole))
	credential.GET("/:credentialId/healthcheck", httpserver.AuthorizeHandler(h.GetCredentialHealth, api3.EditorRole))

	connections := v1.Group("/connections")
	connections.POST("/count", httpserver.AuthorizeHandler(h.CountConnections, api3.ViewerRole))
}

func bindValidate(ctx echo.Context, i interface{}) error {
	if err := ctx.Bind(i); err != nil {
		return err
	}

	if err := ctx.Validate(i); err != nil {
		return err
	}

	return nil
}

// GetProviders godoc
//
//	@Summary		Get providers
//	@Description	Getting cloud providers
//	@Tags			onboard
//	@Produce		json
//	@Success		200	{object}	api.ProvidersResponse
//	@Router			/onboard/api/v1/providers [get]
func (h HttpHandler) GetProviders(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, api.ProvidersResponse{
		{Name: "Sumo Logic", ID: "sumologic", Type: "IT Operations", State: api.ProviderStateDisabled},
		{Name: "Akamai", ID: "akamai", Type: "Content Delivery (CDN)", State: api.ProviderStateDisabled},
		{Name: "Box", ID: "boxnet", Type: "Cloud Storage", State: api.ProviderStateDisabled},
		{Name: "DropBox", ID: "dropbox", Type: "Cloud Storage", State: api.ProviderStateDisabled},
		{Name: "Microsoft OneDrive", ID: "onedrive", Type: "Cloud Storage", State: api.ProviderStateDisabled},
		{Name: "Kubernetes", ID: "kubernetes", Type: "Cointainer Orchestrator", State: api.ProviderStateComingSoon},
		{Name: "Box", ID: "boxnet", Type: "Collaboration & Productivity", State: api.ProviderStateDisabled},
		{Name: "DocuSign", ID: "docusign", Type: "Collaboration & Productivity", State: api.ProviderStateDisabled},
		{Name: "Google Workspace", ID: "googleworkspace", Type: "Collaboration & Productivity", State: api.ProviderStateDisabled},
		{Name: "Microsoft Office 365", ID: "o365", Type: "Collaboration & Productivity", State: api.ProviderStateDisabled},
		{Name: "Microsoft SharePoint", ID: "sharepoint", Type: "Collaboration & Productivity", State: api.ProviderStateDisabled},
		{Name: "Microsoft Teams", ID: "teams", Type: "Collaboration & Productivity", State: api.ProviderStateDisabled},
		{Name: "Slack", ID: "slack", Type: "Collaboration & Productivity", State: api.ProviderStateDisabled},
		{Name: "Trello", ID: "trello", Type: "Collaboration & Productivity", State: api.ProviderStateDisabled},
		{Name: "Zoom", ID: "zoom", Type: "Collaboration & Productivity", State: api.ProviderStateDisabled},
		{Name: "Mailchimp", ID: "mailchimp", Type: "Communications", State: api.ProviderStateDisabled},
		{Name: "PagerDuty", ID: "pagerduty", Type: "Communications", State: api.ProviderStateDisabled},
		{Name: "RingCentral", ID: "ringcentral", Type: "Communications", State: api.ProviderStateDisabled},
		{Name: "Twilio SendGrid", ID: "sendgrid", Type: "Communications", State: api.ProviderStateDisabled},
		{Name: "Mailchimp", ID: "mailchimp", Type: "Communications", State: api.ProviderStateDisabled},
		{Name: "Mailgun", ID: "mailgun", Type: "Communications", State: api.ProviderStateDisabled},
		{Name: "Rubrik", ID: "rubrik", Type: "Data Management", State: api.ProviderStateDisabled},
		{Name: "Snowflake", ID: "snowflake", Type: "Data Management", State: api.ProviderStateDisabled},
		{Name: "talend.com", ID: "talend", Type: "Data Management", State: api.ProviderStateDisabled},
		{Name: "MongoDB Atlas", ID: "mongodbatlast", Type: "Databases", State: api.ProviderStateDisabled},
		{Name: "Elastic Cloud", ID: "elasticcloud", Type: "Databases", State: api.ProviderStateDisabled},
		{Name: "Okta", ID: "okta", Type: "Identity", State: api.ProviderStateDisabled},
		{Name: "JumpCloud", ID: "jumpcloud", Type: "Identity", State: api.ProviderStateDisabled},
		{Name: "Ping Identity", ID: "pingidentity", Type: "Identity", State: api.ProviderStateDisabled},
		{Name: "Auth0.com", ID: "auth0", Type: "Identity", State: api.ProviderStateDisabled},
		{Name: "Microsoft Azure Active Directory", ID: "azuread", Type: "Identity", State: api.ProviderStateComingSoon},
		{Name: "OneLogin", ID: "onelogin", Type: "Identity", State: api.ProviderStateDisabled},
		{Name: "Expensify", ID: "expensify", Type: "Enterprise Applications", State: api.ProviderStateDisabled},
		{Name: "Salesforce", ID: "salesforce", Type: "Enterprise Applications", State: api.ProviderStateDisabled},
		{Name: "Xero", ID: "xero", Type: "Enterprise Applications", State: api.ProviderStateDisabled},
		{Name: "AppViewX", ID: "appviewx", Type: "Enterprise Security", State: api.ProviderStateDisabled},
		{Name: "Rapid7", ID: "rapid7", Type: "Enterprise Security", State: api.ProviderStateDisabled},
		{Name: "Akamai", ID: "akamai", Type: "Edge Compute", State: api.ProviderStateDisabled},
		{Name: "Akamai", ID: "akamai", Type: "Enterprise Security", State: api.ProviderStateDisabled},
		{Name: "Imperva", ID: "imperva", Type: "Enterprise Security", State: api.ProviderStateDisabled},
		{Name: "Cloudflare", ID: "cloudfare", Type: "Enterprise Security", State: api.ProviderStateDisabled},
		{Name: "CyberArk", ID: "cuberark", Type: "Enterprise Security", State: api.ProviderStateDisabled},
		{Name: "Blackberry CylanceProtect", ID: "cylance", Type: "Enterprise Security", State: api.ProviderStateDisabled},
		{Name: "Cisco Duo", ID: "duo", Type: "Enterprise Security", State: api.ProviderStateDisabled},
		{Name: "OneLogin", ID: "onelogin", Type: "Enterprise Security", State: api.ProviderStateDisabled},
		{Name: "OneTrust", ID: "onetrust", Type: "Enterprise Security", State: api.ProviderStateDisabled},
		{Name: "PaloAlto Networks Prisma", ID: "prismacloud", Type: "Enterprise Security", State: api.ProviderStateDisabled},
		{Name: "Ping Identity", ID: "pingidentity", Type: "Enterprise Security", State: api.ProviderStateDisabled},
		{Name: "SignalSciences", ID: "signalscience", Type: "Enterprise Security", State: api.ProviderStateDisabled},
		{Name: "StrongDM", ID: "strongdm", Type: "Enterprise Security", State: api.ProviderStateDisabled},
		{Name: "Sumo Logic", ID: "sumologic", Type: "Enterprise Security", State: api.ProviderStateDisabled},
		{Name: "Tenable", ID: "tenable", Type: "Enterprise Security", State: api.ProviderStateDisabled},
		{Name: "Atlassian", ID: "atlassian", Type: "IT Operations", State: api.ProviderStateDisabled},
		{Name: "DataDog", ID: "datadog", Type: "IT Operations", State: api.ProviderStateDisabled},
		{Name: "PagerDuty", ID: "pagerduty", Type: "IT Operations", State: api.ProviderStateDisabled},
		{Name: "RingCentral", ID: "ringcentral", Type: "IT Operations", State: api.ProviderStateDisabled},
		{Name: "ServiceNow", ID: "servicenow", Type: "IT Operations", State: api.ProviderStateDisabled},
		{Name: "Zendesk", ID: "zendesk", Type: "IT Operations", State: api.ProviderStateDisabled},
		{Name: "Splunk", ID: "splunk", Type: "IT Operations", State: api.ProviderStateDisabled},
		{Name: "Confluent", ID: "confluence", Type: "Messaging", State: api.ProviderStateDisabled},
		{Name: "Splunk", ID: "splunk", Type: "Observability", State: api.ProviderStateDisabled},
		{Name: "DataDog", ID: "datadog", Type: "Observability", State: api.ProviderStateDisabled},
		{Name: "OpenStack", ID: "openstack", Type: "Private Cloud", State: api.ProviderStateDisabled},
		{Name: "VMWare", ID: "vmware", Type: "Private Cloud", State: api.ProviderStateComingSoon},
		{Name: "HPE Helion", ID: "hpehelion", Type: "Private Cloud", State: api.ProviderStateDisabled},
		{Name: "Amazon Web Services", ID: "aws", Type: "Public Cloud", State: api.ProviderStateEnabled},
		{Name: "Google Cloud Platform", ID: "gcp", Type: "Public Cloud", State: api.ProviderStateComingSoon},
		{Name: "Oracle Cloud Infrastructure", ID: "oci", Type: "Public Cloud", State: api.ProviderStateDisabled},
		{Name: "Alibaba Cloud", ID: "alibabacloud", Type: "Public Cloud", State: api.ProviderStateDisabled},
		{Name: "Tencent Cloud", ID: "tencentcloud", Type: "Public Cloud", State: api.ProviderStateDisabled},
		{Name: "IBM Cloud", ID: "ibmcloud", Type: "Public Cloud", State: api.ProviderStateDisabled},
		{Name: "Microsoft Azure", ID: "azure", Type: "Public Cloud", State: api.ProviderStateEnabled},
		{Name: "Salesforce Tableau", ID: "tableau", Type: "Reporting", State: api.ProviderStateDisabled},
		{Name: "Google Looker", ID: "looker", Type: "Reporting", State: api.ProviderStateDisabled},
		{Name: "Gitlab.com", ID: "gitlab", Type: "Source Code Management", State: api.ProviderStateComingSoon},
		{Name: "GitHub", ID: "github", Type: "Source Code Management", State: api.ProviderStateComingSoon},
		{Name: "Azure DevOps", ID: "azuredevops", Type: "Source Code Management", State: api.ProviderStateDisabled},
		{Name: "Jfrog", ID: "jfrog", Type: "Source Code Management", State: api.ProviderStateDisabled},
		{Name: "NewRelic", ID: "newrelic", Type: "Observability", State: api.ProviderStateDisabled},
		{Name: "DynaTrace", ID: "dynatrace", Type: "Observability", State: api.ProviderStateDisabled},
	})
}

// ListConnectors godoc
//
//	@Summary		Get connectors
//	@Description	Getting connectors
//	@Tags			onboard
//	@Produce		json
//	@Success		200	{object}	[]api.ConnectorCount
//	@Router			/onboard/api/v1/connectors [get]
func (h HttpHandler) ListConnectors(ctx echo.Context) error {
	connectors, err := h.db.ListConnectors()
	if err != nil {
		return err
	}

	var res []api.ConnectorCount
	for _, c := range connectors {
		count, err := h.db.CountSourcesOfType(c.Name)
		if err != nil {
			return err
		}
		tags := make(map[string]any)
		err = json.Unmarshal(c.Tags, &tags)
		if err != nil {
			return err
		}
		res = append(res, api.ConnectorCount{
			Connector: api.Connector{
				Name:                c.Name,
				Label:               c.Label,
				ShortDescription:    c.ShortDescription,
				Description:         c.Description,
				Direction:           c.Direction,
				Status:              c.Status,
				Logo:                c.Logo,
				AutoOnboardSupport:  c.AutoOnboardSupport,
				AllowNewConnections: c.AllowNewConnections,
				MaxConnectionLimit:  c.MaxConnectionLimit,
				Tags:                tags,
			},
			ConnectionCount: count,
		})

	}
	return ctx.JSON(http.StatusOK, res)
}

// GetConnector godoc
//
//	@Summary		Get connector
//	@Description	Getting connector
//	@Tags			onboard
//	@Produce		json
//	@Param			connectorName	path		string	true	"Connector name"
//	@Success		200				{object}	api.Connector
//	@Router			/onboard/api/v1/connectors/{connectorName} [get]
func (h HttpHandler) GetConnector(ctx echo.Context) error {
	connectorName, err := source.ParseType(ctx.Param("connectorName"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, fmt.Sprintf("Invalid connector name"))
	}

	c, err := h.db.GetConnector(connectorName)
	if err != nil {
		return err
	}
	count, err := h.db.CountSourcesOfType(c.Name)
	if err != nil {
		return err
	}
	tags := make(map[string]any)
	err = json.Unmarshal(c.Tags, &tags)
	if err != nil {
		return err
	}
	res := api.ConnectorCount{
		Connector: api.Connector{
			Name:                c.Name,
			Label:               c.Label,
			ShortDescription:    c.ShortDescription,
			Description:         c.Description,
			Direction:           c.Direction,
			Status:              c.Status,
			Logo:                c.Logo,
			AutoOnboardSupport:  c.AutoOnboardSupport,
			AllowNewConnections: c.AllowNewConnections,
			MaxConnectionLimit:  c.MaxConnectionLimit,
			Tags:                tags,
		},
		ConnectionCount: count,
	}

	return ctx.JSON(http.StatusOK, res)
}

// GetProviderTypes godoc
//
//	@Summary		Get provider types
//	@Description	Getting provider types
//	@Tags			onboard
//	@Produce		json
//	@Success		200	{object}	api.ProviderTypesResponse
//	@Router			/onboard/api/v1/providers/types [get]
func (h HttpHandler) GetProviderTypes(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, api.ProviderTypesResponse{
		{ID: "1", TypeName: "Public Cloud", State: api.ProviderTypeStateEnabled},
		{ID: "2", TypeName: "Cointainer Orchestrator", State: api.ProviderTypeStateComingSoon},
		{ID: "3", TypeName: "Private Cloud", State: api.ProviderTypeStateComingSoon},
		{ID: "4", TypeName: "Source Code Management", State: api.ProviderTypeStateComingSoon},
		{ID: "5", TypeName: "Identity", State: api.ProviderTypeStateComingSoon},
		{ID: "6", TypeName: "Enterprise Security", State: api.ProviderTypeStateDisabled},
		{ID: "7", TypeName: "Observability", State: api.ProviderTypeStateDisabled},
		{ID: "8", TypeName: "Messaging", State: api.ProviderTypeStateDisabled},
		{ID: "9", TypeName: "Communications", State: api.ProviderTypeStateDisabled},
		{ID: "10", TypeName: "IT Operations", State: api.ProviderTypeStateDisabled},
		{ID: "11", TypeName: "Enterprise Applications", State: api.ProviderTypeStateDisabled},
		{ID: "12", TypeName: "Databases", State: api.ProviderTypeStateDisabled},
		{ID: "13", TypeName: "Data Management", State: api.ProviderTypeStateDisabled},
		{ID: "14", TypeName: "Cloud Storage", State: api.ProviderTypeStateDisabled},
		{ID: "15", TypeName: "Content Delivery (CDN)", State: api.ProviderTypeStateDisabled},
		{ID: "16", TypeName: "Collaboration & Productivity", State: api.ProviderTypeStateDisabled},
		{ID: "17", TypeName: "Edge Compute", State: api.ProviderTypeStateDisabled},
		{ID: "18", TypeName: "Reporting", State: api.ProviderTypeStateDisabled},
	})
}

// PostSourceAws godoc
//
//	@Summary		Create AWS source
//	@Description	Creating AWS source
//	@Tags			onboard
//	@Produce		json
//	@Success		200			{object}	api.CreateSourceResponse
//	@Param			name		body		string				true	"name"
//	@Param			description	body		string				true	"description"
//	@Param			config		body		api.SourceConfigAWS	true	"config"
//	@Router			/onboard/api/v1/source/aws [post]
func (h HttpHandler) PostSourceAws(ctx echo.Context) error {
	var req api.SourceAwsRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	// Check creds section
	err := keibiaws.CheckDescribeRegionsPermission(req.Config.AccessKey, req.Config.SecretKey)
	if err != nil {
		fmt.Printf("error in checking describe regions permission: %v", err)
		return echo.NewHTTPError(http.StatusUnauthorized, PermissionError.Error())
	}

	isAttached, err := keibiaws.CheckAttachedPolicy(req.Config.AccessKey, req.Config.SecretKey, keibiaws.SecurityAuditPolicyARN)
	if err != nil {
		fmt.Printf("error in checking security audit permission: %v", err)
		return echo.NewHTTPError(http.StatusUnauthorized, PermissionError.Error())
	}
	if !isAttached {
		return echo.NewHTTPError(http.StatusUnauthorized, "Failed to find read access policy")
	}

	// Create source section
	cfg, err := keibiaws.GetConfig(context.Background(), req.Config.AccessKey, req.Config.SecretKey, "", "")
	if err != nil {
		return err
	}

	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}

	acc, err := currentAwsAccount(context.Background(), cfg)
	if err != nil {
		return err
	}
	if req.Name != "" {
		acc.AccountName = &req.Name
	}

	count, err := h.db.CountSources()
	if err != nil {
		return err
	}
	if count >= httpserver.GetMaxConnections(ctx) {
		return echo.NewHTTPError(http.StatusBadRequest, "maximum number of connections reached")
	}

	src := NewAWSSource(*acc, req.Description)
	secretBytes, err := h.kms.Encrypt(req.Config.AsMap(), h.keyARN)
	if err != nil {
		return err
	}
	src.Credential.Secret = string(secretBytes)

	err = h.db.orm.Transaction(func(tx *gorm.DB) error {
		err := h.db.CreateSource(&src)
		if err != nil {
			return err
		}

		if err := h.sourceEventsQueue.Publish(api.SourceEvent{
			Action:     api.SourceCreated,
			SourceID:   src.ID,
			AccountID:  src.SourceId,
			SourceType: src.Type,
			Secret:     src.Credential.Secret,
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, src.ToSourceResponse())
}

// PostSourceAzure godoc
//
//	@Summary		Create Azure source
//	@Description	Creating Azure source
//	@Tags			onboard
//	@Produce		json
//	@Success		200			{object}	api.CreateSourceResponse
//	@Param			name		body		string					true	"name"
//	@Param			description	body		string					true	"description"
//	@Param			config		body		api.SourceConfigAzure	true	"config"
//	@Router			/onboard/api/v1/source/azure [post]
func (h HttpHandler) PostSourceAzure(ctx echo.Context) error {
	var req api.SourceAzureRequest

	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	count, err := h.db.CountSources()
	if err != nil {
		return err
	}
	if count >= httpserver.GetMaxConnections(ctx) {
		return echo.NewHTTPError(http.StatusBadRequest, "maximum number of connections reached")
	}

	isAttached, err := keibiazure.CheckRole(keibiazure.AuthConfig{
		TenantID:     req.Config.TenantId,
		ObjectID:     req.Config.ObjectId,
		SecretID:     req.Config.SecretId,
		ClientID:     req.Config.ClientId,
		ClientSecret: req.Config.ClientSecret,
	}, req.Config.SubscriptionId, keibiazure.DefaultReaderRoleDefinitionIDTemplate)
	if err != nil {
		fmt.Printf("error in checking reader role roleAssignment: %v", err)
		return echo.NewHTTPError(http.StatusUnauthorized, PermissionError.Error())
	}
	if !isAttached {
		return echo.NewHTTPError(http.StatusUnauthorized, "Failed to find reader role roleAssignment")
	}

	cred, err := createAzureCredential(
		ctx.Request().Context(),
		fmt.Sprintf("%s - %s - default credentials", source.CloudAzure, req.Config.SubscriptionId),
		source.CredentialTypeAutoGenerated,
		req.Config,
	)
	if err != nil {
		return err
	}

	azSub, err := currentAzureSubscription(ctx.Request().Context(), req.Config.SubscriptionId, keibiazure.AuthConfig{
		TenantID:     req.Config.TenantId,
		ObjectID:     req.Config.ObjectId,
		SecretID:     req.Config.SecretId,
		ClientID:     req.Config.ClientId,
		ClientSecret: req.Config.ClientSecret,
	})
	if err != nil {
		return err
	}

	src := NewAzureSourceWithCredentials(*azSub, source.SourceCreationMethodManual, req.Description, *cred)
	err = h.db.orm.Transaction(func(tx *gorm.DB) error {
		err := h.db.CreateSource(&src)
		if err != nil {
			return err
		}

		secretBytes, err := h.kms.Encrypt(req.Config.AsMap(), h.keyARN)
		if err != nil {
			return err
		}
		src.Credential.Secret = string(secretBytes)

		if err := h.sourceEventsQueue.Publish(api.SourceEvent{
			Action:     api.SourceCreated,
			SourceID:   src.ID,
			AccountID:  src.SourceId,
			SourceType: src.Type,
			Secret:     src.Credential.Secret,
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, src.ToSourceResponse())
}

func (h HttpHandler) checkCredentialHealth(cred Credential) (bool, error) {
	config, err := h.kms.Decrypt(cred.Secret, h.keyARN)
	if err != nil {
		return false, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	switch cred.ConnectorType {
	case source.CloudAWS:
		var awsConfig describe.AWSAccountConfig
		awsConfig, err = describe.AWSAccountConfigFromMap(config)
		if err != nil {
			return false, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		err = keibiaws.CheckGetUserPermission(awsConfig.AccessKey, awsConfig.SecretKey)
		if err == nil {
			metadata, err := getAWSCredentialsMetadata(context.Background(), awsConfig)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			jsonMetadata, err := json.Marshal(metadata)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			cred.Metadata = jsonMetadata
		}

	case source.CloudAzure:
		var azureConfig describe.AzureSubscriptionConfig
		azureConfig, err = describe.AzureSubscriptionConfigFromMap(config)
		if err != nil {
			return false, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		err = keibiazure.CheckSPNAccessPermission(keibiazure.AuthConfig{
			TenantID:            azureConfig.TenantID,
			ObjectID:            azureConfig.ObjectID,
			SecretID:            azureConfig.SecretID,
			ClientID:            azureConfig.ClientID,
			ClientSecret:        azureConfig.ClientSecret,
			CertificatePath:     azureConfig.CertificatePath,
			CertificatePassword: azureConfig.CertificatePass,
			Username:            azureConfig.Username,
			Password:            azureConfig.Password,
		})
		if err == nil {
			metadata, err := getAzureCredentialsMetadata(context.Background(), azureConfig)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			jsonMetadata, err := json.Marshal(metadata)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			cred.Metadata = jsonMetadata
		}
	}

	if err != nil {
		errStr := err.Error()
		cred.HealthReason = &errStr
		cred.HealthStatus = source.HealthStatusUnhealthy
	} else {
		cred.HealthStatus = source.HealthStatusHealthy
		cred.HealthReason = nil
	}
	cred.LastHealthCheckTime = time.Now()

	_, dbErr := h.db.UpdateCredential(&cred)
	if dbErr != nil {
		return false, echo.NewHTTPError(http.StatusInternalServerError, dbErr.Error())
	}

	if err != nil {
		return false, echo.NewHTTPError(http.StatusBadRequest, "credential is not healthy")
	}

	return true, nil
}

func createAzureCredential(ctx context.Context, name string, credType source.CredentialType, config api.SourceConfigAzure) (*Credential, error) {
	azureCnf, err := describe.AzureSubscriptionConfigFromMap(config.AsMap())
	if err != nil {
		return nil, err
	}

	metadata, err := getAzureCredentialsMetadata(ctx, azureCnf)
	if err != nil {
		return nil, err
	}
	cred, err := NewAzureCredential(name, credType, metadata)
	if err != nil {
		return nil, err
	}
	return cred, nil
}

func (h HttpHandler) postAzureCredentials(ctx echo.Context, req api.CreateCredentialRequest) error {
	configStr, err := json.Marshal(req.Config)
	if err != nil {
		return err
	}
	config := api.SourceConfigAzure{}
	err = json.Unmarshal(configStr, &config)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, "invalid config")
	}

	cred, err := createAzureCredential(ctx.Request().Context(), req.Name, source.CredentialTypeManual, config)
	if err != nil {
		return err
	}

	err = h.db.orm.Transaction(func(tx *gorm.DB) error {
		secretBytes, err := h.kms.Encrypt(config.AsMap(), h.keyARN)
		if err != nil {
			return err
		}
		cred.Secret = string(secretBytes)

		if err := h.db.CreateCredential(cred); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	_, err = h.checkCredentialHealth(*cred)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, api.CreateCredentialResponse{ID: cred.ID.String()})
}

func (h HttpHandler) postAWSCredentials(ctx echo.Context, req api.CreateCredentialRequest) error {
	configStr, err := json.Marshal(req.Config)
	if err != nil {
		return err
	}
	config := api.SourceConfigAWS{}
	err = json.Unmarshal(configStr, &config)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, "invalid config")
	}

	awsCnf, err := describe.AWSAccountConfigFromMap(config.AsMap())
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, "invalid config")
	}

	metadata, err := getAWSCredentialsMetadata(ctx.Request().Context(), awsCnf)
	if err != nil {
		return err
	}
	cred, err := NewAWSCredential(req.Name, metadata)
	if err != nil {
		return err
	}

	err = h.db.orm.Transaction(func(tx *gorm.DB) error {
		secretBytes, err := h.kms.Encrypt(config.AsMap(), h.keyARN)
		if err != nil {
			return err
		}
		cred.Secret = string(secretBytes)

		if err := h.db.CreateCredential(cred); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	_, err = h.checkCredentialHealth(*cred)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, api.CreateCredentialResponse{ID: cred.ID.String()})
}

// PostCredentials godoc
//
//	@Summary		Create connection credentials
//	@Description	Creating connection credentials
//	@Tags			onboard
//	@Produce		json
//	@Success		200		{object}	api.CreateCredentialResponse
//	@Param			config	body		api.CreateCredentialRequest	true	"config"
//	@Router			/onboard/api/v1/credential [post]
func (h HttpHandler) PostCredentials(ctx echo.Context) error {
	var req api.CreateCredentialRequest

	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	switch req.SourceType {
	case source.CloudAzure:
		return h.postAzureCredentials(ctx, req)
	case source.CloudAWS:
		return h.postAWSCredentials(ctx, req)
	}

	return echo.NewHTTPError(http.StatusBadRequest, "invalid source type")
}

// ListCredentials godoc
//
//	@Summary		List credentials
//	@Description	List credentials
//	@Tags			onboard
//	@Produce		json
//	@Success		200			{object}	[]api.Credential
//	@Param			connector	query		source.Type	false	"filter by connector type"
//	@Param			health		query		string		false	"filter by health status"	Enums(healthy, unhealthy, initial_discovery)
//	@Param			pageSize	query		int			false	"page size"					default(50)
//	@Param			pageNumber	query		int			false	"page number"				default(1)
//	@Router			/onboard/api/v1/credential [get]
func (h HttpHandler) ListCredentials(ctx echo.Context) error {
	connector, _ := source.ParseType(ctx.QueryParam("connector"))
	health, _ := source.ParseHealthStatus(ctx.QueryParam("health"))
	pageSizeStr := ctx.QueryParam("pageSize")
	pageNumberStr := ctx.QueryParam("pageNumber")
	pageSize := int64(50)
	pageNumber := int64(1)
	if pageSizeStr != "" {
		pageSize, _ = strconv.ParseInt(pageSizeStr, 10, 64)
	}
	if pageNumberStr != "" {
		pageNumber, _ = strconv.ParseInt(pageNumberStr, 10, 64)
	}

	credentials, err := h.db.GetCredentialsByFilters(connector, health)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	apiCredentials := make([]api.Credential, 0, len(credentials))
	for _, cred := range credentials {
		metadata := make(map[string]any)
		err = json.Unmarshal(cred.Metadata, &metadata)
		if err != nil {
			return err
		}
		apiCredentials = append(apiCredentials, api.Credential{
			ID:                  cred.ID.String(),
			Name:                cred.Name,
			ConnectorType:       cred.ConnectorType,
			CredentialType:      cred.CredentialType,
			Enabled:             cred.Enabled,
			OnboardDate:         cred.CreatedAt,
			LastHealthCheckTime: cred.LastHealthCheckTime,
			HealthStatus:        cred.HealthStatus,
			HealthReason:        cred.HealthReason,
			Metadata:            metadata,
		})
	}

	sort.Slice(apiCredentials, func(i, j int) bool {
		return apiCredentials[i].OnboardDate.After(apiCredentials[j].OnboardDate)
	})

	return ctx.JSON(http.StatusOK, utils.Paginate(pageNumber, pageSize, apiCredentials))
}

// GetCredential godoc
//
//	@Summary		List credentials
//	@Description	List credentials
//	@Tags			onboard
//	@Produce		json
//	@Success		200	{object}	api.Credential
//	@Router			/onboard/api/v1/credential/{credentialId} [get]
func (h HttpHandler) GetCredential(ctx echo.Context) error {
	credId, err := uuid.Parse(ctx.Param(paramCredentialId))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	credential, err := h.db.GetCredentialByID(credId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "credential not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	connections, err := h.db.GetSourcesByCredentialID(credId.String())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	metadata := make(map[string]any)
	err = json.Unmarshal(credential.Metadata, &metadata)
	if err != nil {
		return err
	}

	apiCredential := api.Credential{
		ID:                  credential.ID.String(),
		Name:                credential.Name,
		ConnectorType:       credential.ConnectorType,
		CredentialType:      credential.CredentialType,
		Enabled:             credential.Enabled,
		OnboardDate:         credential.CreatedAt,
		LastHealthCheckTime: credential.LastHealthCheckTime,
		HealthStatus:        credential.HealthStatus,
		HealthReason:        credential.HealthReason,
		Metadata:            metadata,
		Connections:         make([]api.Source, 0, len(connections)),
	}
	for _, conn := range connections {
		metadata := make(map[string]any)
		if conn.Metadata.String() != "" {
			err := json.Unmarshal(conn.Metadata, &metadata)
			if err != nil {
				return err
			}
		}

		apiCredential.Connections = append(apiCredential.Connections, api.Source{
			ID:                   conn.ID,
			ConnectionID:         conn.SourceId,
			ConnectionName:       conn.Name,
			Email:                conn.Email,
			Type:                 conn.Type,
			Description:          conn.Description,
			CredentialID:         conn.CredentialID.String(),
			CredentialName:       conn.Credential.Name,
			OnboardDate:          conn.CreatedAt,
			LifecycleState:       api.ConnectionLifecycleState(conn.LifecycleState),
			AssetDiscoveryMethod: conn.AssetDiscoveryMethod,
			HealthState:          conn.HealthState,
			LastHealthCheckTime:  conn.LastHealthCheckTime,
			HealthReason:         conn.HealthReason,
			Metadata:             metadata,
		})
	}

	return ctx.JSON(http.StatusOK, apiCredential)
}

// AutoOnboardCredential godoc
//
//	@Summary		Onboard all available connections for a credential
//	@Description	Onboard all available connections for a credential
//	@Tags			onboard
//	@Produce		json
//	@Success		200	{object}	[]api.Source
//	@Router			/onboard/api/v1/credential/{credentialId}/autoonboard [post]
func (h HttpHandler) AutoOnboardCredential(ctx echo.Context) error {
	credId, err := uuid.Parse(ctx.Param(paramCredentialId))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	credential, err := h.db.GetCredentialByID(credId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "credential not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	onboardedSources := make([]api.Source, 0)
	switch credential.ConnectorType {
	case source.CloudAzure:
		cnf, err := h.kms.Decrypt(credential.Secret, h.keyARN)
		if err != nil {
			return err
		}
		azureCnf, err := describe.AzureSubscriptionConfigFromMap(cnf)

		subs, err := discoverAzureSubscriptions(ctx.Request().Context(), keibiazure.AuthConfig{
			TenantID:     azureCnf.TenantID,
			ObjectID:     azureCnf.ObjectID,
			SecretID:     azureCnf.SecretID,
			ClientID:     azureCnf.ClientID,
			ClientSecret: azureCnf.ClientSecret,
		})

		existingConnections, err := h.db.GetSourcesByCredentialID(credential.ID.String())
		if err != nil {
			return err
		}

		existingConnectionSubIDs := make([]string, 0, len(existingConnections))
		for _, conn := range existingConnections {
			existingConnectionSubIDs = append(existingConnectionSubIDs, conn.SourceId)
		}

		for _, sub := range subs {
			if utils.Includes(existingConnectionSubIDs, sub.SubscriptionID) {
				continue
			}

			count, err := h.db.CountSources()
			if err != nil {
				return err
			}
			if count >= httpserver.GetMaxConnections(ctx) {
				return echo.NewHTTPError(http.StatusBadRequest, "maximum number of connections reached")
			}

			isAttached, err := keibiazure.CheckRole(keibiazure.AuthConfig{
				TenantID:     azureCnf.TenantID,
				ObjectID:     azureCnf.ObjectID,
				SecretID:     azureCnf.SecretID,
				ClientID:     azureCnf.ClientID,
				ClientSecret: azureCnf.ClientSecret,
			}, sub.SubscriptionID, keibiazure.DefaultReaderRoleDefinitionIDTemplate)
			if err != nil {
				continue
			}
			if !isAttached {
				continue
			}

			src := NewAzureSourceWithCredentials(
				sub,
				source.SourceCreationMethodAutoOnboard,
				fmt.Sprintf("Auto onboarded subscription %s", sub.SubscriptionID),
				*credential,
			)

			err = h.db.orm.Transaction(func(tx *gorm.DB) error {
				err := h.db.CreateSource(&src)
				if err != nil {
					return err
				}

				if err := h.sourceEventsQueue.Publish(api.SourceEvent{
					Action:     api.SourceCreated,
					SourceID:   src.ID,
					AccountID:  src.SourceId,
					SourceType: src.Type,
					Secret:     src.Credential.Secret,
				}); err != nil {
					return err
				}

				return nil
			})
			if err != nil {
				return err
			}

			metadata := make(map[string]any)
			if src.Metadata.String() != "" {
				err := json.Unmarshal(src.Metadata, &metadata)
				if err != nil {
					return err
				}
			}

			onboardedSources = append(onboardedSources, api.Source{
				ID:                   src.ID,
				ConnectionID:         src.SourceId,
				ConnectionName:       src.Name,
				Email:                src.Email,
				Type:                 src.Type,
				Description:          src.Description,
				CredentialID:         src.CredentialID.String(),
				CredentialName:       src.Credential.Name,
				OnboardDate:          src.CreatedAt,
				LifecycleState:       api.ConnectionLifecycleState(src.LifecycleState),
				AssetDiscoveryMethod: src.AssetDiscoveryMethod,
				HealthState:          src.HealthState,
				LastHealthCheckTime:  src.LastHealthCheckTime,
				HealthReason:         src.HealthReason,
				Metadata:             metadata,
			})
		}
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "connector doesn't support auto onboard")
	}

	return ctx.JSON(http.StatusOK, onboardedSources)
}

// GetCredentialHealth godoc
//
//	@Summary	Get live credential health status
//	@Tags		onboard
//	@Produce	json
//	@Router		/onboard/api/v1/credential/{credentialId}/healthcheck [post]
func (h HttpHandler) GetCredentialHealth(ctx echo.Context) error {
	credUUID, err := uuid.Parse(ctx.Param("credentialId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
	}

	credential, err := h.db.GetCredentialByID(credUUID)
	if err != nil {
		return err
	}

	isHealthy, err := h.checkCredentialHealth(*credential)
	if err != nil {
		return err
	}
	if !isHealthy {
		return echo.NewHTTPError(http.StatusBadRequest, "credential is not healthy")
	}
	return ctx.JSON(http.StatusOK, struct{}{})
}

// ListSourcesByCredentials godoc
//
//	@Summary		Returns a list of sources
//	@Description	Returning a list of sources including both AWS and Azure unless filtered by Type.
//	@Tags			onboard
//	@Produce		json
//	@Param			connector	query		source.Type	false	"filter by connector type"
//	@Param			pageSize	query		int			false	"page size"		default(50)
//	@Param			pageNumber	query		int			false	"page number"	default(1)
//	@Param			pageSize	query		int			false	"page size"		default(50)
//	@Param			pageNumber	query		int			false	"page number"	default(1)
//
//	@Success		200			{object}	[]api.Credential
//	@Router			/onboard/api/v1/credential/sources/list [get]
func (h HttpHandler) ListSourcesByCredentials(ctx echo.Context) error {
	sType, _ := source.ParseType(ctx.QueryParam("connector"))
	pageSizeStr := ctx.QueryParam("pageSize")
	pageNumberStr := ctx.QueryParam("pageNumber")
	pageSize := int64(50)
	pageNumber := int64(1)
	if pageSizeStr != "" {
		pageSize, _ = strconv.ParseInt(pageSizeStr, 10, 64)
	}
	if pageNumberStr != "" {
		pageNumber, _ = strconv.ParseInt(pageNumberStr, 10, 64)
	}
	var sources []Source
	var err error
	if sType != "" {
		sources, err = h.db.GetSourcesOfType(sType)
		if err != nil {
			return err
		}
	} else {
		sources, err = h.db.ListSources()
		if err != nil {
			return err
		}
	}

	credentials, err := h.db.GetCredentialsByFilters(sType, source.HealthStatusNil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	apiCredentials := make(map[string]api.Credential)
	for _, cred := range credentials {
		metadata := make(map[string]any)
		err = json.Unmarshal(cred.Metadata, &metadata)
		if err != nil {
			return err
		}
		apiCredentials[cred.ID.String()] = api.Credential{
			ID:                  cred.ID.String(),
			Name:                cred.Name,
			ConnectorType:       cred.ConnectorType,
			CredentialType:      cred.CredentialType,
			Enabled:             cred.Enabled,
			OnboardDate:         cred.CreatedAt,
			LastHealthCheckTime: cred.LastHealthCheckTime,
			HealthStatus:        cred.HealthStatus,
			HealthReason:        cred.HealthReason,
			Metadata:            metadata,
			Connections:         nil,
		}
	}

	for _, src := range sources {
		if v, ok := apiCredentials[src.CredentialID.String()]; ok {
			if v.Connections == nil {
				v.Connections = make([]api.Source, 0)
			}
			metadata := make(map[string]any)
			if src.Metadata.String() != "" {
				err := json.Unmarshal(src.Metadata, &metadata)
				if err != nil {
					return err
				}
			}

			v.Connections = append(v.Connections, api.Source{
				ID:                   src.ID,
				ConnectionID:         src.SourceId,
				ConnectionName:       src.Name,
				Email:                src.Email,
				Type:                 src.Type,
				Description:          src.Description,
				CredentialID:         src.CredentialID.String(),
				CredentialName:       src.Credential.Name,
				OnboardDate:          src.CreatedAt,
				LifecycleState:       api.ConnectionLifecycleState(src.LifecycleState),
				AssetDiscoveryMethod: src.AssetDiscoveryMethod,
				HealthState:          src.HealthState,
				LastHealthCheckTime:  src.LastHealthCheckTime,
				HealthReason:         src.HealthReason,
				Metadata:             metadata,
			})
			apiCredentials[src.CredentialID.String()] = v
			v.TotalConnections = utils.PAdd(v.TotalConnections, utils.GetPointer(1))
			if src.Enabled {
				v.EnabledConnections = utils.PAdd(v.EnabledConnections, utils.GetPointer(1))
			}
			if src.HealthState == source.HealthStatusUnhealthy {
				v.UnhealthyConnections = utils.PAdd(v.UnhealthyConnections, utils.GetPointer(1))
			}
		}
	}

	apiCredentialsList := make([]api.Credential, 0, len(apiCredentials))
	for _, v := range apiCredentials {
		if v.Connections == nil {
			continue
		}
		apiCredentialsList = append(apiCredentialsList, v)
	}

	sort.Slice(apiCredentialsList, func(i, j int) bool {
		return apiCredentialsList[i].OnboardDate.After(apiCredentialsList[j].OnboardDate)
	})

	return ctx.JSON(http.StatusOK, utils.Paginate(pageNumber, pageSize, apiCredentialsList))
}

func (h HttpHandler) putAzureCredentials(ctx echo.Context, req api.UpdateCredentialRequest) error {
	id, err := uuid.Parse(ctx.Param("credentialId"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, "invalid id")
	}

	cred, err := h.db.GetCredentialByID(id)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		return ctx.JSON(http.StatusNotFound, "credential not found")
	}

	if req.Name != nil {
		cred.Name = req.Name
	}

	config := api.SourceConfigAzure{}

	if req.Config != nil {
		configStr, err := json.Marshal(req.Config)
		if err != nil {
			return err
		}
		err = json.Unmarshal(configStr, &config)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "invalid config")
		}
		err = h.validator.Struct(config)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, err.Error())
		}

		azureCnf, err := describe.AzureSubscriptionConfigFromMap(config.AsMap())
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "invalid config")
		}

		metadata, err := getAzureCredentialsMetadata(ctx.Request().Context(), azureCnf)
		if err != nil {
			return err
		}
		jsonMetadata, err := json.Marshal(metadata)
		if err != nil {
			return err
		}
		cred.Metadata = jsonMetadata
	}

	err = h.db.orm.Transaction(func(tx *gorm.DB) error {
		secretBytes, err := h.kms.Encrypt(config.AsMap(), h.keyARN)
		if err != nil {
			return err
		}
		cred.Secret = string(secretBytes)

		if _, err := h.db.UpdateCredential(cred); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	_, err = h.checkCredentialHealth(*cred)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, struct{}{})
}

func (h HttpHandler) putAWSCredentials(ctx echo.Context, req api.UpdateCredentialRequest) error {
	id, err := uuid.Parse(ctx.Param("credentialId"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, "invalid id")
	}

	cred, err := h.db.GetCredentialByID(id)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		return ctx.JSON(http.StatusNotFound, "credential not found")
	}

	if req.Name != nil {
		cred.Name = req.Name
	}

	config := api.SourceConfigAWS{}
	if req.Config != nil {
		configStr, err := json.Marshal(req.Config)
		if err != nil {
			return err
		}

		err = json.Unmarshal(configStr, &config)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "invalid config")
		}
		err = h.validator.Struct(config)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, err.Error())
		}

		awsCnf, err := describe.AWSAccountConfigFromMap(config.AsMap())
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "invalid config")
		}

		metadata, err := getAWSCredentialsMetadata(ctx.Request().Context(), awsCnf)
		if err != nil {
			return err
		}
		jsonMetadata, err := json.Marshal(metadata)
		if err != nil {
			return err
		}
		cred.Metadata = jsonMetadata
	}
	err = h.db.orm.Transaction(func(tx *gorm.DB) error {
		secretBytes, err := h.kms.Encrypt(config.AsMap(), h.keyARN)
		if err != nil {
			return err
		}
		cred.Secret = string(secretBytes)

		if err := h.db.CreateCredential(cred); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	_, err = h.checkCredentialHealth(*cred)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, struct{}{})
}

// PutCredentials godoc
//
//	@Summary		Edit a credential by Id
//	@Description	Edit a credential by Id
//	@Tags			onboard
//	@Produce		json
//	@Success		200
//	@Param			config	body	api.UpdateCredentialRequest	true	"config"
//	@Router			/onboard/api/v1/credential/{credentialId} [put]
func (h HttpHandler) PutCredentials(ctx echo.Context) error {
	var req api.UpdateCredentialRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	switch req.Connector {
	case source.CloudAzure:
		return h.putAzureCredentials(ctx, req)
	case source.CloudAWS:
		return h.putAWSCredentials(ctx, req)
	}

	return ctx.JSON(http.StatusBadRequest, "invalid source type")
}

// DeleteCredential godoc
//
//	@Summary		Delete credential
//	@Description	Delete credential
//	@Tags			onboard
//	@Produce		json
//	@Success		200
//	@Router			/onboard/api/v1/credential/{credentialId} [delete]
func (h HttpHandler) DeleteCredential(ctx echo.Context) error {
	credId, err := uuid.Parse(ctx.Param(paramCredentialId))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	credential, err := h.db.GetCredentialByID(credId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "credential not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	sources, err := h.db.GetSourcesByCredentialID(credential.ID.String())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	err = h.db.orm.Transaction(func(tx *gorm.DB) error {
		if err := h.db.DeleteCredential(credential.ID); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		for _, src := range sources {
			if err := h.db.UpdateSourceEnabled(src.ID, false); err != nil {
				return err
			}

			if err := h.sourceEventsQueue.Publish(api.SourceEvent{
				Action:     api.SourceDeleted,
				SourceID:   src.ID,
				SourceType: src.Type,
				Secret:     src.Credential.Secret,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, struct{}{})
}

// DisableCredential godoc
//
//	@Summary		Disable credential
//	@Description	Disable credential
//	@Tags			onboard
//	@Produce		json
//	@Success		200
//	@Router			/onboard/api/v1/credential/{credentialId}/disable [post]
func (h HttpHandler) DisableCredential(ctx echo.Context) error {
	credId, err := uuid.Parse(ctx.Param(paramCredentialId))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	credential, err := h.db.GetCredentialByID(credId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "credential not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if !credential.Enabled {
		return echo.NewHTTPError(http.StatusBadRequest, "credential already disabled")
	}

	sources, err := h.db.GetSourcesByCredentialID(credential.ID.String())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	credential.Enabled = false
	err = h.db.orm.Transaction(func(tx *gorm.DB) error {
		if _, err := h.db.UpdateCredential(credential); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		for _, src := range sources {
			if err := h.db.UpdateSourceEnabled(src.ID, false); err != nil {
				return err
			}

			if err := h.sourceEventsQueue.Publish(api.SourceEvent{
				Action:     api.SourceDeleted,
				SourceID:   src.ID,
				SourceType: src.Type,
				Secret:     src.Credential.Secret,
			}); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, struct{}{})
}

// EnableCredential godoc
//
//	@Summary		Enable credential
//	@Description	Enable credential
//	@Tags			onboard
//	@Produce		json
//	@Success		200
//	@Router			/onboard/api/v1/credential/{credentialId}/enable [post]
func (h HttpHandler) EnableCredential(ctx echo.Context) error {
	credId, err := uuid.Parse(ctx.Param(paramCredentialId))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	credential, err := h.db.GetCredentialByID(credId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "credential not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if credential.Enabled {
		return echo.NewHTTPError(http.StatusBadRequest, "credential already enabled")
	}

	credential.Enabled = true
	if _, err := h.db.UpdateCredential(credential); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return ctx.JSON(http.StatusOK, struct{}{})
}

// GetSourceCred godoc
//
//	@Summary	Get source credential
//	@Tags		onboard
//	@Produce	json
//	@Param		sourceId	query	string	true	"Source ID"
//	@Router		/onboard/api/v1/source/{sourceId}/credentials [post]
func (h HttpHandler) GetSourceCred(ctx echo.Context) error {
	sourceUUID, err := uuid.Parse(ctx.Param("sourceId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
	}

	src, err := h.db.GetSource(sourceUUID)
	if err != nil {
		return err
	}

	cnf, err := h.kms.Decrypt(src.Credential.Secret, h.keyARN)
	if err != nil {
		return err
	}

	switch src.Type {
	case source.CloudAWS:
		awsCnf, err := describe.AWSAccountConfigFromMap(cnf)
		if err != nil {
			return err
		}
		return ctx.JSON(http.StatusOK, api.AWSCredential{
			AccessKey: awsCnf.AccessKey,
		})
	case source.CloudAzure:
		azureCnf, err := describe.AzureSubscriptionConfigFromMap(cnf)
		if err != nil {
			return err
		}
		return ctx.JSON(http.StatusOK, api.AzureCredential{
			ClientID: azureCnf.ClientID,
			TenantID: azureCnf.TenantID,
		})
	default:
		return errors.New("invalid provider")
	}
}

// GetSourceHealth godoc
//
//	@Summary	Get live source health status
//	@Tags		onboard
//	@Produce	json
//	@Param		sourceId	path	string	true	"Source ID"
//	@Router		/onboard/api/v1/source/{sourceId}/healthcheck [post]
func (h HttpHandler) GetSourceHealth(ctx echo.Context) error {
	sourceUUID, err := uuid.Parse(ctx.Param("sourceId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
	}

	src, err := h.db.GetSource(sourceUUID)
	if err != nil {
		return err
	}

	isHealthy, err := h.checkCredentialHealth(src.Credential)
	if err != nil {
		return err
	}
	if !isHealthy {
		return echo.NewHTTPError(http.StatusBadRequest, "credential is not healthy")
	}

	cnf, err := h.kms.Decrypt(src.Credential.Secret, h.keyARN)
	if err != nil {
		return err
	}

	var isAttached bool
	switch src.Type {
	case source.CloudAWS:
		awsCnf, err := describe.AWSAccountConfigFromMap(cnf)
		if err != nil {
			return err
		}
		isAttached, err = keibiaws.CheckAttachedPolicy(awsCnf.AccessKey, awsCnf.SecretKey, keibiaws.SecurityAuditPolicyARN)
		if err == nil && isAttached {
			cfg, err := keibiaws.GetConfig(context.Background(), awsCnf.AccessKey, awsCnf.SecretKey, "", "")
			if err != nil {
				return err
			}
			if cfg.Region == "" {
				cfg.Region = "us-east-1"
			}
			awsAccount, err := currentAwsAccount(ctx.Request().Context(), cfg)
			if err != nil {
				return err
			}
			metadata := NewAWSConnectionMetadata(*awsAccount)
			jsonMetadata, err := json.Marshal(metadata)
			if err != nil {
				return err
			}
			src.Metadata = jsonMetadata
		}
	case source.CloudAzure:
		azureCnf, err := describe.AzureSubscriptionConfigFromMap(cnf)
		if err != nil {
			return err
		}
		authCnf := keibiazure.AuthConfig{
			TenantID:            azureCnf.TenantID,
			ClientID:            azureCnf.ClientID,
			ObjectID:            azureCnf.ObjectID,
			SecretID:            azureCnf.SecretID,
			ClientSecret:        azureCnf.ClientSecret,
			CertificatePath:     azureCnf.CertificatePath,
			CertificatePassword: azureCnf.CertificatePass,
			Username:            azureCnf.Username,
			Password:            azureCnf.Password,
		}
		isAttached, err = keibiazure.CheckRole(authCnf, src.SourceId, keibiazure.DefaultReaderRoleDefinitionIDTemplate)

		if err == nil && isAttached {
			azSub, err := currentAzureSubscription(ctx.Request().Context(), src.SourceId, authCnf)
			metadata := NewAzureConnectionMetadata(*azSub)
			jsonMetadata, err := json.Marshal(metadata)
			if err != nil {
				return err
			}
			src.Metadata = jsonMetadata
		}
	}

	if !isAttached {
		src.HealthState = source.HealthStatusUnhealthy
		if err != nil {
			healthMessage := err.Error()
			src.HealthReason = &healthMessage
		} else {
			src.HealthReason = utils.GetPointer("Failed to find read permission")
		}
		src.LastHealthCheckTime = time.Now()
		src.UpdatedAt = time.Now()
		_, err = h.db.UpdateSource(&src)
		if err != nil {
			return err
		}
		//TODO Mahan: record state change in elastic search
	} else {
		src.HealthState = source.HealthStatusHealthy
		src.HealthReason = nil
		src.LastHealthCheckTime = time.Now()
		_, err = h.db.UpdateSource(&src)
		if err != nil {
			return err
		}
		//TODO Mahan: record state change in elastic search
	}

	metadata := make(map[string]any)
	if src.Metadata.String() != "" {
		err := json.Unmarshal(src.Metadata, &metadata)
		if err != nil {
			return err
		}
	}

	return ctx.JSON(http.StatusOK, &api.Source{
		ID:                   src.ID,
		ConnectionID:         src.SourceId,
		ConnectionName:       src.Name,
		Email:                src.Email,
		Type:                 src.Type,
		Description:          src.Description,
		CredentialID:         src.CredentialID.String(),
		CredentialName:       src.Credential.Name,
		OnboardDate:          src.CreatedAt,
		LifecycleState:       api.ConnectionLifecycleState(src.LifecycleState),
		AssetDiscoveryMethod: src.AssetDiscoveryMethod,
		HealthState:          src.HealthState,
		LastHealthCheckTime:  src.LastHealthCheckTime,
		HealthReason:         src.HealthReason,
		Metadata:             metadata,
	})
}

// PutSourceCred godoc
//
//	@Summary	Put source credential
//	@Tags		onboard
//	@Produce	json
//	@Param		sourceId	query	string	true	"Source ID"
//	@Router		/onboard/api/v1/source/{sourceId}/credentials [put]
func (h HttpHandler) PutSourceCred(ctx echo.Context) error {
	sourceUUID, err := uuid.Parse(ctx.Param("sourceId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
	}

	src, err := h.db.GetSource(sourceUUID)
	if err != nil {
		return err
	}

	cnf, err := h.kms.Decrypt(src.Credential.Secret, h.keyARN)
	if err != nil {
		return err
	}

	switch src.Type {
	case source.CloudAWS:
		awsCnf, err := describe.AWSAccountConfigFromMap(cnf)
		if err != nil {
			return err
		}

		var req api.AWSCredential
		if err := bindValidate(ctx, &req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
		}

		newCnf := api.SourceConfigAWS{
			AccountId: awsCnf.AccountID,
			Regions:   awsCnf.Regions,
			AccessKey: req.AccessKey,
			SecretKey: req.SecretKey,
		}

		isAttached, err := keibiaws.CheckAttachedPolicy(newCnf.AccessKey, newCnf.SecretKey, keibiaws.SecurityAuditPolicyARN)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if !isAttached {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to find read permission")
		}

		secretBytes, err := h.kms.Encrypt(newCnf.AsMap(), h.keyARN)
		if err != nil {
			return err
		}
		src.Credential.Secret = string(secretBytes)

		if _, err := h.db.UpdateSource(&src); err != nil {
			return err
		}
		return ctx.NoContent(http.StatusOK)
	case source.CloudAzure:
		azureCnf, err := describe.AzureSubscriptionConfigFromMap(cnf)
		if err != nil {
			return err
		}

		var req api.AzureCredential
		if err := bindValidate(ctx, &req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
		}

		newCnf := api.SourceConfigAzure{
			SubscriptionId: azureCnf.SubscriptionID,
			TenantId:       azureCnf.TenantID,
			ClientId:       req.ClientID,
			ClientSecret:   req.ClientSecret,
		}
		secretBytes, err := h.kms.Encrypt(newCnf.AsMap(), h.keyARN)
		if err != nil {
			return err
		}
		src.Credential.Secret = string(secretBytes)
		return ctx.NoContent(http.StatusOK)
	default:
		return errors.New("invalid provider")
	}
}

// GetSource godoc
//
//	@Summary		Returns a single source
//	@Description	Returning single source either AWS / Azure.
//	@Tags			onboard
//	@Produce		json
//	@Success		200			{object}	api.Source
//	@Param			sourceId	path		integer	true	"SourceID"
//	@Router			/onboard/api/v1/source/{sourceId} [get]
func (h HttpHandler) GetSource(ctx echo.Context) error {
	srcId, err := uuid.Parse(ctx.Param(paramSourceId))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	src, err := h.db.GetSource(srcId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusBadRequest, "source not found")
		}
		return err
	}

	metadata := make(map[string]any)
	if src.Metadata.String() != "" {
		err := json.Unmarshal(src.Metadata, &metadata)
		if err != nil {
			return err
		}
	}

	return ctx.JSON(http.StatusOK, &api.Source{
		ID:                   src.ID,
		ConnectionID:         src.SourceId,
		ConnectionName:       src.Name,
		Email:                src.Email,
		Type:                 src.Type,
		Description:          src.Description,
		CredentialID:         src.CredentialID.String(),
		CredentialName:       src.Credential.Name,
		OnboardDate:          src.CreatedAt,
		LifecycleState:       api.ConnectionLifecycleState(src.LifecycleState),
		AssetDiscoveryMethod: src.AssetDiscoveryMethod,
		HealthState:          src.HealthState,
		LastHealthCheckTime:  src.LastHealthCheckTime,
		HealthReason:         src.HealthReason,
		Metadata:             metadata,
	})
}

// DeleteSource godoc
//
//	@Summary		Delete a single source
//	@Description	Deleting a single source either AWS / Azure.
//	@Tags			onboard
//	@Produce		json
//	@Success		200
//	@Param			sourceId	path	integer	true	"SourceID"
//	@Router			/onboard/api/v1/source/{sourceId} [delete]
func (h HttpHandler) DeleteSource(ctx echo.Context) error {
	srcId, err := uuid.Parse(ctx.Param(paramSourceId))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	src, err := h.db.GetSource(srcId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusBadRequest, "source not found")
		}
		return err
	}

	err = h.db.orm.Transaction(func(tx *gorm.DB) error {
		if err := h.db.DeleteSource(srcId); err != nil {
			return err
		}

		if src.Credential.CredentialType == source.CredentialTypeAutoGenerated {
			err = h.db.DeleteCredential(src.Credential.ID)
			if err != nil {
				return err
			}
		}

		if err := h.sourceEventsQueue.Publish(api.SourceEvent{
			Action:     api.SourceDeleted,
			SourceID:   src.ID,
			SourceType: src.Type,
			Secret:     src.Credential.Secret,
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

// DisableSource godoc
//
//	@Summary	Disable a single source
//	@Tags		onboard
//	@Produce	json
//	@Success	200
//	@Param		sourceId	path	integer	true	"SourceID"
//	@Router		/onboard/api/v1/source/{sourceId}/disable [post]
func (h HttpHandler) DisableSource(ctx echo.Context) error {
	srcId, err := uuid.Parse(ctx.Param(paramSourceId))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	src, err := h.db.GetSource(srcId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusBadRequest, "source not found")
		}
		return err
	}

	err = h.db.orm.Transaction(func(tx *gorm.DB) error {
		if err := h.db.UpdateSourceEnabled(srcId, false); err != nil {
			return err
		}

		if err := h.sourceEventsQueue.Publish(api.SourceEvent{
			Action:     api.SourceDeleted,
			SourceID:   src.ID,
			SourceType: src.Type,
			Secret:     src.Credential.Secret,
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

// EnableSource godoc
//
//	@Summary	Enable a single source
//	@Tags		onboard
//	@Produce	json
//	@Success	200
//	@Param		sourceId	path	integer	true	"SourceID"
//	@Router		/onboard/api/v1/source/{sourceId}/enable [post]
func (h HttpHandler) EnableSource(ctx echo.Context) error {
	srcId, err := uuid.Parse(ctx.Param(paramSourceId))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	src, err := h.db.GetSource(srcId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusBadRequest, "source not found")
		}
		return err
	}

	err = h.db.orm.Transaction(func(tx *gorm.DB) error {
		if err := h.db.UpdateSourceEnabled(srcId, true); err != nil {
			return err
		}

		if err := h.sourceEventsQueue.Publish(api.SourceEvent{
			Action:     api.SourceCreated,
			SourceID:   src.ID,
			AccountID:  src.SourceId,
			SourceType: src.Type,
			Secret:     src.Credential.Secret,
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

// ListSources godoc
//
//	@Summary		Returns a list of sources
//	@Description	Returning a list of sources including both AWS and Azure unless filtered by Type.
//	@Tags			onboard
//	@Produce		json
//	@Param			connector	query		source.Type	false	"filter by source type"
//	@Success		200			{object}	api.GetSourcesResponse
//	@Router			/onboard/api/v1/sources [get]
func (h HttpHandler) ListSources(ctx echo.Context) error {
	sType := ctx.QueryParam("connector")
	var sources []Source
	if sType != "" {
		st, err := source.ParseType(sType)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid source type: %s", sType))
		}

		sources, err = h.db.GetSourcesOfType(st)
		if err != nil {
			return err
		}
	} else {
		var err error
		sources, err = h.db.ListSources()
		if err != nil {
			return err
		}
	}

	resp := api.GetSourcesResponse{}
	for _, s := range sources {
		metadata := make(map[string]any)
		if s.Metadata.String() != "" {
			err := json.Unmarshal(s.Metadata, &metadata)
			if err != nil {
				return err
			}
		}
		src := api.Source{
			ID:                   s.ID,
			ConnectionID:         s.SourceId,
			ConnectionName:       s.Name,
			Email:                s.Email,
			Type:                 s.Type,
			Description:          s.Description,
			CredentialID:         s.CredentialID.String(),
			CredentialName:       s.Credential.Name,
			OnboardDate:          s.CreatedAt,
			LifecycleState:       api.ConnectionLifecycleState(s.LifecycleState),
			AssetDiscoveryMethod: s.AssetDiscoveryMethod,
			HealthState:          s.HealthState,
			LastHealthCheckTime:  s.LastHealthCheckTime,
			HealthReason:         s.HealthReason,
			Metadata:             metadata,
		}
		resp = append(resp, src)
	}

	return ctx.JSON(http.StatusOK, resp)
}

// GetSources godoc
//
//	@Summary		Returns a list of sources
//	@Description	Returning a list of sources including both AWS and Azure unless filtered by Type.
//	@Tags			onboard
//	@Produce		json
//	@Param			type	query		string					false	"Type"	Enums(aws,azure)
//	@Param			request	body		api.GetSourcesRequest	false	"Request Body"
//	@Success		200		{object}	api.GetSourcesResponse
//	@Router			/onboard/api/v1/sources [post]
func (h HttpHandler) GetSources(ctx echo.Context) error {
	var req api.GetSourcesRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	var reqUUIDs []uuid.UUID
	for _, item := range req.SourceIDs {
		u, err := uuid.Parse(item)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid uuid:"+item)
		}
		reqUUIDs = append(reqUUIDs, u)
	}
	srcs, err := h.db.GetSources(reqUUIDs)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusBadRequest, "source not found")
		}
		return err
	}

	var res []api.Source
	for _, src := range srcs {
		metadata := make(map[string]any)
		if src.Metadata.String() != "" {
			err := json.Unmarshal(src.Metadata, &metadata)
			if err != nil {
				return err
			}
		}
		res = append(res, api.Source{
			ID:                   src.ID,
			ConnectionID:         src.SourceId,
			ConnectionName:       src.Name,
			Email:                src.Email,
			Type:                 src.Type,
			Description:          src.Description,
			CredentialID:         src.CredentialID.String(),
			CredentialName:       src.Credential.Name,
			OnboardDate:          src.CreatedAt,
			LifecycleState:       api.ConnectionLifecycleState(src.LifecycleState),
			AssetDiscoveryMethod: src.AssetDiscoveryMethod,
			HealthState:          src.HealthState,
			LastHealthCheckTime:  src.LastHealthCheckTime,
			HealthReason:         src.HealthReason,
			Metadata:             metadata,
		})
	}
	return ctx.JSON(http.StatusOK, res)
}

// CountSources godoc
//
//	@Summary		Returns a count of sources
//	@Description	Returning a count of sources including both AWS and Azure unless filtered by Type.
//	@Tags			onboard
//	@Produce		json
//	@Param			connector	query		source.Type	false	"filter by source type"
//	@Success		200			{object}	int64
//	@Router			/onboard/api/v1/sources/count [get]
func (h HttpHandler) CountSources(ctx echo.Context) error {
	sType := ctx.QueryParam("connector")
	var count int64
	if sType != "" {
		st, err := source.ParseType(sType)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid source type: %s", sType))
		}

		count, err = h.db.CountSourcesOfType(st)
		if err != nil {
			return err
		}
	} else {
		var err error
		count, err = h.db.CountSources()
		if err != nil {
			return err
		}
	}

	return ctx.JSON(http.StatusOK, count)
}

// CatalogMetrics godoc
//
//	@Summary	Returns the list of metrics for catalog page.
//	@Tags		onboard
//	@Produce	json
//	@Success	200	{object}	api.CatalogMetrics
//	@Router		/onboard/api/v1/catalog/metrics [get]
func (h HttpHandler) CatalogMetrics(ctx echo.Context) error {
	var metrics api.CatalogMetrics

	srcs, err := h.db.ListSources()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	for _, src := range srcs {
		metrics.TotalConnections++
		if src.Enabled {
			metrics.ConnectionsEnabled++
		}

		if src.HealthState == source.HealthStatusUnhealthy {
			metrics.UnhealthyConnections++
		} else {
			metrics.HealthyConnections++
		}
	}

	return ctx.JSON(http.StatusOK, metrics)
}

//go:embed api/catalogs.json
var catalogsJSON string

// CatalogConnectors godoc
//
//	@Summary	Returns the list of connectors for catalog page.
//	@Tags		onboard
//	@Produce	json
//	@Param		category		query		string	false	"Category filter"
//	@Param		state			query		string	false	"State filter"
//	@Param		minConnection	query		string	false	"Minimum connection filter"
//	@Param		id				query		string	false	"ID filter"
//	@Success	200				{object}	[]api.CatalogConnector
//	@Router		/onboard/api/v1/catalog/connectors [get]
func (h HttpHandler) CatalogConnectors(ctx echo.Context) error {
	categoryFilter := ctx.QueryParam("category")
	stateFilter := ctx.QueryParam("state")
	minConnectionFilter := ctx.QueryParam("minConnection")
	idFilter := ctx.QueryParam("id")

	var connectors []api.CatalogConnector
	if err := json.Unmarshal([]byte(catalogsJSON), &connectors); err != nil {
		return err
	}

	for idx, connector := range connectors {
		if !connector.SourceType.IsNull() {
			c, err := h.db.CountSourcesOfType(connector.SourceType)
			if err != nil {
				return err
			}

			connectors[idx].ConnectionCount = c
		}
	}

	var response []api.CatalogConnector
	for _, connector := range connectors {
		if len(categoryFilter) > 0 && connector.Category != categoryFilter {
			continue
		}
		if len(stateFilter) > 0 && connector.State != stateFilter {
			continue
		}
		if len(idFilter) > 0 {
			id, err := strconv.Atoi(idFilter)
			if err != nil {
				return err
			}

			if connector.ID != id {
				continue
			}
		}
		if len(minConnectionFilter) > 0 {
			minConnection, err := strconv.ParseInt(minConnectionFilter, 10, 64)
			if err != nil {
				return err
			}

			if connector.ConnectionCount < minConnection {
				continue
			}
		}
		response = append(response, connector)
	}

	return ctx.JSON(http.StatusOK, response)
}

// CountConnections godoc
//
//	@Summary	Returns a count of connections
//	@Tags		onboard
//	@Produce	json
//	@Param		type	body		api.ConnectionCountRequest	true	"Request"
//	@Success	200		{object}	int64
//	@Router		/onboard/api/v1/connections/count [get]
func (h HttpHandler) CountConnections(ctx echo.Context) error {
	var request api.ConnectionCountRequest
	if err := ctx.Bind(&request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("failed to parse request due to: %v", err))
	}

	var connectors []api.CatalogConnector
	if err := json.Unmarshal([]byte(catalogsJSON), &connectors); err != nil {
		return err
	}

	var condQuery []string
	var params []interface{}
	if request.ConnectorsNames != nil && len(request.ConnectorsNames) > 0 {
		var q []string
		for _, c := range request.ConnectorsNames {
			if len(strings.TrimSpace(c)) == 0 {
				continue
			}

			for _, connector := range connectors {
				if connector.SourceType.IsNull() {
					continue
				}

				if connector.Name == c {
					q = append(q, "?")
					params = append(params, connector.SourceType.String())
				}
			}
		}

		if len(q) > 0 {
			condQuery = append(condQuery, fmt.Sprintf("_type IN (%s)", strings.Join(q, ",")))
		}
	}

	if request.Health != nil {
		condQuery = append(condQuery, "health_state = ?")
		params = append(params, string(*request.Health))
	}

	if request.State != nil {
		condQuery = append(condQuery, "enabled = ?")
		if *request.State == api.ConnectionState_ENABLED {
			params = append(params, true)
		} else {
			params = append(params, false)
		}
	}

	query := strings.Join(condQuery, " AND ")
	count, err := h.db.CountSourcesWithFilters(query, params...)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, count)
}
