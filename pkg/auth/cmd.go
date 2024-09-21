package auth

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	config2 "github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/kaytu-io/kaytu-util/pkg/httpserver"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"os"
	"strconv"

	"github.com/kaytu-io/open-governance/pkg/auth/auth0"
	"github.com/kaytu-io/open-governance/pkg/auth/db"

	"github.com/kaytu-io/open-governance/pkg/workspace/client"

	"crypto/rand"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"google.golang.org/grpc/credentials"
)

var (
	mailApiKey     = os.Getenv("EMAIL_API_KEY")
	mailSender     = os.Getenv("EMAIL_SENDER")
	mailSenderName = os.Getenv("EMAIL_SENDER_NAME")

	dexAuthDomain                = os.Getenv("DEX_AUTH_DOMAIN")
	dexAuthPublicClientID        = os.Getenv("DEX_AUTH_PUBLIC_CLIENT_ID")
	dexGrpcAddress               = os.Getenv("DEX_GRPC_ADDR")
	auth0Domain                  = os.Getenv("AUTH0_DOMAIN")
	auth0ClientID                = os.Getenv("AUTH0_CLIENT_ID")
	auth0ClientIDNative          = os.Getenv("AUTH0_CLIENT_ID_NATIVE")
	auth0ClientIDPennywiseNative = os.Getenv("AUTH0_CLIENT_ID_PENNYWISE_NATIVE")

	auth0ManageDomain       = os.Getenv("AUTH0_MANAGE_DOMAIN")
	auth0ManageClientID     = os.Getenv("AUTH0_MANAGE_CLIENT_ID")
	auth0ManageClientSecret = os.Getenv("AUTH0_MANAGE_CLIENT_SECRET")
	auth0Connection         = os.Getenv("AUTH0_CONNECTION")
	auth0InviteTTL          = os.Getenv("AUTH0_INVITE_TTL")

	httpServerAddress = os.Getenv("HTTP_ADDRESS")

	kaytuHost          = os.Getenv("KAYTU_HOST")
	kaytuKeyEnabledStr = os.Getenv("KAYTU_KEY_ENABLED")
	kaytuPublicKeyStr  = os.Getenv("KAYTU_PUBLIC_KEY")
	kaytuPrivateKeyStr = os.Getenv("KAYTU_PRIVATE_KEY")

	workspaceBaseUrl = os.Getenv("WORKSPACE_BASE_URL")
	metadataBaseUrl  = os.Getenv("METADATA_BASE_URL")
)

func Command() *cobra.Command {
	return &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			return start(cmd.Context())
		},
	}
}

type ServerConfig struct {
	PostgreSQL config2.Postgres
}

// start runs both HTTP and GRPC server.
// GRPC server has Check method to ensure user is
// authenticated and authorized to perform an action.
// HTTP server has multiple endpoints to view and update
// the user roles.
func start(ctx context.Context) error {
	var conf ServerConfig
	config2.ReadFromEnv(&conf, nil)

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

	verifierPennywiseNative, err := newAuth0OidcVerifier(ctx, auth0Domain, auth0ClientIDPennywiseNative)
	if err != nil {
		return fmt.Errorf("open id connect verifier pennywise: %w", err)
	}

	dexVerifier, err := newDexOidcVerifier(ctx, dexAuthDomain, dexAuthPublicClientID)
	if err != nil {
		return fmt.Errorf("open id connect dex verifier: %w", err)
	}

	logger.Info("Instantiated a new Open ID Connect verifier")
	//m := email.NewSendGridClient(mailApiKey, mailSender, mailSenderName, logger)

	workspaceClient := client.NewWorkspaceClient(workspaceBaseUrl)

	inviteTTL, err := strconv.ParseInt(auth0InviteTTL, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse auth0InviteTTL=%s due to %v", auth0InviteTTL, err)
	}

	// setup postgres connection
	cfg := postgres.Config{
		Host:    conf.PostgreSQL.Host,
		Port:    conf.PostgreSQL.Port,
		User:    conf.PostgreSQL.Username,
		Passwd:  conf.PostgreSQL.Password,
		DB:      conf.PostgreSQL.DB,
		SSLMode: conf.PostgreSQL.SSLMode,
	}
	orm, err := postgres.NewClient(&cfg, logger)
	if err != nil {
		return fmt.Errorf("new postgres client: %w", err)
	}

	adb := db.Database{Orm: orm}
	fmt.Println("Connected to the postgres database: ", conf.PostgreSQL.DB)

	err = adb.Initialize()
	if err != nil {
		return fmt.Errorf("new postgres client: %w", err)
	}

	if kaytuKeyEnabledStr == "" {
		kaytuKeyEnabledStr = "false"
	}
	kaytuKeyEnabled, err := strconv.ParseBool(kaytuKeyEnabledStr)
	if err != nil {
		return fmt.Errorf("kaytuKeyEnabled [%s]: %w", kaytuKeyEnabledStr, err)
	}

	var kaytuPublicKey *rsa.PublicKey
	var kaytuPrivateKey *rsa.PrivateKey
	if kaytuKeyEnabled {
		b, err := base64.StdEncoding.DecodeString(kaytuPublicKeyStr)
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
		kaytuPublicKey = pub.(*rsa.PublicKey)

		b, err = base64.StdEncoding.DecodeString(kaytuPrivateKeyStr)
		if err != nil {
			return fmt.Errorf("private key decode: %w", err)
		}
		block, _ = pem.Decode(b)
		if block == nil {
			panic("failed to decode private key")
		}
		pri, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			panic(err)
		}
		kaytuPrivateKey = pri.(*rsa.PrivateKey)
	} else {
		keyPair, err := adb.GetKeyPair()
		if err != nil {
			panic(err)
		}

		if len(keyPair) == 0 {
			kaytuPrivateKey, err = rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				panic(fmt.Sprintf("Error generating RSA key: %v", err))
			}
			kaytuPublicKey = &kaytuPrivateKey.PublicKey

			b, err := x509.MarshalPKIXPublicKey(kaytuPublicKey)
			if err != nil {
				panic(err)
			}
			bp := pem.EncodeToMemory(&pem.Block{
				Type:    "RSA PUBLIC KEY",
				Headers: nil,
				Bytes:   b,
			})
			str := base64.StdEncoding.EncodeToString(bp)
			err = adb.AddConfiguration(&db.Configuration{
				Key:   "public_key",
				Value: str,
			})
			if err != nil {
				panic(err)
			}

			b, err = x509.MarshalPKCS8PrivateKey(kaytuPrivateKey)
			if err != nil {
				panic(err)
			}
			bp = pem.EncodeToMemory(&pem.Block{
				Type:    "RSA PRIVATE KEY",
				Headers: nil,
				Bytes:   b,
			})
			str = base64.StdEncoding.EncodeToString(bp)
			err = adb.AddConfiguration(&db.Configuration{
				Key:   "private_key",
				Value: str,
			})
			if err != nil {
				panic(err)
			}

		} else {
			for _, k := range keyPair {
				if k.Key == "public_key" {
					b, err := base64.StdEncoding.DecodeString(k.Value)
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
					kaytuPublicKey = pub.(*rsa.PublicKey)
				} else if k.Key == "private_key" {
					b, err := base64.StdEncoding.DecodeString(k.Value)
					if err != nil {
						return fmt.Errorf("private key decode: %w", err)
					}
					block, _ := pem.Decode(b)
					if block == nil {
						panic("failed to decode private key")
					}
					pri, err := x509.ParsePKCS8PrivateKey(block.Bytes)
					if err != nil {
						panic(err)
					}
					kaytuPrivateKey = pri.(*rsa.PrivateKey)
				}
			}
		}
	}

	auth0Service := auth0.New(auth0ManageDomain, auth0ClientID, auth0ManageClientID, auth0ManageClientSecret,
		auth0Connection, int(inviteTTL), adb)

	authServer := &Server{
		host:                    kaytuHost,
		kaytuPublicKey:          kaytuPublicKey,
		verifier:                verifier,
		verifierNative:          verifierNative,
		verifierPennywiseNative: verifierPennywiseNative,
		dexVerifier:             dexVerifier,
		logger:                  logger,
		workspaceClient:         workspaceClient,
		db:                      adb,
		auth0Service:            auth0Service,
		updateLoginUserList:     nil,
		updateLogin:             make(chan User, 100000),
	}
	go authServer.WorkspaceMapUpdater()
	go authServer.UpdateLastLoginLoop()

	errors := make(chan error, 1)
	go func() {
		routes := httpRoutes{
			logger: logger,
			//emailService:    m,
			workspaceClient: workspaceClient,
			metadataBaseUrl: metadataBaseUrl,
			auth0Service:    auth0Service,
			kaytuPrivateKey: kaytuPrivateKey,
			db:              adb,
			authServer:      authServer,
		}
		errors <- fmt.Errorf("http server: %w", httpserver.RegisterAndStart(ctx, logger, httpServerAddress, &routes))
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
		ca, err := os.ReadFile(caPath)
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
