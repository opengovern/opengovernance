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

func (h HttpHandler) PutSource(ctx echo.Context) error {
	panic("not implemented yet")
}

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

	var resp api.GetSourcesResponse
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

// DiscoverAwsAccounts returns the list of available AWS accounts given the credentials.
// If the account is part of an organization and the account has premission to list the
// accounts, it will return all the accounts in that organization. Otherwise, it will return
// the single account these credentials belong to.
func (h HttpHandler) DiscoverAwsAccounts(ctx echo.Context) error {
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
