package insight

import (
	"fmt"
	"path"

	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/db"
	"gitlab.com/keibiengine/keibi-engine/pkg/migrator/internal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func PopulateDatabase(dbc *gorm.DB, insightsPath string) error {
	p := GitParser{}
	if err := p.ExtractInsights(path.Join(insightsPath, internal.InsightsSubPath)); err != nil {
		return err
	}
	if err := p.ExtractInsightGroups(path.Join(insightsPath, internal.InsightGroupsSubPath)); err != nil {
		return err
	}

	for _, obj := range p.insights {
		err := dbc.Model(&db.Insight{}).Clauses(clause.OnConflict{
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
			return fmt.Errorf("failure in insert: %v", err)
		}
		for _, tag := range obj.Tags {
			err = dbc.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "key"}, {Name: "insight_id"}}, // key columns
				DoUpdates: clause.AssignmentColumns([]string{"key", "value"}),   // column needed to be updated
			}).Create(&tag).Error
		}
		if err != nil {
			return fmt.Errorf("failure in tag insert: %v", err)
		}
	}

	for _, obj := range p.insightGroups {
		err := dbc.Model(&db.InsightGroupInsight{}).Where("insight_group_id = ?", obj.ID).Delete(&db.InsightGroupInsight{}).Error
		if err != nil {
			return fmt.Errorf("failure in delete: %v", err)
		}
		err = dbc.Model(&db.InsightGroup{}).Where("id = ?", obj.ID).Delete(&db.InsightGroup{}).Error
		if err != nil {
			return fmt.Errorf("failure in delete: %v", err)
		}
		insightIDsList := make([]uint, 0, len(obj.Insights))
		for _, insight := range obj.Insights {
			insightIDsList = append(insightIDsList, insight.ID)
		}
		obj.Insights = nil
		err = dbc.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},                                                              // key column
			DoUpdates: clause.AssignmentColumns([]string{"short_title", "long_title", "description", "logo_url"}), // column needed to be updated
		}).Create(&obj).Error
		if err != nil {
			return fmt.Errorf("failure in insert: %v", err)
		}

		for _, insightID := range insightIDsList {
			err = dbc.Clauses(clause.OnConflict{DoNothing: true}).Create(&db.InsightGroupInsight{
				InsightGroupID: obj.ID,
				InsightID:      insightID,
			}).Error
			if err != nil {
				return fmt.Errorf("failure in insert: %v", err)
			}
		}
	}

	return nil
}
