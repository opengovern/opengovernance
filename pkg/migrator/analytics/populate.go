package analytics

import (
	"encoding/json"
	analyticsDB "github.com/kaytu-io/kaytu-engine/pkg/analytics/db"
	"github.com/kaytu-io/kaytu-util/pkg/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func PopulateDatabase(logger *zap.Logger, dbc *gorm.DB, analyticsPath string) error {
	err := filepath.Walk(analyticsPath, func(path string, info fs.FileInfo, err error) error {
		if strings.HasSuffix(path, ".json") {
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
			dbMetric := analyticsDB.AnalyticMetric{
				ID:         id,
				Connectors: connectors,
				Name:       metric.Name,
				Query:      metric.Query,
				Tags:       tags,
			}

			err = dbc.Model(&analyticsDB.AnalyticMetric{}).Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}},                                     // key column
				DoUpdates: clause.AssignmentColumns([]string{"connectors", "name", "query"}), // column needed to be updated
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
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
