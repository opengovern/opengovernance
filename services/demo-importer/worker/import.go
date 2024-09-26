package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgtype"
	"github.com/kaytu-io/open-governance/services/demo-importer/db"
	"github.com/kaytu-io/open-governance/services/demo-importer/db/model"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"go.uber.org/zap"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
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
		jobsStatusJson, err := json.Marshal(model.ESImportProgress{
			Progress: 0,
		})
		if err != nil {
			return err
		}
		jp := pgtype.JSONB{}
		err = jp.Set(jobsStatusJson)
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
		jobsStatus := model.ESImportProgress{
			Progress: 0,
		}
		err = updateJob(migratorDb, m, "Creating Indices", jobsStatus)
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
	var totalTasks int64
	var completedTasks int64

	for _, file := range dataFiles {
		if strings.HasSuffix(file, ".mapping.json") || strings.HasSuffix(file, ".settings.json") {
			continue
		}

		indexName := strings.TrimSuffix(filepath.Base(file), ".json")
		if _, exists := indexConfigs[indexName]; exists {
			atomic.AddInt64(&totalTasks, 1)
			wg.Add(1)

			go func(file, indexName string) {
				defer wg.Done()
				ProcessJSONFile(ctx, logger, client, file, indexName)

				atomic.AddInt64(&completedTasks, 1)

				m.Status = fmt.Sprintf("Importing Indices")
				jobsStatus := model.ESImportProgress{
					Progress: float64(completedTasks) / float64(totalTasks),
				}
				err = updateJob(migratorDb, m, m.Status, jobsStatus)
				if err != nil {
					fmt.Println("Error updating migration job:", err.Error())
				}
				fmt.Printf("Completed %d/%d tasks\n", completedTasks, totalTasks)
			}(file, indexName)
		} else {
			fmt.Println("No index config found for file: %s", file)
		}
	}

	wg.Wait()

	fmt.Println("All indexing operations completed.")

	jobsStatus := model.ESImportProgress{
		Progress: float64(completedTasks) / float64(totalTasks),
	}
	err = updateJob(migratorDb, m, "COMPLETED", jobsStatus)
	if err != nil {
		return err
	}

	return nil
}

func updateJob(migratorDb db.Database, m *model.Migration, status string, jobsStatus model.ESImportProgress) error {
	jobsStatusJson, err := json.Marshal(jobsStatus)
	if err != nil {
		return err
	}

	jp := pgtype.JSONB{}
	err = jp.Set(jobsStatusJson)
	if err != nil {
		return err
	}
	m.JobsStatus = jp
	m.Status = status

	err = migratorDb.UpdateMigrationJob(m.ID, m.Status, m.JobsStatus)
	if err != nil {
		return err
	}
	return nil
}
