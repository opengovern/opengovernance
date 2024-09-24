package worker

import (
	"context"
	"fmt"
	"github.com/jackc/pgtype"
	"github.com/kaytu-io/open-governance/services/demo-importer/db"
	"github.com/kaytu-io/open-governance/services/demo-importer/db/model"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"go.uber.org/zap"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func ImportJob(ctx context.Context, logger *zap.Logger, migratorDb db.Database, client *opensearchapi.Client, dir string) error {
	indexConfigs, err := ReadIndexConfigs(dir)
	if err != nil {
		logger.Error("Error reading index configs", zap.Error(err))
		return err
	}
	logger.Info("Read Index Configs Done")

	m, err := migratorDb.GetMigration(model.MigrationJobName)
	if err != nil {
		logger.Error("Error reading migration job", zap.Error(err))
		return err
	}
	if m == nil {
		jp := pgtype.JSONB{}
		err = jp.Set([]byte(""))
		if err != nil {
			return err
		}
		m = &model.Migration{
			ID:             model.MigrationJobName,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
			AdditionalInfo: "",
			Status:         "Creating Indices",
			JobsStatus:     jp,
		}
		err = migratorDb.CreateMigration(m)
		if err != nil {
			return err
		}
	} else {
		jp := pgtype.JSONB{}
		err = jp.Set([]byte(""))
		if err != nil {
			return err
		}
		err = migratorDb.UpdateMigrationJob(model.MigrationJobName, "Creating Indices", jp)
		if err != nil {
			return err
		}
	}

	for indexName, config := range indexConfigs {
		err := CreateIndex(ctx, client, indexName, config.Settings, config.Mappings)
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
			go ProcessJSONFile(ctx, logger, client, file, indexName, &wg)
		} else {
			fmt.Println("No index config found for file: %s", file)
		}
	}

	m.Status = fmt.Sprintf("Importing Indices")

	err = migratorDb.UpdateMigrationJob(m.ID, m.Status, m.JobsStatus)
	if err != nil {
		return err
	}

	wg.Wait()

	fmt.Println("All indexing operations completed.")

	err = migratorDb.UpdateMigrationJob(m.ID, "COMPLETED", m.JobsStatus)
	if err != nil {
		return err
	}

	return nil
}
