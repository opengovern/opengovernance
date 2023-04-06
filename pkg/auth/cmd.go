package auth

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"

	"gitlab.com/keibiengine/keibi-engine/pkg/auth/auth0"

	"gitlab.com/keibiengine/keibi-engine/pkg/workspace/client"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/email"

	envoyauth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	mailApiKey     = os.Getenv("EMAIL_API_KEY")
	mailSender     = os.Getenv("EMAIL_SENDER")
	mailSenderName = os.Getenv("EMAIL_SENDER_NAME")

	auth0Domain         = os.Getenv("AUTH0_DOMAIN")
	auth0ClientID       = os.Getenv("AUTH0_CLIENT_ID")
	auth0ClientIDNative = os.Getenv("AUTH0_CLIENT_ID_NATIVE")
	auth0ClientSecret   = os.Getenv("AUTH0_CLIENT_SECRET")

	auth0ManageDomain       = os.Getenv("AUTH0_MANAGE_DOMAIN")
	auth0ManageClientID     = os.Getenv("AUTH0_MANAGE_CLIENT_ID")
	auth0ManageClientSecret = os.Getenv("AUTH0_MANAGE_CLIENT_SECRET")
	auth0Connection         = os.Getenv("AUTH0_CONNECTION")
	auth0InviteTTL          = os.Getenv("AUTH0_INVITE_TTL")

	httpServerAddress = os.Getenv("HTTP_ADDRESS")

	keibiHost       = os.Getenv("KEIBI_HOST")
	keibiPublicKey  = os.Getenv("KEIBI_PUBLIC_KEY")
	keibiPrivateKey = os.Getenv("KEIBI_PRIVATE_KEY")

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

	verifier, err := newAuth0OidcVerifier(ctx, auth0Domain, auth0ClientID)
	if err != nil {
		return fmt.Errorf("open id connect verifier: %w", err)
	}

	verifierNative, err := newAuth0OidcVerifier(ctx, auth0Domain, auth0ClientIDNative)
	if err != nil {
		return fmt.Errorf("open id connect verifier: %w", err)
	}

	logger.Info("Instantiated a new Open ID Connect verifier")
	m := email.NewSendGridClient(mailApiKey, mailSender, mailSenderName, logger)

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

	b, err := base64.StdEncoding.DecodeString(keibiPublicKey)
	if err != nil {
		return fmt.Errorf("public key decode: %w", err)
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return fmt.Errorf("failed to decode my private key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return err
	}

	b, err = base64.StdEncoding.DecodeString(keibiPrivateKey)
	if err != nil {
		return fmt.Errorf("public key decode: %w", err)
	}
	block, _ = pem.Decode(b)
	if block == nil {
		panic("failed to decode private key")
	}
	pri, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		panic(err)
	}

	authServer := Server{
		host:            keibiHost,
		keibiPublicKey:  pub.(*rsa.PublicKey),
		verifier:        verifier,
		verifierNative:  verifierNative,
		logger:          logger,
		workspaceClient: workspaceClient,
	}
	authServer.cache = cache.New(&cache.Options{
		Redis:      rdb,
		LocalCache: cache.NewTinyLFU(10000, 5*time.Minute),
	})

	inviteTTL, err := strconv.ParseInt(auth0InviteTTL, 10, 64)
	if err != nil {
		return err
	}

	auth0Service := auth0.New(auth0ManageDomain, auth0ClientIDNative, auth0ClientID, auth0ManageClientID, auth0ManageClientSecret,
		auth0Connection, int(inviteTTL))

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
			logger:          logger,
			emailService:    m,
			workspaceClient: workspaceClient,
			auth0Service:    auth0Service,
			keibiPrivateKey: pri,
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
