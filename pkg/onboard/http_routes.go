package onboard

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/describer"
	"gorm.io/gorm/clause"
)

func (h *HttpHandler) Register(v1 *echo.Group) {
	o := v1.Group("/organizations")

	o.POST("", h.PostOrganization)
	o.GET("/:organizationId", h.GetOrganization)
	o.PUT("/:organizationId", h.PutOrganization)
	o.DELETE("/:organizationId", h.DeleteOrganization)

	o.POST("/:organizationId/sources/aws", h.PostSourceAws)
	o.POST("/:organizationId/sources/azure", h.PostSourceAzure)
	o.GET("/:organizationId/sources/:sourceId", h.GetSource)
	o.PUT("/:organizationId/sources/:sourceId", h.PutSource)
	o.DELETE("/:organizationId/sources/:sourceId", h.DeleteSource)
}

func (c *Context) BindValidate(i interface{}) error {
	if err := c.Bind(i); err != nil {
		return err
	}

	if err := c.Validate(i); err != nil {
		return err
	}

	return nil
}

func (h *HttpHandler) PostOrganization(ctx echo.Context) error {
	cc := ctx.(*Context)
	req := &OrganizationRequest{}

	if err := cc.BindValidate(req); err != nil {
		return cc.JSON(http.StatusBadRequest, NewError(err))
	}

	org := req.toOrganization()

	// create an organization path in the vault
	pathRef, err := h.vault.NewOrganization(org.ID)
	if err != nil {
		return cc.JSON(http.StatusInternalServerError, NewError(err))
	}
	org.VaultRef = pathRef

	// save organization to the database
	org, err = h.db.CreateOrganization(org)
	if err != nil {
		return cc.JSON(http.StatusInternalServerError, NewError(err))
	}

	return cc.JSON(http.StatusCreated, org.toOrganizationResponse())
}

func (h *HttpHandler) GetOrganization(ctx echo.Context) error {
	p := ctx.Param("organizationId")
	orgId, err := uuid.Parse(p)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, NewError(err))
	}

	org, err := h.db.GetOrganization(orgId)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, NewError(err))
	}

	return ctx.JSON(http.StatusOK, org.toOrganizationResponse())
}

func (h *HttpHandler) PutOrganization(ctx echo.Context) error {
	cc := ctx.(*Context)
	req := &OrganizationRequest{}
	if err := cc.BindValidate(req); err != nil {
		return cc.JSON(http.StatusBadRequest, NewError(err))
	}

	org, err := h.db.UpdateOrganization(req.toOrganization())

	if err != nil {
		return cc.JSON(http.StatusInternalServerError, NewError(err))
	}

	return cc.JSON(http.StatusCreated, org.toOrganizationResponse())
}

func (h *HttpHandler) DeleteOrganization(ctx echo.Context) error {
	p := ctx.Param("organizationId")
	orgId, err := uuid.Parse(p)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, NewError(err))
	}

	// get organization
	org, err := h.db.GetOrganization(orgId)
	if err != nil || org == nil {
		return ctx.JSON(http.StatusBadRequest, NewError(err))
	}

	// delete organization from the vault
	err = h.vault.DeleteOrganization(org.VaultRef)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, NewError(err))
	}

	// delete organization from the database
	if err := h.db.DeleteOrganization(orgId); err != nil {
		return ctx.JSON(http.StatusInternalServerError, NewError(err))
	}

	return ctx.NoContent(http.StatusOK)
}

func (h *HttpHandler) PostSourceAws(ctx echo.Context) error {
	cc := ctx.(*Context)
	req := &SourceAwsRequest{}

	if err := cc.BindValidate(req); err != nil {
		return cc.JSON(http.StatusBadRequest, NewError(err))
	}

	p := ctx.Param("organizationId")
	orgId, err := uuid.Parse(p)
	if err != nil {
		return cc.JSON(http.StatusBadRequest, NewError(err))
	}

	cfg, err := describer.GetConfig(ctx.Request().Context(), req.Config.AccessKey, req.Config.SecretKey)
	if err != nil {
		return cc.JSON(http.StatusBadRequest, NewError(err))
	}

	src := req.toSource(orgId)

	// ensure that the org id is valid
	org, err := h.db.GetOrganization(orgId)
	if err != nil || org == nil {
		return cc.JSON(http.StatusNotFound, NewError(err))
	}

	accID, err := describer.GetAccountId(ctx.Request().Context(), cfg)
	if err != nil {
		return cc.JSON(http.StatusBadRequest, NewError(err))
	}

	acc, err := describer.DescribeAccountById(ctx.Request().Context(), cfg, accID)
	if err != nil {
		return cc.JSON(http.StatusBadRequest, NewError(err))
	}

	// save source to the database
	src, err = h.db.CreateSource(src)
	if err != nil {
		return cc.JSON(http.StatusInternalServerError, NewError(err))
	}

	// Prefill AWS metadata
	awsmetadata := AWSMetadata{}
	awsmetadata.Email = *acc.Email
	awsmetadata.Name = *acc.Name
	awsmetadata.SourceID = src.ID.String()

	atx := h.db.orm.
		Model(&AWSMetadata{}).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(awsmetadata)

	if atx.Error != nil {
		return cc.JSON(http.StatusBadRequest, NewError(atx.Error))
	} else if atx.RowsAffected != 1 {
		return cc.JSON(http.StatusBadRequest, NewError(fmt.Errorf("create aws metadata: didn't create aws metadata due to id conflict: %w", atx.Error)))
	}

	// write config to the vault
	pathRef, err := h.vault.WriteSourceConfig(orgId, src.ID, string(SourceCloudAWS), req.Config)
	if err != nil {
		return cc.JSON(http.StatusInternalServerError, NewError(err))
	}
	src.ConfigRef = pathRef


	err = h.sourceEventsQueue.Publish(SourceEvent{
		Action:     SourceCreated,
		SourceID:   src.ID,
		SourceType: src.Type,
		ConfigRef:  src.ConfigRef,
	})
	if err != nil {
		fmt.Println(err.Error()) // TODO
	}


	return cc.JSON(http.StatusCreated, src.toSourceResponse())
}

func (h *HttpHandler) PostSourceAzure(ctx echo.Context) error {
	cc := ctx.(*Context)
	req := &SourceAzureRequest{}

	if err := cc.BindValidate(req); err != nil {
		return cc.JSON(http.StatusBadRequest, NewError(err))
	}

	p := ctx.Param("organizationId")
	orgId, err := uuid.Parse(p)
	if err != nil {
		return cc.JSON(http.StatusBadRequest, NewError(err))
	}
	src := req.toSource(orgId)

	// ensure that the org id is valid
	org, err := h.db.GetOrganization(orgId)
	if err != nil || org == nil {
		return cc.JSON(http.StatusBadRequest, NewError(err))
	}

	// write config to the vault
	pathRef, err := h.vault.WriteSourceConfig(orgId, src.ID, string(SourceCloudAzure), req.Config)
	if err != nil {
		return cc.JSON(http.StatusInternalServerError, NewError(err))
	}
	src.ConfigRef = pathRef

	err = h.sourceEventsQueue.Publish(SourceEvent{
		Action:     SourceCreated,
		SourceID:   src.ID,
		SourceType: src.Type,
		ConfigRef:  src.ConfigRef,
	})
	if err != nil {
		fmt.Println(err.Error()) // TODO
	}
	// TODO: synchronize transactions & error handling

	// save source to the database
	src, err = h.db.CreateSource(src)
	if err != nil {
		return cc.JSON(http.StatusInternalServerError, NewError(err))
	}

	return cc.JSON(http.StatusCreated, src.toSourceResponse())
}

func (h *HttpHandler) GetSource(ctx echo.Context) error {
	cc := ctx.(*Context)
	p := ctx.Param("organizationId")
	orgId, err := uuid.Parse(p)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, NewError(err))
	}

	p = ctx.Param("sourceId")
	srcId, err := uuid.Parse(p)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, NewError(err))
	}

	src, err := h.db.GetSource(srcId)
	if err != nil || src == nil {
		return ctx.JSON(http.StatusBadRequest, NewError(err))
	}

	if src.OrganizationID != orgId {
		return ctx.JSON(http.StatusBadRequest, fmt.Errorf("no source with id %q was found for organization with id %q", srcId, orgId))
	}

	return cc.JSON(http.StatusOK, src.toSourceResponse())
}

func (h *HttpHandler) PutSource(ctx echo.Context) error {
	panic("not implemented yet")
}

func (h *HttpHandler) DeleteSource(ctx echo.Context) error {
	p := ctx.Param("organizationId")
	orgId, err := uuid.Parse(p)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, NewError(err))
	}

	p = ctx.Param("sourceId")
	srcId, err := uuid.Parse(p)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, NewError(err))
	}

	// get organization
	src, err := h.db.GetSource(srcId)
	if err != nil || src == nil {
		return ctx.JSON(http.StatusBadRequest, NewError(err))
	}

	if src.OrganizationID != orgId {
		return ctx.JSON(http.StatusBadRequest, fmt.Errorf("no source with id %q was found for organization with id %q", srcId, orgId))
	}

	// delete source from vault
	err = h.vault.DeleteSourceConfig(src.ConfigRef)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, fmt.Errorf("error occured while trying to delete source with id %q", srcId))
	}

	err = h.sourceEventsQueue.Publish(SourceEvent{
		Action:     SourceDeleted,
		SourceID:   src.ID,
		SourceType: src.Type,
		ConfigRef:  src.ConfigRef,
	})
	if err != nil {
		fmt.Println(err.Error()) // TODO
	}

	// TODO: synchronize transactions & error handling

	// delete organization from the database
	if err := h.db.DeleteSource(srcId); err != nil {
		return ctx.JSON(http.StatusInternalServerError, NewError(err))
	}

	return ctx.NoContent(http.StatusOK)
}
