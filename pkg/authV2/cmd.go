package authV2

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"strconv"

	config2 "github.com/opengovern/og-util/pkg/config"
	"github.com/opengovern/og-util/pkg/httpserver"
	"github.com/opengovern/og-util/pkg/postgres"
	"github.com/opengovern/opengovernance/pkg/authV2/db"

	"crypto/rand"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"google.golang.org/grpc/credentials"
)

var (

	dexAuthDomain                = os.Getenv("DEX_AUTH_DOMAIN")
	dexAuthPublicClientID        = os.Getenv("DEX_AUTH_PUBLIC_CLIENT_ID")
	dexGrpcAddress               = os.Getenv("DEX_GRPC_ADDR")
	
	httpServerAddress = os.Getenv("HTTP_ADDRESS")

	kaytuHost          = os.Getenv("KAYTU_HOST")
	kaytuKeyEnabledStr = os.Getenv("KAYTU_KEY_ENABLED")
	kaytuPublicKeyStr  = os.Getenv("KAYTU_PUBLIC_KEY")
	kaytuPrivateKeyStr = os.Getenv("KAYTU_PRIVATE_KEY")

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

	logger = logger.Named("authV2")



	dexVerifier, err := newDexOidcVerifier(ctx, dexAuthDomain, dexAuthPublicClientID)
	if err != nil {
		return fmt.Errorf("open id connect dex verifier: %w", err)
	}

	logger.Info("Instantiated a new Open ID Connect verifier")
	//m := email.NewSendGridClient(mailApiKey, mailSender, mailSenderName, logger)

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


	authServer := &Server{
		host:                    kaytuHost,
		kaytuPublicKey:          kaytuPublicKey,
		
		dexVerifier:             dexVerifier,
		logger:                  logger,
		db:                      adb,
		updateLoginUserList:     nil,
		updateLogin:             make(chan User, 100000),
	}

	go authServer.UpdateLastLoginLoop()

	errors := make(chan error, 1)
	go func() {
		routes := httpRoutes{
			logger: logger,
			kaytuPrivateKey:   kaytuPrivateKey,
			db:                adb,
			authServer:        authServer,
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