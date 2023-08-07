package analytics

import (
	"encoding/json"
	"fmt"
	analyticsDB "github.com/kaytu-io/kaytu-engine/pkg/analytics/db"
	"github.com/kaytu-io/kaytu-util/pkg/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func PopulateDatabase(logger *zap.Logger, dbc *gorm.DB, analyticsPath string) error {
	err := filepath.Walk(analyticsPath+"/assets", func(path string, info fs.FileInfo, err error) error {
		if strings.HasSuffix(path, ".json") {
			return PopulateItem(logger, dbc, path, info, true)
		}
		return nil
	})
	if err != nil {
		return err
	}

	err = filepath.Walk(analyticsPath+"/spend", func(path string, info fs.FileInfo, err error) error {
		if strings.HasSuffix(path, ".json") {
			return PopulateItem(logger, dbc, path, info, false)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func PopulateItem(logger *zap.Logger, dbc *gorm.DB, path string, info fs.FileInfo, isAsset bool) error {
	id := strings.TrimSuffix(info.Name(), ".json")

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var metric Metric
	err = json.Unmarshal(content, &metric)
	if err != nil {
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
		metric.FinderQuery = fmt.Sprintf(`select * from kaytu_lookup where resource_type in (%s)`, strings.Join(tarr, ","))
	}

	dbMetric := analyticsDB.AnalyticMetric{
		ID:          id,
		Connectors:  connectors,
		Type:        metricType,
		Name:        metric.Name,
		Query:       metric.Query,
		Tables:      metric.Tables,
		FinderQuery: metric.FinderQuery,
		Tags:        tags,
	}

	err = dbc.Model(&analyticsDB.AnalyticMetric{}).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}}, // key column
		DoUpdates: clause.AssignmentColumns([]string{"connectors", "name", "query",
			"tables", "finder_query"}), // column needed to be updated
	}).Create(dbMetric).Error

	if err != nil {
		logger.Error("failure in insert", zap.Error(err))
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
