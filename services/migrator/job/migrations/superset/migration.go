package superset

import (
	"fmt"
	"github.com/gruntwork-io/go-commons/random"
	"github.com/kaytu-io/kaytu-engine/services/migrator/config"
	"go.uber.org/zap"
	"os"
	"path/filepath"
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

	databasesResult, err := ssWrapper.listDatabaseV1()
	if err != nil {
		logger.Error("failed to list databases", zap.Error(err))
		return err
	}

	for _, database := range databasesResult.Result {
		if database.DatabaseName == "Steampipe" {
			return nil
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

	err = ssWrapper.createDatabaseV1(createDatabaseRequest)
	if err != nil {
		logger.Error("failed to create database", zap.Error(err))
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
		if filepath.Ext(path) == ".zip" {
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

	return nil
}
