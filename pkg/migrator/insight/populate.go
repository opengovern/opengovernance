package insight

import (
	"fmt"
	"path"

	"github.com/kaytu-io/kaytu-engine/pkg/compliance/db"
	"github.com/kaytu-io/kaytu-engine/pkg/migrator/internal"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func PopulateDatabase(logger *zap.Logger, dbc *gorm.DB, insightsPath string) error {
	p := GitParser{}
	if err := p.ExtractInsights(path.Join(insightsPath, internal.InsightsSubPath)); err != nil {
		return err
	}
	if err := p.ExtractInsightGroups(path.Join(insightsPath, internal.InsightGroupsSubPath)); err != nil {
		return err
	}

	err := dbc.Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&db.InsightGroupInsight{}).Where("1=1").Unscoped().Delete(&db.InsightGroupInsight{}).Error
		if err != nil {
			logger.Error("failure in delete", zap.Error(err))
			return err
		}
		err = tx.Model(&db.InsightGroup{}).Where("1=1").Unscoped().Delete(&db.InsightGroup{}).Error
		if err != nil {
			logger.Error("failure in delete", zap.Error(err))
			return err
		}
		err = tx.Model(&db.InsightTag{}).Where("1=1").Unscoped().Delete(&db.InsightTag{}).Error
		if err != nil {
			logger.Error("failure in delete insight tags", zap.Error(err))
			return err
		}
		err = tx.Model(&db.Insight{}).Where("1=1").Unscoped().Delete(&db.Insight{}).Error
		if err != nil {
			logger.Error("failure in delete insights", zap.Error(err))
			return err
		}

		for _, obj := range p.insights {
			err = tx.Model(&db.Insight{}).Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "id"}}, // key column
				DoUpdates: clause.AssignmentColumns([]string{"query_id", "connector", "short_title", "long_title",
					"description", "logo_url", "links", "enabled", "internal"}), // column needed to be updated
			}).Create(map[string]any{
				"id":          obj.ID,
				"query_id":    obj.QueryID,
				"connector":   obj.Connector,
				"short_title": obj.ShortTitle,
				"long_title":  obj.LongTitle,
				"description": obj.Description,
				"logo_url":    obj.LogoURL,
				"links":       obj.Links,
				"enabled":     obj.Enabled,
				"internal":    obj.Internal,
			}).Error
			if err != nil {
				logger.Error("failure in insert", zap.Error(err))
				return err
			}
			for _, tag := range obj.Tags {
				err = tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "key"}, {Name: "insight_id"}}, // key columns
					DoUpdates: clause.AssignmentColumns([]string{"key", "value"}),   // column needed to be updated
				}).Create(&tag).Error
			}
			if err != nil {
				logger.Error("failure in tag insert", zap.Error(err))
				return err
			}
		}

		for _, obj := range p.insightGroups {
			insightIDsList := make([]uint, 0, len(obj.Insights))
			for _, insight := range obj.Insights {
				insightIDsList = append(insightIDsList, insight.ID)
			}
			obj.Insights = nil
			err = tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}},                                                              // key column
				DoUpdates: clause.AssignmentColumns([]string{"short_title", "long_title", "description", "logo_url"}), // column needed to be updated
			}).Create(&obj).Error
			if err != nil {
				logger.Error("failure in insert", zap.Error(err))
				return err
			}

			for _, insightID := range insightIDsList {
				err = tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&db.InsightGroupInsight{
					InsightGroupID: obj.ID,
					InsightID:      insightID,
				}).Error
				if err != nil {
					logger.Error("failure in insert", zap.Error(err))
					return err
				}
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failure in transaction: %v", err)
	}

	return nil
}
