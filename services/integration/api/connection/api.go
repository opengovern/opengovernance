package connection

import (
	"net/http"

	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/services/integration/service"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type API struct {
	keyARN          string
	kms             *vault.KMSVaultSourceConfig
	svc             service.Connection
	tracer          trace.Tracer
	logger          *zap.Logger
	masterAccessKey string
	masterSecretKey string
}

func New(
	keyARN string,
	kms *vault.KMSVaultSourceConfig,
	svc service.Connection,
	logger *zap.Logger,
	masterAccessKey, masterSecretKey string,
) API {
	return API{
		keyARN:          keyARN,
		kms:             kms,
		svc:             svc,
		tracer:          otel.GetTracerProvider().Tracer("integration.http.sources"),
		logger:          logger.Named("source"),
		masterAccessKey: masterAccessKey,
		masterSecretKey: masterSecretKey,
	}
}

func (h API) CredentialV2ToV1(newCred string) (string, error) {
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

func (h API) List(c echo.Context) error {
	ctx := c.Request().Context()

	types := httpserver.QueryArrayParam(c, "connector")

	sources, err := h.svc.List(ctx, source.ParseTypes(types))
	if err != nil {
		h.logger.Error("failed to read sources from the service", zap.Error(err))

		return echo.ErrInternalServerError
	}

	var resp ListConnectionsResponse
	for _, s := range sources {
		apiRes := NewConnection(s)
		if httpserver.GetUserRole(c) == api.InternalRole {
			apiRes.Credential = NewCredential(s.Credential)
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

func (s API) Get(c echo.Context) error {
	return nil
}

func (s API) Count(c echo.Context) error {
	return nil
}

func (s API) Register(g *echo.Group) {
	g.GET("/", httpserver.AuthorizeHandler(s.List, api.ViewerRole))
	g.POST("/", httpserver.AuthorizeHandler(s.Get, api.KaytuAdminRole))
	g.GET("/count", httpserver.AuthorizeHandler(s.Count, api.ViewerRole))
}
