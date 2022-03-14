package onboard

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
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
	source.PUT("/:sourceId", h.PutSource)
	source.DELETE("/:sourceId", h.DeleteSource)

	v1.GET("/sources", h.GetSources)

	disc := v1.Group("/discover")

	disc.GET("/aws/accounts", h.DiscoverAwsAccounts)
	disc.GET("/azure/subscriptions", h.DiscoverAzureSubscriptions)

	v1.GET("/providers", h.GetProviders)
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
// @Tags         onboard
// @Produce      json
// @Success      200  {object}  api.ProvidersResponse
// @Router       /onboard/api/v1/providers [get]
func (h HttpHandler) GetProviders(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, api.ProvidersResponse{
		{
			ID:      "aws",
			Name:    "Amazon Web Services",
			Enabled: true,
			Type:    "PublicCloud",
		},
		{
			ID:      "azure",
			Name:    "Microsoft Azure",
			Enabled: true,
			Type:    "PublicCloud",
		},
	})
}

// PostSourceAws godoc
// @Summary      Create AWS source
// @Description  Creating AWS source
// @Tags         onboard
// @Produce      json
// @Success      200  {object}  api.CreateSourceResponse
// @Param        name              body      string  true  "name"
// @Param        description       body      string  true  "description"
// @Param        config            body      api.SourceConfigAWS  true  "config"
// @Router       /onboard/api/v1/aws [post]
func (h HttpHandler) PostSourceAws(ctx echo.Context) error {
	var req api.SourceAwsRequest
	if err := bindValidate(ctx, &req); err != nil {
		return ctx.JSON(http.StatusBadRequest, err)
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
		return ctx.JSON(http.StatusInternalServerError, err)
	}

	return ctx.JSON(http.StatusOK, src.toSourceResponse())
}

// PostSourceAzure godoc
// @Summary      Create Azure source
// @Description  Creating Azure source
// @Tags         onboard
// @Produce      json
// @Success      200  {object}  api.CreateSourceResponse
// @Param        name              body      string  true  "name"
// @Param        description       body      string  true  "description"
// @Param        config            body      api.SourceConfigAzure  true  "config"
// @Router       /onboard/api/v1/azure [post]
func (h HttpHandler) PostSourceAzure(ctx echo.Context) error {
	var req api.SourceAzureRequest

	if err := bindValidate(ctx, &req); err != nil {
		return ctx.JSON(http.StatusBadRequest, err)
	}

	src := NewAzureSource(req)

	err := h.vault.Write(src.ConfigRef, req.Config.AsMap())
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, err)
	}

	err = h.db.orm.Transaction(func(tx *gorm.DB) error {
		err = h.db.CreateSource(&src)
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
		return ctx.JSON(http.StatusInternalServerError, err)
	}

	return ctx.JSON(http.StatusOK, src.toSourceResponse())
}

// GetSource godoc
// @Summary      Returns a single source
// @Description  Returning single source either AWS / Azure.
// @Tags         onboard
// @Produce      json
// @Success      200  {object}  api.Source
// @Param        sourceId   path      integer  true  "SourceID"
// @Router       /onboard/api/v1/{sourceId} [get]
func (h HttpHandler) GetSource(ctx echo.Context) error {
	srcId, err := uuid.Parse(ctx.Param(paramSourceId))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err)
	}

	src, err := h.db.GetSource(srcId)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err)
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
// @Param        sourceId   path      integer  true  "SourceID"
// @Router       /onboard/api/v1/{sourceId} [delete]
func (h HttpHandler) DeleteSource(ctx echo.Context) error {
	srcId, err := uuid.Parse(ctx.Param(paramSourceId))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err)
	}

	src, err := h.db.GetSource(srcId)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err)
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
		return ctx.JSON(http.StatusBadRequest, err)
	}

	return ctx.NoContent(http.StatusOK)
}

// GetSources godoc
// @Summary      Returns a list of sources
// @Description  Returning a list of sources including both AWS and Azure unless filtered by Type.
// @Tags         onboard
// @Produce      json
// @Param        type query string false "Type" Enums(aws,azure)
// @Success      200  {object}  api.GetSourcesResponse
// @Router       /onboard/api/v1/sources [get]
func (h HttpHandler) GetSources(ctx echo.Context) error {
	sType := ctx.QueryParam("type")
	var sources []Source
	if sType != "" {
		st, ok := api.AsSourceType(sType)
		if !ok {
			return ctx.JSON(http.StatusBadRequest, fmt.Errorf("invalid source type: %s", sType))
		}

		var err error
		sources, err = h.db.GetSourcesOfType(st)
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, err)
		}
	} else {
		var err error
		sources, err = h.db.GetSources()
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, err)
		}
	}

	resp := api.GetSourcesResponse{}
	for _, s := range sources {
		resp = append(resp, api.Source{
			ID:          s.ID,
			Name:        s.Name,
			SourceId:    s.SourceId,
			Type:        s.Type,
			Description: s.Description,
		})
	}

	return ctx.JSON(http.StatusOK, resp)
}

func (h HttpHandler) PutSource(ctx echo.Context) error {
	panic("not implemented yet")
}

// DiscoverAwsAccounts godoc
// @Summary     Returns the list of available AWS accounts given the credentials.
// @Description  If the account is part of an organization and the account has premission to list the accounts, it will return all the accounts in that organization. Otherwise, it will return the single account these credentials belong to.
// @Tags         onboard
// @Produce      json
// @Success      200  {object}  []api.DiscoverAWSAccountsResponse
// @Param        accessKey       body      string  true  "AccessKey"
// @Param        secretKey       body      string  true  "SecretKey"
// @Router       /onboard/api/v1/aws/accounts [get]
func (h HttpHandler) DiscoverAwsAccounts(ctx echo.Context) error {
	// DiscoverAwsAccounts returns the list of available AWS accounts given the credentials.
	// If the account is part of an organization and the account has premission to
	// list the accounts, it will return all the accounts in that organization.
	// Otherwise, it will return the single account these credentials belong to.
	var req api.DiscoverAWSAccountsRequest
	if err := bindValidate(ctx, &req); err != nil {
		return ctx.JSON(http.StatusBadRequest, err)
	}

	accounts, err := discoverAwsAccounts(ctx.Request().Context(), req)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	return ctx.JSON(http.StatusOK, accounts)
}

// DiscoverAzureSubscriptions godoc
// @Summary     Returns the list of available Azure subscriptions.
// @Description  Returning the list of available Azure subscriptions.
// @Tags         onboard
// @Produce      json
// @Success      200  {object}  []api.DiscoverAzureSubscriptionsResponse
// @Param        tenantId       body      string  true  "TenantId"
// @Param        clientId       body      string  true  "ClientId"
// @Param        clientSecret   body      string  true  "ClientSecret"
// @Router       /onboard/api/v1/azure/subscriptions [get]
func (h *HttpHandler) DiscoverAzureSubscriptions(ctx echo.Context) error {
	var req api.DiscoverAzureSubscriptionsRequest
	if err := bindValidate(ctx, &req); err != nil {
		return ctx.JSON(http.StatusBadRequest, err)
	}

	subs, err := discoverAzureSubscriptions(ctx.Request().Context(), req)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err)
	}

	return ctx.JSON(http.StatusOK, subs)
}
