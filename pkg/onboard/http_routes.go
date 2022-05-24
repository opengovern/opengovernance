package onboard

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe"
	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"
	"gorm.io/gorm"
)

const (
	paramSourceId = "sourceId"
)

func (h HttpHandler) Register(r *echo.Echo) {
	v1 := r.Group("/api/v1")

	source := v1.Group("/source")

	source.POST("/aws", h.PostSourceAws)
	source.POST("/azure", h.PostSourceAzure)
	source.GET("/:sourceId", h.GetSource)
	source.GET("/:sourceId/credentials", h.GetSourceCred)
	source.PUT("/:sourceId/credentials", h.PutSourceCred)
	source.PUT("/:sourceId", h.PutSource)
	source.DELETE("/:sourceId", h.DeleteSource)

	v1.GET("/sources", h.GetSources)
	v1.GET("/sources/count", h.CountSources)

	disc := v1.Group("/discover")

	disc.POST("/aws/accounts", h.DiscoverAwsAccounts)
	disc.POST("/azure/subscriptions", h.DiscoverAzureSubscriptions)

	v1.GET("/providers", h.GetProviders)
	v1.GET("/providers/types", h.GetProviderTypes)
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
// @Summary      Get providers
// @Description  Getting cloud providers
// @Tags     onboard
// @Produce  json
// @Success      200  {object}  api.ProvidersResponse
// @Router       /onboard/api/v1/providers [get]
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

// GetProviderTypes godoc
// @Summary      Get provider types
// @Description  Getting provider types
// @Tags     onboard
// @Produce  json
// @Success      200  {object}  api.ProviderTypesResponse
// @Router       /onboard/api/v1/providers/types [get]
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
// @Summary      Create AWS source
// @Description  Creating AWS source
// @Tags         onboard
// @Produce      json
// @Success      200          {object}  api.CreateSourceResponse
// @Param        name         body      string               true  "name"
// @Param        description  body      string               true  "description"
// @Param        config       body      api.SourceConfigAWS  true  "config"
// @Router       /onboard/api/v1/source/aws [post]
func (h HttpHandler) PostSourceAws(ctx echo.Context) error {
	var req api.SourceAwsRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	src := NewAWSSource(req)

	err := h.db.orm.Transaction(func(tx *gorm.DB) error {
		err := h.db.CreateSource(&src)
		if err != nil {
			return err
		}

		// TODO: Handle edge case where writing to Vault succeeds and writing to event queue fails.
		if err := h.vault.Write(src.ConfigRef, req.Config.AsMap()); err != nil {
			return err
		}

		err = h.sourceEventsQueue.Publish(api.SourceEvent{
			Action:     api.SourceCreated,
			SourceID:   src.ID,
			SourceType: src.Type,
			ConfigRef:  src.ConfigRef,
		})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, src.toSourceResponse())
}

// PostSourceAzure godoc
// @Summary      Create Azure source
// @Description  Creating Azure source
// @Tags         onboard
// @Produce      json
// @Success      200          {object}  api.CreateSourceResponse
// @Param        name         body      string                 true  "name"
// @Param        description  body      string                 true  "description"
// @Param        config       body      api.SourceConfigAzure  true  "config"
// @Router       /onboard/api/v1/source/azure [post]
func (h HttpHandler) PostSourceAzure(ctx echo.Context) error {
	var req api.SourceAzureRequest

	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	src := NewAzureSource(req)

	//verify spn exist
	err := h.db.orm.Transaction(func(tx *gorm.DB) error {
		err := h.db.CreateSource(&src)
		if err != nil {
			return err
		}

		// TODO: Handle edge case where writing to Vault succeeds and writing to event queue fails.
		if err := h.vault.Write(src.ConfigRef, req.Config.AsMap()); err != nil {
			return err
		}

		err = h.sourceEventsQueue.Publish(api.SourceEvent{
			Action:     api.SourceCreated,
			SourceID:   src.ID,
			SourceType: src.Type,
			ConfigRef:  src.ConfigRef,
		})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, src.toSourceResponse())
}

// GetSourceCred godoc
// @Summary  Get source credential
// @Tags         onboard
// @Produce      json
// @Param    sourceId  query  string  true  "Source ID"
// @Router   /onboard/api/v1/{sourceId}/credentials [post]
func (h HttpHandler) GetSourceCred(ctx echo.Context) error {
	sourceUUID, err := uuid.Parse(ctx.Param("sourceId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
	}

	src, err := h.db.GetSource(sourceUUID)
	if err != nil {
		return err
	}

	cnf, err := h.vault.Read(src.ConfigRef)
	if err != nil {
		return err
	}

	switch src.Type {
	case api.SourceCloudAWS:
		awsCnf, err := describe.AWSAccountConfigFromMap(cnf)
		if err != nil {
			return err
		}
		return ctx.JSON(http.StatusOK, api.AWSCredential{
			AccessKey: awsCnf.AccessKey,
		})
	case api.SourceCloudAzure:
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

// PutSourceCred godoc
// @Summary  Put source credential
// @Tags         onboard
// @Produce      json
// @Param    sourceId  query  string  true  "Source ID"
// @Router   /onboard/api/v1/{sourceId}/credentials [post]
func (h HttpHandler) PutSourceCred(ctx echo.Context) error {
	sourceUUID, err := uuid.Parse(ctx.Param("sourceId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
	}

	src, err := h.db.GetSource(sourceUUID)
	if err != nil {
		return err
	}

	cnf, err := h.vault.Read(src.ConfigRef)
	if err != nil {
		return err
	}

	switch src.Type {
	case api.SourceCloudAWS:
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
			AccessKey: awsCnf.AccessKey,
			SecretKey: req.SecretKey,
		}
		if err := h.vault.Write(src.ConfigRef, newCnf.AsMap()); err != nil {
			return err
		}
		return ctx.NoContent(http.StatusOK)
	case api.SourceCloudAzure:
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
			ClientId:       azureCnf.ClientID,
			ClientSecret:   req.ClientSecret,
		}
		if err := h.vault.Write(src.ConfigRef, newCnf.AsMap()); err != nil {
			return err
		}
		return ctx.NoContent(http.StatusOK)
	default:
		return errors.New("invalid provider")
	}
}

// GetSource godoc
// @Summary      Returns a single source
// @Description  Returning single source either AWS / Azure.
// @Tags         onboard
// @Produce      json
// @Success      200       {object}  api.Source
// @Param        sourceId  path      integer  true  "SourceID"
// @Router       /onboard/api/v1/source/{sourceId} [get]
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

	return ctx.JSON(http.StatusOK, &api.Source{
		ID:          src.ID,
		SourceId:    src.SourceId,
		Name:        src.Name,
		Type:        src.Type,
		Description: src.Description,
	})
}

// DeleteSource godoc
// @Summary      Delete a single source
// @Description  Deleting a single source either AWS / Azure.
// @Tags         onboard
// @Produce      json
// @Success      200
// @Param        sourceId  path  integer  true  "SourceID"
// @Router       /onboard/api/v1/source/{sourceId} [delete]
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

		// TODO: Handle edge case where deleting from Vault succeeds and writing to event queue fails.
		err = h.vault.Delete(src.ConfigRef)
		if err != nil {
			return err
		}

		err = h.sourceEventsQueue.Publish(api.SourceEvent{
			Action:     api.SourceDeleted,
			SourceID:   src.ID,
			SourceType: src.Type,
			ConfigRef:  src.ConfigRef,
		})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

// GetSources godoc
// @Summary      Returns a list of sources
// @Description  Returning a list of sources including both AWS and Azure unless filtered by Type.
// @Tags         onboard
// @Produce      json
// @Param        type  query     string  false  "Type"  Enums(aws,azure)
// @Success      200   {object}  api.GetSourcesResponse
// @Router       /onboard/api/v1/sources [get]
func (h HttpHandler) GetSources(ctx echo.Context) error {
	sType := ctx.QueryParam("type")
	var sources []Source
	if sType != "" {
		st, ok := api.AsSourceType(sType)
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid source type: %s", sType))
		}

		var err error
		sources, err = h.db.GetSourcesOfType(st)
		if err != nil {
			return err
		}
	} else {
		var err error
		sources, err = h.db.GetSources()
		if err != nil {
			return err
		}
	}

	resp := api.GetSourcesResponse{}
	for _, s := range sources {
		source := api.Source{
			ID:          s.ID,
			Name:        s.Name,
			SourceId:    s.SourceId,
			Type:        s.Type,
			Description: s.Description,
			OnboardDate: s.CreatedAt,
		}
		resp = append(resp, source)
	}

	return ctx.JSON(http.StatusOK, resp)
}

// CountSources godoc
// @Summary      Returns a count of sources
// @Description  Returning a count of sources including both AWS and Azure unless filtered by Type.
// @Tags         onboard
// @Produce      json
// @Param        type  query     string  false  "Type"  Enums(aws,azure)
// @Success      200   {object}  int64
// @Router       /onboard/api/v1/sources/count [get]
func (h HttpHandler) CountSources(ctx echo.Context) error {
	sType := ctx.QueryParam("type")
	var count int64
	if sType != "" {
		st, ok := api.AsSourceType(sType)
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid source type: %s", sType))
		}

		var err error
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

func (h HttpHandler) PutSource(ctx echo.Context) error {
	panic("not implemented yet")
}

// DiscoverAwsAccounts godoc
// @Summary      Returns the list of available AWS accounts given the credentials.
// @Description  If the account is part of an organization and the account has premission to list the accounts, it will return all the accounts in that organization. Otherwise, it will return the single account these credentials belong to.
// @Tags         onboard
// @Produce      json
// @Success      200        {object}  []api.DiscoverAWSAccountsResponse
// @Param        accessKey  body      string  true  "AccessKey"
// @Param        secretKey  body      string  true  "SecretKey"
// @Router       /onboard/api/v1/discover/aws/accounts [post]
func (h HttpHandler) DiscoverAwsAccounts(ctx echo.Context) error {
	// DiscoverAwsAccounts returns the list of available AWS accounts given the credentials.
	// If the account is part of an organization and the account has premission to
	// list the accounts, it will return all the accounts in that organization.
	// Otherwise, it will return the single account these credentials belong to.
	var req api.DiscoverAWSAccountsRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	accounts, err := discoverAwsAccounts(ctx.Request().Context(), req)
	if err != nil {
		return err
	}

	for _, account := range accounts {
		_, err := h.db.GetSourceBySourceID(account.AccountID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return err
		}
		account.Status = "DUPLICATE"
	}
	return ctx.JSON(http.StatusOK, accounts)
}

// DiscoverAzureSubscriptions godoc
// @Summary      Returns the list of available Azure subscriptions.
// @Description  Returning the list of available Azure subscriptions.
// @Tags         onboard
// @Produce      json
// @Success      200           {object}  []api.DiscoverAzureSubscriptionsResponse
// @Param        tenantId      body      string  true  "TenantId"
// @Param        clientId      body      string  true  "ClientId"
// @Param        clientSecret  body      string  true  "ClientSecret"
// @Router       /onboard/api/v1/discover/azure/subscriptions [post]
func (h *HttpHandler) DiscoverAzureSubscriptions(ctx echo.Context) error {
	var req api.DiscoverAzureSubscriptionsRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	subs, err := discoverAzureSubscriptions(ctx.Request().Context(), req)
	if err != nil {
		return err
	}

	for _, sub := range subs {
		_, err := h.db.GetSourceBySourceID(sub.SubscriptionID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return err
		}
		sub.Status = "DUPLICATE"
	}
	return ctx.JSON(http.StatusOK, subs)
}
