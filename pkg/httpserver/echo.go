package httpserver

import (
	"context"
	"fmt"
	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	jwtvalidator "github.com/auth0/go-jwt-middleware/v2/validator"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/metrics"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.uber.org/zap"
	"gopkg.in/go-playground/validator.v9"
)

var (
	agentHost    = os.Getenv("JAEGER_AGENT_HOST")
	serviceName  = os.Getenv("JAEGER_SERVICE_NAME")
	sampleRate   = os.Getenv("JAEGER_SAMPLE_RATE")
	authDomain   = os.Getenv("AUTH_DOMAIN")
	authAudience = os.Getenv("AUTH_AUDIENCE")
)

type Routes interface {
	Register(router *echo.Echo)
}

type EmptyRoutes struct{}

func (EmptyRoutes) Register(router *echo.Echo) {}

func Register(logger *zap.Logger, routes Routes) (*echo.Echo, *sdktrace.TracerProvider) {
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Recover())
	e.Use(Logger(logger))
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Skipper: func(c echo.Context) bool {
			// skip metric endpoints
			if strings.HasPrefix(c.Path(), "/metrics") {
				return true
			}
			// skip if client does not accept gzip
			acceptEncodingHeader := c.Request().Header.Values(echo.HeaderAcceptEncoding)
			for _, value := range acceptEncodingHeader {
				if strings.TrimSpace(value) == "gzip" {
					return false
				}
			}
			return true
		},
		Level: 5,
	}))
	e.Use(validatorMiddleware)

	metrics.AddEchoMiddleware(e)

	e.Pre(middleware.RemoveTrailingSlash())

	tp, err := initTracer()
	if err != nil {
		logger.Error(err.Error())
		return nil, nil
	}

	e.Validator = customValidator{
		validate: validator.New(),
	}

	routes.Register(e)

	return e, tp
}

func RegisterAndStart(ctx context.Context, logger *zap.Logger, address string, routes Routes) error {
	e, tp := Register(logger, routes)

	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
		}
	}()
	e.Use(otelecho.Middleware(serviceName))

	return e.Start(address)
}

type customValidator struct {
	validate *validator.Validate
}

func (v customValidator) Validate(i interface{}) error {
	return v.validate.Struct(i)
}

func QueryArrayParam(ctx echo.Context, paramName string) []string {
	var values []string
	for k, v := range ctx.QueryParams() {
		if k == paramName || k == paramName+"[]" {
			values = append(values, v...)
		}
	}
	return values
}

func initTracer() (*sdktrace.TracerProvider, error) {
	exporter, err := jaeger.New(jaeger.WithAgentEndpoint(jaeger.WithAgentHost(agentHost)))
	if err != nil {
		return nil, err
	}

	sampleRateFloat := 1.0
	if sampleRate != "" {
		sampleRateFloat, err = strconv.ParseFloat(sampleRate, 64)
		if err != nil {
			fmt.Println("Error parsing sample rate for Jaeger. Using default value of 1.0", err)
			sampleRateFloat = 1
		}
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(sampleRateFloat)),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, nil
}

func validatorMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		issuerURL, err := url.Parse("https://" + authDomain + "/")
		if err != nil {
			return err
		}

		provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

		jwtValidator, err := jwtvalidator.New(
			provider.KeyFunc,
			jwtvalidator.RS256,
			issuerURL.String(),
			[]string{authAudience},
			jwtvalidator.WithCustomClaims(
				func() jwtvalidator.CustomClaims {
					return nil
				},
			),
			jwtvalidator.WithAllowedClockSkew(time.Minute),
		)
		if err != nil {
			return err
		}

		errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"message":"Failed to validate JWT."}`))
		}

		jwtMiddleware := jwtmiddleware.New(
			jwtValidator.ValidateToken,
			jwtmiddleware.WithErrorHandler(errorHandler),
		)

		// Adapter function to convert Echo handler to http.HandlerFunc
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.SetRequest(r)
			c.SetResponse(echo.NewResponse(w, c.Echo()))
			if err := next(c); err != nil {
				c.Error(err)
			}
		})

		// Wrap the Echo context response writer and request with standard http.ResponseWriter and http.Request
		writer := c.Response().Writer
		request := c.Request()

		// Check JWT and handle the request
		jwtMiddleware.CheckJWT(h).ServeHTTP(writer, request)

		// If the token is valid, the request will be handled by next(c)
		return nil
	}
}
