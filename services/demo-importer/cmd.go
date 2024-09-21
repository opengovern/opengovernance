package demo_importer

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/config"
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

			logger.Info("Downloading file", zap.String("address", cnf.DemoDataS3URL))
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

			err = worker.ImportPsqlData(ctx, cnf, "/demo-data/postgres")
			if err != nil {
				return fmt.Errorf("failure while importing postgres databases: %w", err)
			}

			logger.Info("postgres data imported")

			err = worker.ImportJob(ctx, logger, es, "/demo-data/es-demo")
			if err != nil {
				return fmt.Errorf("failure while importing es indices: %w", err)
			}

			return nil
		},
	}

	return cmd
}
