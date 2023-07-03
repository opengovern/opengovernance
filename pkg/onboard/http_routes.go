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

	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	"go.uber.org/zap"

	api3 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/labstack/echo/v4"

	keibiaws "github.com/kaytu-io/kaytu-aws-describer/aws"
	keibiazure "github.com/kaytu-io/kaytu-azure-describer/azure"

	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-engine/pkg/describe"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
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
	sourceApiGroup.GET("/:sourceId/credentials/full", httpserver.AuthorizeHandler(h.GetSourceFullCred, api3.KeibiAdminRole))
	sourceApiGroup.PUT("/:sourceId/credentials", httpserver.AuthorizeHandler(h.PutSourceCred, api3.EditorRole))
	sourceApiGroup.DELETE("/:sourceId", httpserver.AuthorizeHandler(h.DeleteSource, api3.EditorRole))
	sourceApiGroup.GET("/account/:accountId", httpserver.AuthorizeHandler(h.GetSourcesByAccount, api3.ViewerRole))

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
	connections.GET("/summary", httpserver.AuthorizeHandler(h.ListConnectionsSummaries, api3.ViewerRole))
	connections.GET("/summary/:connectionId", httpserver.AuthorizeHandler(h.GetConnectionSummary, api3.ViewerRole))
	connections.POST("/:connectionId/state", httpserver.AuthorizeHandler(h.ChangeConnectionLifecycleState, api3.EditorRole))
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
//	@Summary		List connectors
//	@Description	Returns list of all connectors
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200	{object}	[]api.ConnectorCount
//	@Router			/onboard/api/v1/connector [get]
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
//	@Description	Returns connector details by name
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Param			connectorName	path		string	true	"Connector name"
//	@Success		200				{object}	api.Connector
//	@Router			/onboard/api/v1/connector/{connectorName} [get]
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
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200		{object}	api.CreateSourceResponse
//	@Param			request	body		api.SourceAwsRequest	true	"Request"
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
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200		{object}	api.CreateSourceResponse
//	@Param			request	body		api.SourceAzureRequest	true	"Request"
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

	src := NewAzureConnectionWithCredentials(*azSub, source.SourceCreationMethodManual, req.Description, *cred)
	secretBytes, err := h.kms.Encrypt(req.Config.AsMap(), h.keyARN)
	if err != nil {
		return err
	}
	src.Credential.Secret = string(secretBytes)

	err = h.db.orm.Transaction(func(tx *gorm.DB) error {
		err = h.db.CreateSource(&src)
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
	secretBytes, err := h.kms.Encrypt(config.AsMap(), h.keyARN)
	if err != nil {
		return err
	}
	cred.Secret = string(secretBytes)

	err = h.db.orm.Transaction(func(tx *gorm.DB) error {
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
	if req.Name == "" {
		if metadata.OrganizationID != nil {
			req.Name = *metadata.OrganizationID
		} else {
			req.Name = metadata.AccountID
		}
	}
	cred, err := NewAWSCredential(req.Name, metadata)
	if err != nil {
		return err
	}
	secretBytes, err := h.kms.Encrypt(config.AsMap(), h.keyARN)
	if err != nil {
		return err
	}
	cred.Secret = string(secretBytes)

	err = h.db.orm.Transaction(func(tx *gorm.DB) error {
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
//	@Security		BearerToken
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
//	@Description	Retrieving list of credentials with their details
//	@Security		BearerToken
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
//	@Summary		Get Credential
//	@Description	Retrieving credential details by credential ID
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200				{object}	api.Credential
//	@Param			credentialId	path		string	true	"Credential ID"
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
		Connections:         make([]api.Connection, 0, len(connections)),
	}
	for _, conn := range connections {
		metadata := make(map[string]any)
		if conn.Metadata.String() != "" {
			err := json.Unmarshal(conn.Metadata, &metadata)
			if err != nil {
				return err
			}
		}

		apiCredential.Connections = append(apiCredential.Connections, api.Connection{
			ID:                   conn.ID,
			ConnectionID:         conn.SourceId,
			ConnectionName:       conn.Name,
			Email:                conn.Email,
			Connector:            conn.Type,
			Description:          conn.Description,
			CredentialID:         conn.CredentialID.String(),
			CredentialName:       conn.Credential.Name,
			OnboardDate:          conn.CreatedAt,
			LifecycleState:       api.ConnectionLifecycleState(conn.LifecycleState),
			AssetDiscoveryMethod: conn.AssetDiscoveryMethod,
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
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Param			credentialId	path		string	true	"CredentialID"
//	@Success		200				{object}	[]api.Connection
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

	onboardedSources := make([]api.Connection, 0)
	switch credential.ConnectorType {
	case source.CloudAzure:
		cnf, err := h.kms.Decrypt(credential.Secret, h.keyARN)
		if err != nil {
			return err
		}
		azureCnf, err := describe.AzureSubscriptionConfigFromMap(cnf)
		if err != nil {
			return err
		}
		h.logger.Info("discovering subscriptions", zap.String("credentialId", credential.ID.String()))
		subs, err := discoverAzureSubscriptions(ctx.Request().Context(), keibiazure.AuthConfig{
			TenantID:     azureCnf.TenantID,
			ObjectID:     azureCnf.ObjectID,
			SecretID:     azureCnf.SecretID,
			ClientID:     azureCnf.ClientID,
			ClientSecret: azureCnf.ClientSecret,
		})
		if err != nil {
			h.logger.Error("failed to discover subscriptions", zap.Error(err))
			return err
		}
		h.logger.Info("discovered subscriptions", zap.Int("count", len(subs)))

		existingConnections, err := h.db.GetSourcesOfType(credential.ConnectorType)
		if err != nil {
			return err
		}

		existingConnectionSubIDs := make([]string, 0, len(existingConnections))
		for _, conn := range existingConnections {
			existingConnectionSubIDs = append(existingConnectionSubIDs, conn.SourceId)
		}
		subsToOnboard := make([]azureSubscription, 0)
		for _, sub := range subs {
			if !utils.Includes(existingConnectionSubIDs, sub.SubscriptionID) {
				subsToOnboard = append(subsToOnboard, sub)
			}
		}

		h.logger.Info("onboarding subscriptions", zap.Int("count", len(subsToOnboard)))

		for _, sub := range subsToOnboard {
			h.logger.Info("onboarding subscription", zap.String("subscriptionId", sub.SubscriptionID))
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
				h.logger.Warn("failed to check role", zap.Error(err))
				continue
			}
			if !isAttached {
				h.logger.Warn("role not attached", zap.String("subscriptionId", sub.SubscriptionID))
				continue
			}

			src := NewAzureConnectionWithCredentials(
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

			onboardedSources = append(onboardedSources, api.Connection{
				ID:                   src.ID,
				ConnectionID:         src.SourceId,
				ConnectionName:       src.Name,
				Email:                src.Email,
				Connector:            src.Type,
				Description:          src.Description,
				CredentialID:         src.CredentialID.String(),
				CredentialName:       src.Credential.Name,
				OnboardDate:          src.CreatedAt,
				LifecycleState:       api.ConnectionLifecycleState(src.LifecycleState),
				AssetDiscoveryMethod: src.AssetDiscoveryMethod,
				LastHealthCheckTime:  src.LastHealthCheckTime,
				HealthReason:         src.HealthReason,
				Metadata:             metadata,
			})
		}
	case source.CloudAWS:
		cnf, err := h.kms.Decrypt(credential.Secret, h.keyARN)
		if err != nil {
			return err
		}
		awsCnf, err := describe.AWSAccountConfigFromMap(cnf)
		if err != nil {
			return err
		}
		cfg, err := keibiaws.GetConfig(
			ctx.Request().Context(),
			awsCnf.AccessKey,
			awsCnf.SecretKey,
			"",
			awsCnf.AssumeRoleARN)
		h.logger.Info("discovering accounts", zap.String("credentialId", credential.ID.String()))
		accounts, err := discoverAWSAccounts(ctx.Request().Context(), cfg)
		if err != nil {
			h.logger.Error("failed to discover accounts", zap.Error(err))
			return err
		}
		h.logger.Info("discovered accounts", zap.Int("count", len(accounts)))
		existingConnections, err := h.db.GetSourcesOfType(credential.ConnectorType)
		if err != nil {
			return err
		}
		existingConnectionAccountIDs := make([]string, 0, len(existingConnections))
		for _, conn := range existingConnections {
			existingConnectionAccountIDs = append(existingConnectionAccountIDs, conn.SourceId)
		}
		accountsToOnboard := make([]awsAccount, 0)
		for _, account := range accounts {
			if !utils.Includes(existingConnectionAccountIDs, account.AccountID) {
				accountsToOnboard = append(accountsToOnboard, account)
			}
		}

		// TODO add tag filter

		h.logger.Info("onboarding accounts", zap.Int("count", len(accountsToOnboard)))
		for _, account := range accountsToOnboard {
			h.logger.Info("onboarding account", zap.String("accountID", account.AccountID))
			count, err := h.db.CountSources()
			if err != nil {
				return err
			}
			if count >= httpserver.GetMaxConnections(ctx) {
				return echo.NewHTTPError(http.StatusBadRequest, "maximum number of connections reached")
			}

			src := NewAWSConnectionWithCredentials(
				account,
				source.SourceCreationMethodAutoOnboard,
				fmt.Sprintf("Auto onboarded account %s", account.AccountID),
				*credential,
			)

			err = h.db.orm.Transaction(func(tx *gorm.DB) error {
				err := h.db.CreateSource(&src)
				if err != nil {
					return err
				}

				//TODO: add enable account

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

			onboardedSources = append(onboardedSources, api.Connection{
				ID:                   src.ID,
				ConnectionID:         src.SourceId,
				ConnectionName:       src.Name,
				Email:                src.Email,
				Connector:            src.Type,
				Description:          src.Description,
				CredentialID:         src.CredentialID.String(),
				CredentialName:       src.Credential.Name,
				OnboardDate:          src.CreatedAt,
				LifecycleState:       api.ConnectionLifecycleState(src.LifecycleState),
				AssetDiscoveryMethod: src.AssetDiscoveryMethod,
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
//	@Security	BearerToken
//	@Produce	json
//	@Param		credentialId	path	string	true	"CredentialID"
//	@Success	200
//	@Router		/onboard/api/v1/credential/{credentialId}/healthcheck [get]
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
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Param			connector	query		source.Type	false	"filter by connector type"
//	@Param			pageSize	query		int			false	"page size"		default(50)
//	@Param			pageNumber	query		int			false	"page number"	default(1)
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
				v.Connections = make([]api.Connection, 0)
			}
			metadata := make(map[string]any)
			if src.Metadata.String() != "" {
				err := json.Unmarshal(src.Metadata, &metadata)
				if err != nil {
					return err
				}
			}

			v.Connections = append(v.Connections, api.Connection{
				ID:                   src.ID,
				ConnectionID:         src.SourceId,
				ConnectionName:       src.Name,
				Email:                src.Email,
				Connector:            src.Type,
				Description:          src.Description,
				CredentialID:         src.CredentialID.String(),
				CredentialName:       src.Credential.Name,
				OnboardDate:          src.CreatedAt,
				LifecycleState:       api.ConnectionLifecycleState(src.LifecycleState),
				AssetDiscoveryMethod: src.AssetDiscoveryMethod,
				LastHealthCheckTime:  src.LastHealthCheckTime,
				HealthReason:         src.HealthReason,
				Metadata:             metadata,
			})
			apiCredentials[src.CredentialID.String()] = v
			v.TotalConnections = utils.PAdd(v.TotalConnections, utils.GetPointer(1))
			if src.LifecycleState.IsEnabled() {
				v.EnabledConnections = utils.PAdd(v.EnabledConnections, utils.GetPointer(1))
			}
			if src.LifecycleState == ConnectionLifecycleStateUnhealthy {
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
	secretBytes, err := h.kms.Encrypt(config.AsMap(), h.keyARN)
	if err != nil {
		return err
	}
	cred.Secret = string(secretBytes)

	err = h.db.orm.Transaction(func(tx *gorm.DB) error {
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
	secretBytes, err := h.kms.Encrypt(config.AsMap(), h.keyARN)
	if err != nil {
		return err
	}
	cred.Secret = string(secretBytes)

	err = h.db.orm.Transaction(func(tx *gorm.DB) error {
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
//	@Summary		Edit credential
//	@Description	Edit a credential by Id
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200
//	@Param			credentialId	path	string						true	"Credential ID"
//	@Param			config			body	api.UpdateCredentialRequest	true	"config"
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
//	@Description	Remove a credential by Id
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200
//	@Param			credentialId	path	string	true	"CredentialID"
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
			if err := h.db.UpdateSourceLifecycleState(src.ID, ConnectionLifecycleStateNotOnboard); err != nil {
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
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200
//	@Param			credentialId	path	string	true	"CredentialID"
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
			if err := h.db.UpdateSourceLifecycleState(src.ID, ConnectionLifecycleStateNotOnboard); err != nil {
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
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200
//	@Param			credentialId	path	string	true	"CredentialID"
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
//	@Summary		Get source credential
//	@Description	Returns credential for a source with given source ID.
//	@Description	The responses are different for different source types.
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Param			sourceId	path		string	true	"Source ID"
//	@Success		200			{object}	api.AWSCredential
//	@Success		200			{object}	api.AzureCredential
//	@Router			/onboard/api/v1/source/{sourceId}/credentials [get]
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

func (h HttpHandler) GetSourceFullCred(ctx echo.Context) error {
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
			SecretKey: awsCnf.SecretKey,
		})
	case source.CloudAzure:
		azureCnf, err := describe.AzureSubscriptionConfigFromMap(cnf)
		if err != nil {
			return err
		}
		return ctx.JSON(http.StatusOK, api.AzureCredential{
			ClientID:     azureCnf.ClientID,
			TenantID:     azureCnf.TenantID,
			ClientSecret: azureCnf.ClientSecret,
		})
	default:
		return errors.New("invalid provider")
	}
}

func (h HttpHandler) updateConnectionHealth(connection Source, healthStatus source.HealthStatus, reason *string) (Source, error) {
	if connection.LifecycleState != ConnectionLifecycleStateNotOnboard && connection.LifecycleState != ConnectionLifecycleStateArchived {
		if healthStatus == source.HealthStatusUnhealthy {
			connection.LifecycleState = ConnectionLifecycleStateUnhealthy
		} else if healthStatus == source.HealthStatusHealthy {
			connection.LifecycleState = ConnectionLifecycleStateOnboard
		}
	}
	connection.HealthReason = reason
	connection.LastHealthCheckTime = time.Now()
	_, err := h.db.UpdateSource(&connection)
	if err != nil {
		return Source{}, err
	}
	//TODO Mahan: record state change in elastic search
	return connection, nil
}

// GetSourceHealth godoc
//
//	@Summary		Get source health
//	@Description	Get live source health status with given source ID.
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Param			sourceId	path		string	true	"Source ID"
//	@Success		200			{object}	api.Connection
//	@Router			/onboard/api/v1/source/{sourceId}/healthcheck [post]
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
		if herr, ok := err.(*echo.HTTPError); ok {
			if herr.Code == http.StatusInternalServerError {
				return herr
			}
		} else {
			return err
		}
	}
	if !isHealthy {
		src, err = h.updateConnectionHealth(src, source.HealthStatusUnhealthy, utils.GetPointer("Credential is not healthy"))
		if err != nil {
			h.logger.Error("failed to update source health", zap.Error(err))
			return err
		}
	} else {
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
				if err != nil {
					return err
				}
				metadata := NewAzureConnectionMetadata(*azSub)
				jsonMetadata, err := json.Marshal(metadata)
				if err != nil {
					return err
				}
				src.Metadata = jsonMetadata
			}
		}
		if err != nil {
			h.logger.Warn("failed to check read permission", zap.Error(err))
		}

		if !isAttached {
			healthMessage := err.Error()
			if err == nil {
				healthMessage = "Failed to find read permission"
			}
			src, err = h.updateConnectionHealth(src, source.HealthStatusUnhealthy, &healthMessage)
			if err != nil {
				h.logger.Warn("failed to update source health", zap.Error(err))
				return err
			}
		} else {
			src, err = h.updateConnectionHealth(src, source.HealthStatusHealthy, nil)
			if err != nil {
				h.logger.Warn("failed to update source health", zap.Error(err))
				return err
			}
		}
	}

	metadata := make(map[string]any)
	if src.Metadata.String() != "" {
		err := json.Unmarshal(src.Metadata, &metadata)
		if err != nil {
			return err
		}
	}

	return ctx.JSON(http.StatusOK, &api.Connection{
		ID:                   src.ID,
		ConnectionID:         src.SourceId,
		ConnectionName:       src.Name,
		Email:                src.Email,
		Connector:            src.Type,
		Description:          src.Description,
		CredentialID:         src.CredentialID.String(),
		CredentialName:       src.Credential.Name,
		OnboardDate:          src.CreatedAt,
		LifecycleState:       api.ConnectionLifecycleState(src.LifecycleState),
		AssetDiscoveryMethod: src.AssetDiscoveryMethod,
		LastHealthCheckTime:  src.LastHealthCheckTime,
		HealthReason:         src.HealthReason,
		Metadata:             metadata,
	})
}

// PutSourceCred godoc
//
//	@Summary		Update source credential
//	@Description	Update source credential
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Param			sourceId	path	string	true	"Source ID"
//	@Success		200
//	@Router			/onboard/api/v1/source/{sourceId}/credentials [put]
func (h HttpHandler) PutSourceCred(ctx echo.Context) error {
	sourceUUID, err := uuid.Parse(ctx.Param("sourceId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
	}

	src, err := h.db.GetSource(sourceUUID)
	if err != nil {
		return err
	}

	switch src.Type {
	case source.CloudAWS:
		var req api.SourceConfigAWS
		if err := bindValidate(ctx, &req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
		}

		req.AccountId = src.SourceId

		isAttached, err := keibiaws.CheckAttachedPolicy(req.AccessKey, req.SecretKey, keibiaws.SecurityAuditPolicyARN)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if !isAttached {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to find read permission")
		}

		secretBytes, err := h.kms.Encrypt(req.AsMap(), h.keyARN)
		if err != nil {
			return err
		}
		src.Credential.Secret = string(secretBytes)

		if _, err := h.db.UpdateCredential(&src.Credential); err != nil {
			return err
		}
		return ctx.NoContent(http.StatusOK)
	case source.CloudAzure:
		var req api.SourceConfigAzure
		if err := bindValidate(ctx, &req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
		}

		req.SubscriptionId = src.SourceId
		secretBytes, err := h.kms.Encrypt(req.AsMap(), h.keyARN)
		if err != nil {
			return err
		}
		src.Credential.Secret = string(secretBytes)
		if _, err := h.db.UpdateCredential(&src.Credential); err != nil {
			return err
		}

		return ctx.NoContent(http.StatusOK)
	default:
		return errors.New("invalid provider")
	}
}

// GetSource godoc
//
//	@Summary		Get source
//	@Description	Returning single source either AWS / Azure.
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200			{object}	api.Connection
//	@Param			sourceId	path		integer	true	"Source ID"
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

	return ctx.JSON(http.StatusOK, &api.Connection{
		ID:                   src.ID,
		ConnectionID:         src.SourceId,
		ConnectionName:       src.Name,
		Email:                src.Email,
		Connector:            src.Type,
		Description:          src.Description,
		CredentialID:         src.CredentialID.String(),
		CredentialName:       src.Credential.Name,
		OnboardDate:          src.CreatedAt,
		LifecycleState:       api.ConnectionLifecycleState(src.LifecycleState),
		AssetDiscoveryMethod: src.AssetDiscoveryMethod,
		LastHealthCheckTime:  src.LastHealthCheckTime,
		HealthReason:         src.HealthReason,
		Metadata:             metadata,
	})
}

// DeleteSource godoc
//
//	@Summary		Delete source
//	@Description	Deleting a single source either AWS / Azure for the given source id.
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200
//	@Param			sourceId	path	integer	true	"Source ID"
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

// ChangeConnectionLifecycleState godoc
//
//	@Summary		Enable a single source
//	@Description	Enabling a single source either with connection ID.
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200
//	@Param			connectionId	path	integer										true	"ConnectionID"
//	@Param			request			body	api.ChangeConnectionLifecycleStateRequest	true	"Request"
//	@Router			/onboard/api/v1/connections/{connectionId}/state [post]
func (h HttpHandler) ChangeConnectionLifecycleState(ctx echo.Context) error {
	connectionId, err := uuid.Parse(ctx.Param("connectionId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid connection id")
	}

	var req api.ChangeConnectionLifecycleStateRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	} else if err = req.State.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	connection, err := h.db.GetSource(connectionId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusBadRequest, "source not found")
		}
		return err
	}

	reqState := ConnectionLifecycleStateFromApi(req.State)
	if reqState == connection.LifecycleState {
		return echo.NewHTTPError(http.StatusBadRequest, "connection already in requested state")
	}

	if reqState.IsEnabled() != connection.LifecycleState.IsEnabled() {
		err = h.db.orm.Transaction(func(tx *gorm.DB) error {
			if err := h.db.UpdateSourceLifecycleState(connectionId, reqState); err != nil {
				return err
			}

			action := api.SourceDeleted
			if reqState.IsEnabled() {
				action = api.SourceCreated
			}

			if err := h.sourceEventsQueue.Publish(api.SourceEvent{
				Action:     action,
				SourceID:   connection.ID,
				AccountID:  connection.SourceId,
				SourceType: connection.Type,
				Secret:     connection.Credential.Secret,
			}); err != nil {
				return err
			}
			return nil
		})
	} else {
		err = h.db.UpdateSourceLifecycleState(connectionId, reqState)
	}
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

// ListSources godoc
//
//	@Summary		List all sources
//	@Description	Returning a list of sources including both AWS and Azure unless filtered by Type.
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Param			connector	query		[]source.Type	false	"filter by source type"
//	@Success		200			{object}	api.GetSourcesResponse
//	@Router			/onboard/api/v1/sources [get]
func (h HttpHandler) ListSources(ctx echo.Context) error {
	var err error
	sType := httpserver.QueryArrayParam(ctx, "connector")
	var sources []Source
	if len(sType) > 0 {
		st := source.ParseTypes(sType)
		sources, err = h.db.GetSourcesOfTypes(st)
		if err != nil {
			return err
		}
	} else {
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
		src := api.Connection{
			ID:                   s.ID,
			ConnectionID:         s.SourceId,
			ConnectionName:       s.Name,
			Email:                s.Email,
			Connector:            s.Type,
			Description:          s.Description,
			CredentialID:         s.CredentialID.String(),
			CredentialName:       s.Credential.Name,
			OnboardDate:          s.CreatedAt,
			LifecycleState:       api.ConnectionLifecycleState(s.LifecycleState),
			AssetDiscoveryMethod: s.AssetDiscoveryMethod,
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
//	@Summary		Get filtered sources
//	@Description	Returning a list of sources including both AWS and Azure unless filtered by Type.
//	@Security		BearerToken
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

	var res []api.Connection
	for _, src := range srcs {
		metadata := make(map[string]any)
		if src.Metadata.String() != "" {
			err := json.Unmarshal(src.Metadata, &metadata)
			if err != nil {
				return err
			}
		}
		res = append(res, api.Connection{
			ID:                   src.ID,
			ConnectionID:         src.SourceId,
			ConnectionName:       src.Name,
			Email:                src.Email,
			Connector:            src.Type,
			Description:          src.Description,
			CredentialID:         src.CredentialID.String(),
			CredentialName:       src.Credential.Name,
			OnboardDate:          src.CreatedAt,
			LifecycleState:       api.ConnectionLifecycleState(src.LifecycleState),
			AssetDiscoveryMethod: src.AssetDiscoveryMethod,
			LastHealthCheckTime:  src.LastHealthCheckTime,
			HealthReason:         src.HealthReason,
			Metadata:             metadata,
		})
	}
	return ctx.JSON(http.StatusOK, res)
}

// GetSourcesByAccount godoc
//
//	@Summary		List account sources
//	@Description	Returning account source either AWS / Azure.
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200			{object}	api.Connection
//	@Param			account_id	path		integer	true	"Source ID"
//	@Router			/onboard/api/v1/source/account/{account_id} [get]
func (h HttpHandler) GetSourcesByAccount(ctx echo.Context) error {
	accId := ctx.Param("accountId")

	src, err := h.db.GetSourceBySourceID(accId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusBadRequest, "source not found")
		}
		return err
	}

	return ctx.JSON(http.StatusOK, src.toApi())
}

// CountSources godoc
//
//	@Summary		Count sources
//	@Description	Returning number of sources including both AWS and Azure unless filtered by Type.
//	@Security		BearerToken
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
//	@Summary		Get catalog metrics
//	@Description	Returns the list of metrics for catalog page.
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200	{object}	api.CatalogMetrics
//	@Router			/onboard/api/v1/catalog/metrics [get]
func (h HttpHandler) CatalogMetrics(ctx echo.Context) error {
	var metrics api.CatalogMetrics

	srcs, err := h.db.ListSources()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	for _, src := range srcs {
		metrics.TotalConnections++
		if src.LifecycleState.IsEnabled() {
			metrics.ConnectionsEnabled++
		}

		if src.LifecycleState == ConnectionLifecycleStateUnhealthy {
			metrics.UnhealthyConnections++
		} else if src.LifecycleState == ConnectionLifecycleStateOnboard {
			metrics.HealthyConnections++
		}
	}

	return ctx.JSON(http.StatusOK, metrics)
}

//go:embed api/catalogs.json
var catalogsJSON string

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
//	@Summary		Connections count
//	@Description	Returns a count of connections
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Param			type	body		api.ConnectionCountRequest	true	"Request"
//	@Success		200		{object}	int64
//	@Router			/onboard/api/v1/connections/count [get]
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

	if request.State != nil {
		condQuery = append(condQuery, "lifecycle_state = ?")
		params = append(params, string(*request.State))
	}

	query := strings.Join(condQuery, " AND ")
	count, err := h.db.CountSourcesWithFilters(query, params...)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, count)
}

// ListConnectionsSummaries godoc
//
//	@Summary		List connections summaries
//	@Description	Returns a list of connections summaries
//	@Security		BearerToken
//	@Tags			connections
//	@Accept			json
//	@Produce		json
//	@Param			connector		query		[]source.Type	true	"Connector"
//	@Param			connectionId	query		[]string		false	"Connection IDs"
//	@Param			lifecycleState	query		string			false	"lifecycle state filter"
//	@Param			pageSize		query		int				false	"page size - default is 20"
//	@Param			pageNumber		query		int				false	"page number - default is 1"
//	@Param			startTime		query		int				false	"start time in unix seconds"
//	@Param			endTime			query		int				false	"end time in unix seconds"
//	@Param			sortBy			query		string			false	"column to sort by - default is cost"	Enums(onboard_date,resource_count,cost,growth,growth_rate,cost_growth,cost_growth_rate)
//	@Success		200				{object}	api.ListConnectionSummaryResponse
//	@Router			/onboard/api/v1/connections/summary [get]
func (h HttpHandler) ListConnectionsSummaries(ctx echo.Context) error {
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now()
	if endTimeStr != "" {
		endTimeUnix, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "endTime is not a valid integer")
		}
		endTime = time.Unix(endTimeUnix, 0)
	}
	startTimeStr := ctx.QueryParam("startTime")
	startTime := endTime.AddDate(0, 0, -7)
	if startTimeStr != "" {
		startTimeUnix, err := strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "startTime is not a valid integer")
		}
		startTime = time.Unix(startTimeUnix, 0)
	}
	pageNumber, pageSize, err := utils.PageConfigFromStrings(ctx.QueryParam("pageNumber"), ctx.QueryParam("pageSize"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	sortBy := strings.ToLower(ctx.QueryParam("sortBy"))
	if sortBy == "" {
		sortBy = "cost"
	}

	if sortBy != "cost" && sortBy != "growth" &&
		sortBy != "growth_rate" && sortBy != "cost_growth" &&
		sortBy != "cost_growth_rate" && sortBy != "onboard_date" &&
		sortBy != "resource_count" {
		return ctx.JSON(http.StatusBadRequest, "sortBy is not a valid value")
	}
	var lifecycleStateSlice []ConnectionLifecycleState
	lifecycleState := ctx.QueryParam("lifecycleState")
	if lifecycleState != "" {
		lifecycleStateSlice = append(lifecycleStateSlice, ConnectionLifecycleState(lifecycleState))
	}

	connections, err := h.db.ListSourcesWithFilters(connectors, connectionIDs, lifecycleStateSlice)
	if err != nil {
		return err
	}

	connectionData, err := h.inventoryClient.ListConnectionsData(httpclient.FromEchoContext(ctx), nil, &startTime, &endTime)
	if err != nil {
		return err
	}

	result := api.ListConnectionSummaryResponse{
		ConnectionCount:       len(connections),
		OldConnectionCount:    0,
		TotalCost:             0,
		TotalResourceCount:    0,
		TotalOldResourceCount: 0,
		TotalUnhealthyCount:   0,
		TotalDisabledCount:    0,
		Connections:           make([]api.Connection, 0, len(connections)),
	}

	for _, connection := range connections {
		if data, ok := connectionData[connection.ID.String()]; ok {
			localData := data
			apiConn := connection.toApi()
			apiConn.Cost = localData.TotalCost
			apiConn.DailyCostAtStartTime = localData.DailyCostAtStartTime
			apiConn.DailyCostAtEndTime = localData.DailyCostAtEndTime
			apiConn.ResourceCount = localData.Count
			apiConn.OldResourceCount = localData.OldCount
			apiConn.LastInventory = localData.LastInventory
			if localData.TotalCost != nil {
				result.TotalCost += *localData.TotalCost
			}
			if localData.Count != nil {
				result.TotalResourceCount += *localData.Count
			}
			if localData.OldCount != nil {
				result.OldConnectionCount++
				result.TotalOldResourceCount += *localData.OldCount
			}

			switch connection.LifecycleState {
			case ConnectionLifecycleStateNotOnboard:
				result.TotalDisabledCount++
			case ConnectionLifecycleStateUnhealthy:
				result.TotalUnhealthyCount++

			}
			result.Connections = append(result.Connections, apiConn)
		} else {
			result.Connections = append(result.Connections, connection.toApi())
		}
	}

	sort.Slice(result.Connections, func(i, j int) bool {
		switch sortBy {
		case "onboard_date":
			return result.Connections[i].OnboardDate.Before(result.Connections[j].OnboardDate)
		case "resource_count":
			if result.Connections[i].ResourceCount == nil && result.Connections[j].ResourceCount == nil {
				break
			}
			if result.Connections[i].ResourceCount == nil {
				return false
			}
			if result.Connections[j].ResourceCount == nil {
				return true
			}
			if *result.Connections[i].ResourceCount != *result.Connections[j].ResourceCount {
				return *result.Connections[i].ResourceCount > *result.Connections[j].ResourceCount
			}
		case "cost":
			if result.Connections[i].Cost == nil && result.Connections[j].Cost == nil {
				break
			}
			if result.Connections[i].Cost == nil {
				return false
			}
			if result.Connections[j].Cost == nil {
				return true
			}
			if *result.Connections[i].Cost != *result.Connections[j].Cost {
				return *result.Connections[i].Cost > *result.Connections[j].Cost
			}
		case "growth":
			diffi := utils.PSub(result.Connections[i].ResourceCount, result.Connections[i].OldResourceCount)
			diffj := utils.PSub(result.Connections[j].ResourceCount, result.Connections[j].OldResourceCount)
			if diffi == nil && diffj == nil {
				break
			}
			if diffi == nil {
				return false
			}
			if diffj == nil {
				return true
			}
			if *diffi != *diffj {
				return *diffi > *diffj
			}
		case "growth_rate":
			diffi := utils.PSub(result.Connections[i].ResourceCount, result.Connections[i].OldResourceCount)
			diffj := utils.PSub(result.Connections[j].ResourceCount, result.Connections[j].OldResourceCount)
			if diffi == nil && diffj == nil {
				break
			}
			if diffi == nil {
				return false
			}
			if diffj == nil {
				return true
			}
			if result.Connections[i].OldResourceCount == nil && result.Connections[j].OldResourceCount == nil {
				break
			}
			if result.Connections[i].OldResourceCount == nil {
				return true
			}
			if result.Connections[j].OldResourceCount == nil {
				return false
			}
			if *result.Connections[i].OldResourceCount == 0 && *result.Connections[j].OldResourceCount == 0 {
				break
			}
			if *result.Connections[i].OldResourceCount == 0 {
				return false
			}
			if *result.Connections[j].OldResourceCount == 0 {
				return true
			}
			if float64(*diffi)/float64(*result.Connections[i].OldResourceCount) != float64(*diffj)/float64(*result.Connections[j].OldResourceCount) {
				return float64(*diffi)/float64(*result.Connections[i].OldResourceCount) > float64(*diffj)/float64(*result.Connections[j].OldResourceCount)
			}
		case "cost_growth":
			diffi := utils.PSub(result.Connections[i].DailyCostAtEndTime, result.Connections[i].DailyCostAtStartTime)
			diffj := utils.PSub(result.Connections[j].DailyCostAtEndTime, result.Connections[j].DailyCostAtStartTime)
			if diffi == nil && diffj == nil {
				break
			}
			if diffi == nil {
				return false
			}
			if diffj == nil {
				return true
			}
			if *diffi != *diffj {
				return *diffi > *diffj
			}
		case "cost_growth_rate":
			diffi := utils.PSub(result.Connections[i].DailyCostAtEndTime, result.Connections[i].DailyCostAtStartTime)
			diffj := utils.PSub(result.Connections[j].DailyCostAtEndTime, result.Connections[j].DailyCostAtStartTime)
			if diffi == nil && diffj == nil {
				break
			}
			if diffi == nil {
				return false
			}
			if diffj == nil {
				return true
			}
			if result.Connections[i].DailyCostAtStartTime == nil && result.Connections[j].DailyCostAtStartTime == nil {
				break
			}
			if result.Connections[i].DailyCostAtStartTime == nil {
				return true
			}
			if result.Connections[j].DailyCostAtStartTime == nil {
				return false
			}
			if *result.Connections[i].DailyCostAtStartTime == 0 && *result.Connections[j].DailyCostAtStartTime == 0 {
				break
			}
			if *result.Connections[i].DailyCostAtStartTime == 0 {
				return false
			}
			if *result.Connections[j].DailyCostAtStartTime == 0 {
				return true
			}
			if *diffi/(*result.Connections[i].DailyCostAtStartTime) != *diffj/(*result.Connections[j].DailyCostAtStartTime) {
				return *diffi/(*result.Connections[i].DailyCostAtStartTime) > *diffj/(*result.Connections[j].DailyCostAtStartTime)
			}
		}
		return result.Connections[i].ConnectionName < result.Connections[j].ConnectionName
	})

	result.Connections = utils.Paginate(pageNumber, pageSize, result.Connections)
	return ctx.JSON(http.StatusOK, result)
}

// GetConnectionSummary godoc
//
//	@Summary		Get connection summary
//	@Description	Returns a connections summaries
//	@Security		BearerToken
//	@Tags			connections
//	@Accept			json
//	@Produce		json
//	@Param			startTime		query		int		false	"start time in unix seconds"
//	@Param			endTime			query		int		false	"end time in unix seconds"
//	@Param			connectionId	path		string	true	"ConnectionID"
//	@Success		200				{object}	api.Connection
//	@Router			/onboard/api/v1/connections/summary/{connectionId} [get]
func (h HttpHandler) GetConnectionSummary(ctx echo.Context) error {
	connectionId, err := uuid.Parse(ctx.Param("connectionId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid connectionId uuid")
	}
	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now()
	if endTimeStr != "" {
		endTimeUnix, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "endTime is not a valid integer")
		}
		endTime = time.Unix(endTimeUnix, 0)
	}
	startTimeStr := ctx.QueryParam("startTime")
	startTime := endTime.AddDate(0, 0, -7)
	if startTimeStr != "" {
		startTimeUnix, err := strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "startTime is not a valid integer")
		}
		startTime = time.Unix(startTimeUnix, 0)
	}

	connection, err := h.db.GetSource(connectionId)
	if err != nil {
		return err
	}

	connectionData, err := h.inventoryClient.GetConnectionData(httpclient.FromEchoContext(ctx), connectionId.String(), &startTime, &endTime)
	if err != nil {
		return err
	}

	result := connection.toApi()
	result.Cost = connectionData.TotalCost
	result.ResourceCount = connectionData.Count
	result.OldResourceCount = connectionData.OldCount
	result.LastInventory = connectionData.LastInventory

	return ctx.JSON(http.StatusOK, result)
}
