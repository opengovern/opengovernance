package worker

import (
	"fmt"
	essdk "github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/open-governance/services/demo-importer/types"
	"github.com/labstack/gommon/log"
	"go.uber.org/zap"
	"path/filepath"
	"strings"
	"sync"
)

func ImportJob(logger *zap.Logger, client essdk.Client) error {
	osClient := client.ES()

	dir := types.ESBackupPath

	indexConfigs, err := ReadIndexConfigs(dir)
	if err != nil {
		logger.Error("Error reading index configs", zap.Error(err))
		return err
	}
	logger.Info("Read Index Configs Done")

	for indexName, config := range indexConfigs {
		err := CreateIndex(osClient, indexName, config.Settings, config.Mappings)
		if err != nil {
			logger.Error("Error creating index", zap.String("indexName", indexName), zap.Error(err))
			return err
		}
	}
	logger.Info("Create Indices Done")

	dataFiles, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		logger.Error("Error reading data files", zap.Error(err))
		return err
	}

	logger.Info("Read Data Files Done", zap.String("files", strings.Join(dataFiles, ",")))

	var wg sync.WaitGroup

	for _, file := range dataFiles {
		if strings.HasSuffix(file, ".mapping.json") || strings.HasSuffix(file, ".settings.json") {
			continue
		}

		indexName := strings.TrimSuffix(filepath.Base(file), ".json")
		if _, exists := indexConfigs[indexName]; exists {
			wg.Add(1)
			go ProcessJSONFile(logger, osClient, file, indexName, &wg)
		} else {
			log.Printf("No index config found for file: %s", file)
		}
	}

	wg.Wait()

	fmt.Println("All indexing operations completed.")

	return nil
}
