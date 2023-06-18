package insight

import (
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func PopulateDatabase(dbc *gorm.DB, insightsPath string) error {
	p := GitParser{}
	if err := p.ExtractInsights(insightsPath); err != nil {
		return err
	}

	for _, obj := range p.insights {
		err := dbc.Clauses(clause.OnConflict{
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

	return nil
}
