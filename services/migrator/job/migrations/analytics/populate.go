package analytics

import (
	"context"
	"errors"
	"fmt"
	"github.com/goccy/go-yaml"
	"github.com/opengovern/og-util/pkg/model"
	"github.com/opengovern/og-util/pkg/postgres"
	analyticsDB "github.com/opengovern/opengovernance/pkg/analytics/db"
	"github.com/opengovern/opengovernance/pkg/inventory"
	"github.com/opengovern/opengovernance/pkg/metadata/models"
	"github.com/opengovern/opengovernance/services/migrator/config"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var QueryParameters []models.QueryParameter

type Migration struct {
}

func (m Migration) IsGitBased() bool {
	return true
}

func (m Migration) AttachmentFolderPath() string {
	return ""
}

func (m Migration) Run(ctx context.Context, conf config.MigratorConfig, logger *zap.Logger) error {
	orm, err := postgres.NewClient(&postgres.Config{
		Host:    conf.PostgreSQL.Host,
		Port:    conf.PostgreSQL.Port,
		User:    conf.PostgreSQL.Username,
		Passwd:  conf.PostgreSQL.Password,
		DB:      "inventory",
		SSLMode: conf.PostgreSQL.SSLMode,
	}, logger)
	if err != nil {
		return fmt.Errorf("new inventory postgres client: %w", err)
	}

	err = filepath.Walk(config.AssetsGitPath, func(path string, info fs.FileInfo, err error) error {
		if strings.HasSuffix(path, ".yaml") {
			return populateItem(logger, orm, path, info, true)
		}
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		logger.Error("failed to get assets", zap.Error(err))
		return err
	}

	err = filepath.Walk(config.SpendGitPath, func(path string, info fs.FileInfo, err error) error {
		if strings.HasSuffix(path, ".yaml") {
			return populateItem(logger, orm, path, info, false)
		}
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		logger.Error("failed to get spends", zap.Error(err))
		return err
	}

	err = filepath.Walk(config.QueriesGitPath, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(path, ".yaml") {
			return populateFinderItem(logger, orm, path, info)
		}
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		logger.Error("failed to get queries", zap.Error(err))
		return err
	}

	metadataOrm, err := postgres.NewClient(&postgres.Config{
		Host:    conf.PostgreSQL.Host,
		Port:    conf.PostgreSQL.Port,
		User:    conf.PostgreSQL.Username,
		Passwd:  conf.PostgreSQL.Password,
		DB:      "metadata",
		SSLMode: conf.PostgreSQL.SSLMode,
	}, logger)
	if err != nil {
		return fmt.Errorf("new metadata postgres client: %w", err)
	}

	err = metadataOrm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, obj := range QueryParameters {
			err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "key"}}, // key column
				DoUpdates: clause.AssignmentColumns([]string{"value"}),
			}).Create(&obj).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		logger.Error("failed to insert query params", zap.Error(err))
		return err
	}

	return nil
}

func populateItem(logger *zap.Logger, dbc *gorm.DB, path string, info fs.FileInfo, isAsset bool) error {
	id := strings.TrimSuffix(info.Name(), ".yaml")
	if !isAsset {
		id = "spend_" + id
	}

	content, err := os.ReadFile(path)
	if err != nil {
		logger.Error("failure in reading file", zap.String("path", path), zap.Error(err))
		return err
	}

	var metric Metric
	err = yaml.Unmarshal(content, &metric)
	if err != nil {
		logger.Error("failure in unmarshal", zap.String("path", path), zap.Error(err))
		return err
	}

	var integrationTypes []string
	for _, c := range metric.IntegrationTypes {
		integrationTypes = append(integrationTypes, c.String())
	}

	var tags []analyticsDB.MetricTag
	for k, v := range metric.Tags {
		tags = append(tags, analyticsDB.MetricTag{
			Tag: model.Tag{
				Key:   k,
				Value: v,
			},
			ID: id,
		})
	}

	metricType := analyticsDB.MetricTypeAssets
	if !isAsset {
		metricType = analyticsDB.MetricTypeSpend
	}

	awsR := regexp.MustCompile(`'(aws::[\w\d]+::[\w\d]+)'`)
	azureR := regexp.MustCompile(`'(microsoft.[\w\d/]+)'`)

	if metric.Tables == nil || len(metric.Tables) == 0 {
		awsTables := awsR.FindAllString(metric.Query, -1)
		azureTables := azureR.FindAllString(metric.Query, -1)
		for _, t := range awsTables {
			t = strings.Trim(t, "'")
			metric.Tables = append(metric.Tables, t)
		}
		for _, t := range azureTables {
			t = strings.Trim(t, "'")
			metric.Tables = append(metric.Tables, t)
		}
	}

	if len(metric.FinderQuery) == 0 {
		var tarr []string
		for _, t := range metric.Tables {
			tarr = append(tarr, fmt.Sprintf("'%s'", t))
		}
		if metricType == analyticsDB.MetricTypeSpend {
			metric.FinderQuery = fmt.Sprintf(`select * from platform_cost where service_name in (%s)`, strings.Join(tarr, ","))
			metric.FinderPerConnectionQuery = fmt.Sprintf(`select * from platform_cost where service_name in (%s) and connection_id IN (<CONNECTION_ID_LIST>)`, strings.Join(tarr, ","))
		} else {
			metric.FinderQuery = fmt.Sprintf(`select * from platform_lookup where resource_type in (%s)`, strings.Join(tarr, ","))
			metric.FinderPerConnectionQuery = fmt.Sprintf(`select * from platform_lookup where resource_type in (%s) and connection_id IN (<CONNECTION_ID_LIST>)`, strings.Join(tarr, ","))
		}
	}

	dbMetric := analyticsDB.AnalyticMetric{
		ID:                       id,
		IntegrationTypes:         integrationTypes,
		Type:                     metricType,
		Name:                     metric.Name,
		Query:                    metric.Query,
		Tables:                   metric.Tables,
		FinderQuery:              metric.FinderQuery,
		FinderPerConnectionQuery: metric.FinderPerConnectionQuery,
		Status:                   analyticsDB.AnalyticMetricStatus(metric.Status),
		Tags:                     tags,
	}

	err = dbc.Model(&analyticsDB.AnalyticMetric{}).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}}, // key column
		DoUpdates: clause.AssignmentColumns([]string{"integration_types", "name", "query",
			"tables", "finder_query", "finder_per_connection_query", "type", "status"}), // column needed to be updated
	}).Create(dbMetric).Error

	if err != nil {
		logger.Error("failure in insert", zap.String("path", path), zap.Error(err))
		return err
	}

	for _, t := range dbMetric.Tags {
		err = dbc.Model(&analyticsDB.MetricTag{}).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}, {Name: "id"}}, // key column
			DoUpdates: clause.AssignmentColumns([]string{"value"}),  // column needed to be updated
		}).Create(t).Error
	}
	return nil
}

func populateFinderItem(logger *zap.Logger, dbc *gorm.DB, path string, info fs.FileInfo) error {
	id := strings.TrimSuffix(info.Name(), ".yaml")

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var item NamedQuery
	err = yaml.Unmarshal(content, &item)
	if err != nil {
		logger.Error("failure in unmarshal", zap.String("path", path), zap.Error(err))
		return err
	}

	var integrationTypes []string
	for _, c := range item.IntegrationTypes {
		integrationTypes = append(integrationTypes, string(c))
	}

	tx := dbc.Begin()
	defer tx.Rollback()

	// logger.Info("Query Update", zap.String("id", id), zap.Any("tags", item.Tags))

	err = tx.Model(&inventory.NamedQuery{}).Where("id = ?", id).Unscoped().Delete(&inventory.NamedQuery{}).Error
	if err != nil {
		logger.Error("failure in deleting NamedQuery", zap.String("id", id), zap.Error(err))
		return err
	}

	err = tx.Model(&inventory.NamedQueryTag{}).Where("named_query_id = ?", id).Unscoped().Delete(&inventory.NamedQueryTag{}).Error
	if err != nil {
		logger.Error("failure in deleting NamedQueryTag", zap.String("named_query_id", id), zap.Error(err))
		return err
	}

	err = tx.Model(&inventory.QueryParameter{}).Where("query_id = ?", id).Unscoped().Delete(&inventory.QueryParameter{}).Error
	if err != nil {
		logger.Error("failure in deleting QueryParameter", zap.String("id", id), zap.Error(err))
		return err
	}
	err = tx.Model(&inventory.Query{}).Where("id = ?", id).Unscoped().Delete(&inventory.Query{}).Error
	if err != nil {
		logger.Error("failure in deleting Query", zap.String("id", id), zap.Error(err))
		return err
	}

	isBookmarked := false
	tags := make([]inventory.NamedQueryTag, 0, len(item.Tags))
	for k, v := range item.Tags {
		if k == "platform_queries_bookmark" {
			isBookmarked = true
		}
		tag := inventory.NamedQueryTag{
			NamedQueryID: id,
			Tag: model.Tag{
				Key:   k,
				Value: v,
			},
		}
		tags = append(tags, tag)
	}

	dbMetric := inventory.NamedQuery{
		ID:               id,
		IntegrationTypes: integrationTypes,
		Title:            item.Title,
		Description:      item.Description,
		IsBookmarked:     isBookmarked,
		QueryID:          &id,
	}
	queryParams := []inventory.QueryParameter{}
	for _, qp := range item.Query.Parameters {
		queryParams = append(queryParams, inventory.QueryParameter{
			Key:      qp.Key,
			Required: qp.Required,
			QueryID:  dbMetric.ID,
		})
		if qp.DefaultValue != nil {
			queryParamObj := models.QueryParameter{
				Key:   qp.Key,
				Value: *qp.DefaultValue,
			}
			QueryParameters = append(QueryParameters, queryParamObj)
		}
	}
	query := inventory.Query{
		ID:             dbMetric.ID,
		QueryToExecute: item.Query.QueryToExecute,
		PrimaryTable:   item.Query.PrimaryTable,
		ListOfTables:   item.Query.ListOfTables,
		Engine:         item.Query.Engine,
		Parameters:     queryParams,
		Global:         item.Query.Global,
	}
	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}}, // key column
		DoNothing: true,
	}).Create(&query).Error
	if err != nil {
		logger.Error("failure in Creating Query", zap.String("query_id", id), zap.Error(err))
		return err
	}
	for _, param := range query.Parameters {
		err = tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}, {Name: "query_id"}}, // key columns
			DoNothing: true,
		}).Create(&param).Error
		if err != nil {
			return fmt.Errorf("failure in query parameter insert: %v", err)
		}
	}

	err = tx.Model(&inventory.NamedQuery{}).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}}, // key column
		DoNothing: true,                          // column needed to be updated
	}).Create(dbMetric).Error
	if err != nil {
		logger.Error("failure in insert query", zap.Error(err))
		return err
	}

	// logger.Info("parsed the tags", zap.String("id", id), zap.Any("tags", tags))

	if len(tags) > 0 {
		for _, tag := range tags {
			err = tx.Model(&inventory.NamedQueryTag{}).Create(&tag).Error
			if err != nil {
				logger.Error("failure in insert tags", zap.Error(err))
				return err
			}
		}
	}
	// logger.Info("inserted tags", zap.String("id", id))
	err = tx.Commit().Error
	if err != nil {
		logger.Error("failure in commit", zap.Error(err))
		return err
	}
	// logger.Info("commit finish", zap.String("id", id))

	return nil
}
