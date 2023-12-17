package source

import (
	"net/http"

	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"github.com/kaytu-io/kaytu-engine/services/integration/repository"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Source struct {
	keyARN          string
	kms             *vault.KMSVaultSourceConfig
	repo            repository.Source
	tracer          trace.Tracer
	masterAccessKey string
	masterSecretKey string
}

func New(keyARN string, kms *vault.KMSVaultSourceConfig, repo repository.Source, masterAccessKey, masterSecretKey string) Source {
	return Source{
		keyARN:          keyARN,
		kms:             kms,
		repo:            repo,
		tracer:          otel.GetTracerProvider().Tracer("integration.http.sources"),
		masterAccessKey: masterAccessKey,
		masterSecretKey: masterSecretKey,
	}
}

func (h Source) CredentialV2ToV1(newCred string) (string, error) {
	cnf, err := h.kms.Decrypt(newCred, h.keyARN)
	if err != nil {
		return "", err
	}

	awsCnf, err := AWSCredentialV2ConfigFromMap(cnf)
	if err != nil {
		return "", err
	}

	newConf := AWSCredentialConfig{
		AccountId:            awsCnf.AccountID,
		Regions:              nil,
		AccessKey:            h.masterAccessKey,
		SecretKey:            h.masterSecretKey,
		AssumeRoleName:       awsCnf.AssumeRoleName,
		AssumeAdminRoleName:  awsCnf.AssumeRoleName,
		AssumeRolePolicyName: "",
		ExternalId:           awsCnf.ExternalId,
	}
	newSecret, err := h.kms.Encrypt(newConf.AsMap(), h.keyARN)
	if err != nil {
		return "", err
	}

	return string(newSecret), nil
}

func (h Source) List(c echo.Context) error {
	ctx := c.Request().Context()

	sType := httpserver.QueryArrayParam(c, "connector")

	var (
		sources []model.Source
		err     error
	)

	_, span := h.tracer.Start(ctx, "list", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	if len(sType) > 0 {
		st := source.ParseTypes(sType)
		span.SetName("list.with-types")

		sources, err = h.repo.GetSourcesOfTypes(st)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		span.End()
	} else {
		span.SetName("list.without-types")

		sources, err = h.repo.ListSources()
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}

	resp := GetSourcesResponse{}
	for _, s := range sources {
		apiRes := SourceToAPI(s)
		if httpserver.GetUserRole(c) == api.InternalRole {
			apiRes.Credential = CredentialToAPI(s.Credential)
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
