package demo_importer

import (
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	es "github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/open-governance/services/demo-importer/fetch"
	"github.com/kaytu-io/open-governance/services/demo-importer/types"
	"github.com/kaytu-io/open-governance/services/demo-importer/worker"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
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
			err = os.MkdirAll(types.DemoDataPath, os.ModePerm)
			if err != nil {
				return fmt.Errorf("failure creating path: %w", err)
			}

			esClient, err := es.NewClient(es.ClientConfig{
				Addresses:    []string{cnf.ElasticSearch.Address},
				Username:     &cnf.ElasticSearch.Username,
				Password:     &cnf.ElasticSearch.Password,
				IsOpenSearch: &cnf.ElasticSearch.IsOpenSearch,
				IsOnAks:      &cnf.ElasticSearch.IsOnAks,
				ExternalID:   &cnf.ElasticSearch.ExternalID,
			})
			if err != nil {
				return fmt.Errorf("failure while creating ES Client: %w", err)
			}

			cmd.SilenceUsage = true

			logger.Info("Downloading file", zap.String("address", cnf.DemoDataS3URL))
			filePath, err := fetch.DownloadS3Object(cnf.DemoDataS3URL)
			if err != nil {
				return err
			}

			logger.Info("File Downloaded", zap.String("file", filePath))

			decryptedData, err := fetch.DecryptString(filePath, cnf.OpensslPassword)

			logger.Info("Successfully decrypted", zap.String("file", filePath))

			err = os.WriteFile(types.DemoDecryptedDataFilePath, decryptedData, 0644)
			if err != nil {
				return err
			}

			logger.Info("Successfully decrypted file written", zap.String("file", filePath))

			file, err := os.Open(types.DemoDecryptedDataFilePath)
			if err != nil {
				return err
			}
			defer file.Close()

			err = fetch.Unzip(file)
			if err != nil {
				return fmt.Errorf("failure while unzipping file: %w", err)
			}

			logger.Info("Successfully unzipped", zap.String("file", filePath))

			files, err := os.ReadDir(types.DemoDataPath)
			if err != nil {
				return fmt.Errorf("failure while reading directory: %w", err)
			}

			for _, file := range files {
				if file.IsDir() {
					logger.Info(fmt.Sprintf("[DIR]  %s\n", file.Name()))
				} else {
					logger.Info(fmt.Sprintf("[FILE] %s\n", file.Name()))
				}
			}

			err = worker.ImportJob(logger, esClient)
			if err != nil {
				return fmt.Errorf("failure while importing job: %w", err)
			}

			return nil
		},
	}

	return cmd
}
