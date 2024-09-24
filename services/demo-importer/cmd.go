package demo_importer

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/api"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/kaytu-io/kaytu-util/pkg/httpclient"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/kaytu-io/open-governance/pkg/metadata/client"
	"github.com/kaytu-io/open-governance/pkg/metadata/models"
	"github.com/kaytu-io/open-governance/pkg/workspace/db"
	db2 "github.com/kaytu-io/open-governance/services/demo-importer/db"
	"github.com/kaytu-io/open-governance/services/demo-importer/fetch"
	"github.com/kaytu-io/open-governance/services/demo-importer/types"
	"github.com/kaytu-io/open-governance/services/demo-importer/worker"
	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"net/http"
	"os"
)

func Command() *cobra.Command {
	var (
		cnf types.DemoImporterConfig
	)
	config.ReadFromEnv(&cnf, nil)
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	logger.Info("running", zap.String("es_address", cnf.ElasticSearch.Address), zap.String("es_arn", cnf.ElasticSearch.AssumeRoleArn))

	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			err = os.MkdirAll(types.DemoDataPath, os.ModePerm)
			if err != nil {
				return fmt.Errorf("failure creating path: %w", err)
			}
			psqlCfg := postgres.Config{
				Host:    cnf.PostgreSQL.Host,
				Port:    cnf.PostgreSQL.Port,
				User:    cnf.PostgreSQL.Username,
				Passwd:  cnf.PostgreSQL.Password,
				DB:      "workspace",
				SSLMode: cnf.PostgreSQL.SSLMode,
			}
			orm, err := postgres.NewClient(&psqlCfg, logger)
			if err != nil {
				return fmt.Errorf("new postgres client: %w", err)
			}

			cfg := opensearchapi.Config{
				opensearch.Config{
					Addresses:           []string{cnf.ElasticSearch.Address},
					Username:            cnf.ElasticSearch.Username,
					Password:            cnf.ElasticSearch.Password,
					CompressRequestBody: true,
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{
							InsecureSkipVerify: true,
						},
					},
				},
			}
			es, err := opensearchapi.NewClient(cfg)

			cmd.SilenceUsage = true

			metadataClient := client.NewMetadataServiceClient(cnf.Metadata.BaseURL)

			s3Url := cnf.DemoDataS3URL
			value, err := metadataClient.GetConfigMetadata(&httpclient.Context{
				UserRole: api.AdminRole,
			}, models.DemoDataS3URL)
			if err == nil && len(value.GetValue().(string)) > 0 {
				s3Url = value.GetValue().(string)
			} else if err != nil {
				logger.Error("failed to get demo data s3 url from metadata", zap.Error(err))
			}

			logger.Info("Downloading file", zap.String("address", s3Url))
			filePath, err := fetch.DownloadS3Object(cnf.DemoDataS3URL)
			if err != nil {
				return err
			}

			logger.Info("File Downloaded", zap.String("file", filePath))

			decryptedData, err := fetch.DecryptString(filePath, cnf.OpensslPassword)

			logger.Info("Successfully decrypted", zap.String("file", filePath))

			decryptedFile := types.DemoDecryptedDataFilePath
			err = os.WriteFile(decryptedFile, decryptedData, 0644)
			if err != nil {
				return err
			}

			logger.Info("Successfully decrypted file written", zap.String("file", filePath))

			file, err := os.Open(decryptedFile)
			if err != nil {
				return err
			}
			defer file.Close()

			err = fetch.Unzip(file)
			if err != nil {
				return fmt.Errorf("failure while unzipping file: %w", err)
			}

			logger.Info("Successfully unzipped", zap.String("file", filePath))

			psqlMigratorCfg := postgres.Config{
				Host:    cnf.PostgreSQL.Host,
				Port:    cnf.PostgreSQL.Port,
				User:    cnf.PostgreSQL.Username,
				Passwd:  cnf.PostgreSQL.Password,
				DB:      "migrator",
				SSLMode: cnf.PostgreSQL.SSLMode,
			}
			migratorOrm, err := postgres.NewClient(&psqlMigratorCfg, logger)
			if err != nil {
				return fmt.Errorf("new postgres client: %w", err)
			}

			migratorDb := db2.Database{
				ORM: migratorOrm,
			}
			err = migratorDb.Initialize()
			if err != nil {
				return fmt.Errorf("failure while initializing database: %w", err)
			}

			err = worker.ImportJob(ctx, logger, migratorDb, es, "/demo-data/es-demo")
			if err != nil {
				return fmt.Errorf("failure while importing es indices: %w", err)
			}

			var workspaces []db.Workspace
			err = orm.Model(&db.Workspace{}).Find(&workspaces).Error
			if err != nil {
				return fmt.Errorf("failure while getting workspaces: %w", err)
			}
			fmt.Println("Workspaces: ", workspaces)

			err = orm.Model(&db.Workspace{}).Where("name = 'main'").Update("contain_sample_data", true).Error
			if err != nil {
				return fmt.Errorf("failure while updating main workspace: %w", err)
			}

			return nil
		},
	}

	return cmd
}
