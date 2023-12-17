package source

import (
	"net/http"

	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/services/onboard/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/labstack/echo/v4"
)

type Source struct{}

func (s Source) List(c echo.Context) error {
	var err error
	sType := httpserver.QueryArrayParam(c, "connector")
	var sources []model.Source
	if len(sType) > 0 {
		st := source.ParseTypes(sType)
		// trace :
		_, span := tracer.Start(ctx.Request().Context(), "new_GetSourcesOfTypes", trace.WithSpanKind(trace.SpanKindServer))
		span.SetName("new_GetSourcesOfTypes")

		sources, err = h.db.GetSourcesOfTypes(st)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		span.End()
	} else {
		// trace :
		_, span := tracer.Start(ctx.Request().Context(), "new_ListSources", trace.WithSpanKind(trace.SpanKindServer))
		span.SetName("new_ListSources")

		sources, err = h.db.ListSources()
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		span.End()
	}

	resp := api.GetSourcesResponse{}
	for _, s := range sources {
		apiRes := s.ToAPI()
		if httpserver.GetUserRole(c) == api.InternalRole {
			apiRes.Credential = s.Credential.ToAPI()
			apiRes.Credential.Config = s.Credential.Secret
			if apiRes.Credential.Version == 2 {
				apiRes.Credential.Config, err = h.CredentialV2ToV1(s.Credential.Secret)
				if err != nil {
					return err
				}
			}
		}
		resp = append(resp, apiRes)
	}

	return c.JSON(http.StatusOK, resp)
}

func (s Source) Get(c echo.Context) error {
	return nil
}

func (s Source) Count(c echo.Context) error {
	return nil
}

func (s Source) Register(g *echo.Group) {
	g.GET("/sources", httpserver.AuthorizeHandler(s.List, api.ViewerRole))
	g.POST("/sources", httpserver.AuthorizeHandler(s.Get, api.KaytuAdminRole))
	g.GET("/sources/count", httpserver.AuthorizeHandler(s.Count, api.ViewerRole))
}
