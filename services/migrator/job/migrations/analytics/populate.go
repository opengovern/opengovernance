package analytics

import (
	"context"
	"errors"
	"fmt"
	"github.com/goccy/go-yaml"
	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	analyticsDB "github.com/kaytu-io/open-governance/pkg/analytics/db"
	"github.com/kaytu-io/open-governance/pkg/inventory"
	"github.com/kaytu-io/open-governance/services/migrator/config"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

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
		return fmt.Errorf("new postgres client: %w", err)
	}

	err = filepath.Walk(config.AssetsGitPath, func(path string, info fs.FileInfo, err error) error {
		if strings.HasSuffix(path, ".yaml") {
			return populateItem(logger, orm, path, info, true)
		}
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	err = filepath.Walk(config.SpendGitPath, func(path string, info fs.FileInfo, err error) error {
		if strings.HasSuffix(path, ".yaml") {
			return populateItem(logger, orm, path, info, false)
		}
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	err = filepath.Walk(config.FinderPopularGitPath, func(path string, info fs.FileInfo, err error) error {
		if strings.HasSuffix(path, ".yaml") {
			return populateFinderItem(logger, orm, path, info, true)
		}
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	err = filepath.Walk(config.FinderOthersGitPath, func(path string, info fs.FileInfo, err error) error {
		if strings.HasSuffix(path, ".yaml") {
			return populateFinderItem(logger, orm, path, info, false)
		}
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
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

	var connectors []string
	for _, c := range metric.Connectors {
		connectors = append(connectors, c.String())
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
			metric.FinderQuery = fmt.Sprintf(`select * from kaytu_cost where service_name in (%s)`, strings.Join(tarr, ","))
			metric.FinderPerConnectionQuery = fmt.Sprintf(`select * from kaytu_cost where service_name in (%s) and connection_id IN (<CONNECTION_ID_LIST>)`, strings.Join(tarr, ","))
		} else {
			metric.FinderQuery = fmt.Sprintf(`select * from kaytu_lookup where resource_type in (%s)`, strings.Join(tarr, ","))
			metric.FinderPerConnectionQuery = fmt.Sprintf(`select * from kaytu_lookup where resource_type in (%s) and connection_id IN (<CONNECTION_ID_LIST>)`, strings.Join(tarr, ","))
		}
	}

	dbMetric := analyticsDB.AnalyticMetric{
		ID:                       id,
		Connectors:               connectors,
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
		DoUpdates: clause.AssignmentColumns([]string{"connectors", "name", "query",
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

func populateFinderItem(logger *zap.Logger, dbc *gorm.DB, path string, info fs.FileInfo, isPopular bool) error {
	id := strings.TrimSuffix(info.Name(), ".yaml")

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var item SmartQuery
	err = yaml.Unmarshal(content, &item)
	if err != nil {
		logger.Error("failure in unmarshal", zap.String("path", path), zap.Error(err))
		return err
	}

	var connectors []string
	for _, c := range item.Connectors {
		connectors = append(connectors, string(c))
	}

	tx := dbc.Begin()
	defer tx.Rollback()

	logger.Info("Query Update", zap.String("id", id), zap.Any("tags", item.Tags))

	err = tx.Model(&inventory.SmartQuery{}).Where("id = ?", id).Unscoped().Delete(&inventory.SmartQuery{}).Error
	if err != nil {
		logger.Error("failure in deleting SmartQuery", zap.String("id", id), zap.Error(err))
		return err
	}

	err = tx.Model(&inventory.SmartQueryTag{}).Where("smart_query_id = ?", id).Unscoped().Delete(&inventory.SmartQueryTag{}).Error
	if err != nil {
		logger.Error("failure in deleting SmartQueryTag", zap.String("smart_query_id", id), zap.Error(err))
		return err
	}

	dbMetric := inventory.SmartQuery{
		ID:          id,
		Connectors:  connectors,
		Title:       item.Title,
		Description: item.Description,
		Engine:      item.Query.Engine,
		Query:       item.Query.QueryToExecute,
		IsPopular:   isPopular,
	}

	tags := make([]inventory.SmartQueryTag, 0, len(item.Tags))
	for k, v := range item.Tags {
		tag := inventory.SmartQueryTag{
			SmartQueryID: id,
			Tag: model.Tag{
				Key:   k,
				Value: v,
			},
		}
		tags = append(tags, tag)
	}

	err = tx.Model(&inventory.SmartQuery{}).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},                                                                             // key column
		DoUpdates: clause.AssignmentColumns([]string{"connectors", "title", "query", "description", "engine", "is_popular"}), // column needed to be updated
	}).Create(dbMetric).Error
	if err != nil {
		logger.Error("failure in insert query", zap.Error(err))
		return err
	}

	logger.Info("parsed the tags", zap.String("id", id), zap.Any("tags", tags))

	if len(tags) > 0 {
		for _, tag := range tags {
			err = tx.Model(&inventory.SmartQueryTag{}).Create(&tag).Error
			if err != nil {
				logger.Error("failure in insert tags", zap.Error(err))
				return err
			}
		}
	}
	logger.Info("inserted tags", zap.String("id", id))
	err = tx.Commit().Error
	if err != nil {
		logger.Error("failure in commit", zap.Error(err))
		return err
	}
	logger.Info("commit finish", zap.String("id", id))

	return nil
}
