package auth

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"github.com/go-redis/redis/v8"

	"gitlab.com/keibiengine/keibi-engine/pkg/workspace/client"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/email"

	envoyauth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/postgres"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	postgreSQLHost     = os.Getenv("POSTGRESQL_HOST")
	postgreSQLPort     = os.Getenv("POSTGRESQL_PORT")
	postgreSQLDb       = os.Getenv("POSTGRESQL_DB")
	postgreSQLUser     = os.Getenv("POSTGRESQL_USERNAME")
	postgreSQLPassword = os.Getenv("POSTGRESQL_PASSWORD")

	mailApiKey     = os.Getenv("EMAIL_API_KEY")
	mailSender     = os.Getenv("EMAIL_SENDER")
	mailSenderName = os.Getenv("EMAIL_SENDER_NAME")

	auth0Domain   = os.Getenv("AUTH0_DOMAIN")
	auth0ClientID = os.Getenv("AUTH0_CLIENT_ID")

	httpServerAddress  = os.Getenv("HTTP_ADDRESS")
	inviteLinkTemplate = os.Getenv("INVITE_LINK_TEMPLATE")

	keibiHost = os.Getenv("KEIBI_HOST")

	workspaceBaseUrl = os.Getenv("WORKSPACE_BASE_URL")

	RedisAddress = os.Getenv("REDIS_ADDRESS")

	grpcServerAddress = os.Getenv("GRPC_ADDRESS")
	grpcTlsCertPath   = os.Getenv("GRPC_TLS_CERT_PATH")
	grpcTlsKeyPath    = os.Getenv("GRPC_TLS_KEY_PATH")
	grpcTlsCAPath     = os.Getenv("GRPC_TLS_CA_PATH")
)

func Command() *cobra.Command {
	return &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			return start(cmd.Context())
		},
	}
}

// start runs both HTTP and GRPC server.
// GRPC server has Check method to ensure user is
// authenticated and authorized to perform an action.
// HTTP server has multiple endpoints to view and update
// the user roles.
func start(ctx context.Context) error {
	logger, err := zap.NewProduction()
	if err != nil {
		return err
	}

	cfg := postgres.Config{
		Host:   postgreSQLHost,
		Port:   postgreSQLPort,
		User:   postgreSQLUser,
		Passwd: postgreSQLPassword,
		DB:     postgreSQLDb,
	}
	orm, err := postgres.NewClient(&cfg, logger)
	if err != nil {
		return fmt.Errorf("new postgres client: %w", err)
	}
	logger.Info("Connected to the postgres database: ", zap.String("orm", "postgresDb"))

	verifier, err := newAuth0OidcVerifier(ctx, auth0Domain, auth0ClientID)
	if err != nil {
		return fmt.Errorf("open id connect verifier: %w", err)
	}
	logger.Info("Instantiated a new Open ID Connect verifier")
	m := email.NewSendGridClient(mailApiKey, mailSender, mailSenderName, logger)

	db := NewDatabase(orm)
	err = db.Initialize()
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}

	creds, err := newServerCredentials(
		grpcTlsCertPath,
		grpcTlsKeyPath,
		grpcTlsCAPath,
	)
	if err != nil {
		return fmt.Errorf("grpc tls creds: %w", err)
	}

	workspaceClient := client.NewWorkspaceClient(workspaceBaseUrl)

	rdb := redis.NewClient(&redis.Options{
		Addr:     RedisAddress,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	authServer := Server{
		host:            keibiHost,
		db:              db,
		verifier:        verifier,
		logger:          logger,
		workspaceClient: workspaceClient,
		rdb:             rdb,
	}

	grpcServer := grpc.NewServer(grpc.Creds(creds))
	envoyauth.RegisterAuthorizationServer(grpcServer, authServer)

	lis, err := net.Listen("tcp", grpcServerAddress)
	if err != nil {
		return fmt.Errorf("grpc listen: %w", err)
	}

	errors := make(chan error, 1)
	go func() {
		errors <- fmt.Errorf("grpc server: %w", grpcServer.Serve(lis))
	}()

	go func() {
		routes := httpRoutes{
			logger:             logger,
			db:                 db,
			emailService:       m,
			inviteLinkTemplate: inviteLinkTemplate,
			workspaceClient:    workspaceClient,
		}
		errors <- fmt.Errorf("http server: %w", httpserver.RegisterAndStart(logger, httpServerAddress, &routes))
	}()

	return <-errors
}

// newServerCredentials loads TLS transport credentials for the GRPC server.
func newServerCredentials(certPath string, keyPath string, caPath string) (credentials.TransportCredentials, error) {
	srv, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}

	p := x509.NewCertPool()

	if caPath != "" {
		ca, err := ioutil.ReadFile(caPath) //nolint(gosec)
		if err != nil {
			return nil, err
		}

		p.AppendCertsFromPEM(ca)
	}

	return credentials.NewTLS(&tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{srv},
		RootCAs:      p,
	}), nil
}
