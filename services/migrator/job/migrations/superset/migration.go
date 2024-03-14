package superset

import (
	"fmt"
	"github.com/gruntwork-io/go-commons/random"
	"github.com/kaytu-io/kaytu-engine/services/migrator/config"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"strings"
)

type Migration struct {
}

func (m Migration) IsGitBased() bool {
	return false
}

func (m Migration) AttachmentFolderPath() string {
	// Make sure this migration always runs by creating a file in a directory that is not cleaned up
	err := os.MkdirAll("/tmp/superset", os.ModePerm)
	if err != nil {
		fmt.Println("failed to create superset directory", err)
	}
	randomText, _ := random.RandomString(128, random.LowerLetters+random.UpperLetters+random.Digits)
	err = os.WriteFile("/tmp/superset/superset.txt", []byte(randomText), 0644)
	if err != nil {
		fmt.Println("failed to create superset file", err)
	}
	return "/tmp/superset"
}

func (m Migration) Run(conf config.MigratorConfig, logger *zap.Logger) error {
	ssWrapper, err := newSupersetWrapper(logger, conf.SupersetBaseURL, conf.SupersetAdminPassword)
	if err != nil {
		return err
	}

	err = filepath.WalkDir(config.SuperSetGitPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			logger.Error("failed to walk path", zap.Error(err))
			return err
		}
		if d.IsDir() {
			return nil
		}
		// check if it's a zip file
		if strings.HasSuffix(path, ".zip") {
			logger.Info("importing dashboard", zap.String("path", path))
			err = ssWrapper.importDashboardV1(path, fmt.Sprintf("{\"databases/Steampipe.yaml\": \"%s\"}", conf.Steampipe.Password), true)
			if err != nil {
				logger.Error("failed to import", zap.Error(err), zap.String("path", path))
				return err
			}
		}
		return nil
	})
	if err != nil {
		logger.Error("failed to walk path", zap.Error(err))
		return err
	}

	dashboardsRes, err := ssWrapper.listDashboardsV1()
	if err != nil {
		logger.Error("failed to list dashboards", zap.Error(err))
		return err
	}

	for _, dashboard := range dashboardsRes.Result {
		err = ssWrapper.enableEmbeddingV1(dashboard.Id)
		if err != nil {
			logger.Error("failed to enable embedding", zap.Error(err), zap.Any("dashboard", dashboard))
		}
		logger.Info("enabled embedding", zap.String("dashboard_title", dashboard.DashboardTitle))
	}

	databasesResult, err := ssWrapper.listDatabaseV1()
	if err != nil {
		logger.Error("failed to list databases", zap.Error(err))
		return err
	}

	steampipeDbId := -1
	for _, db := range databasesResult.Result {
		if db.DatabaseName == "Steampipe" {
			steampipeDbId = db.ID
		}
	}

	createDatabaseRequest := createDatabaseV1Request{}
	createDatabaseRequest.DatabaseName = "Steampipe"
	createDatabaseRequest.Engine = "postgresql"
	createDatabaseRequest.ConfigurationMethod = "dynamic_form"
	createDatabaseRequest.EngineInformation.DisableSSHTunneling = false
	createDatabaseRequest.EngineInformation.SupportsFileUpload = true
	createDatabaseRequest.Driver = "psycopg2"
	createDatabaseRequest.SqlAlchemyUriPlaceholder = "postgresql://user:password@host:port/dbname[?key=value&key=value...]"
	createDatabaseRequest.Extra = "{\"allows_virtual_table_explore\":true}"
	createDatabaseRequest.ExposeInSqllab = true
	createDatabaseRequest.Parameters.Host = conf.Steampipe.Host
	createDatabaseRequest.Parameters.Port = conf.Steampipe.Port
	createDatabaseRequest.Parameters.Database = conf.Steampipe.DB
	createDatabaseRequest.Parameters.Username = conf.Steampipe.Username
	createDatabaseRequest.Parameters.Password = conf.Steampipe.Password
	createDatabaseRequest.MaskedEncryptedExtra = "{}"

	if steampipeDbId == -1 {
		err = ssWrapper.createDatabaseV1(createDatabaseRequest)
		if err != nil {
			logger.Error("failed to create database", zap.Error(err))
			return err
		}
		logger.Info("created database", zap.String("database_name", "Steampipe"))
	} else {
		err = ssWrapper.updateDatabaseV1(steampipeDbId, createDatabaseRequest)
		if err != nil {
			logger.Error("failed to update database", zap.Error(err))
			return err
		}
		logger.Info("updated database", zap.String("database_name", "Steampipe"))
	}

	return nil
}
