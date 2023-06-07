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
			Columns: []clause.Column{{Name: "id"}}, // key colume
			DoUpdates: clause.AssignmentColumns([]string{"query_id", "connector", "short_title", "long_title",
				"description", "logo_url", "tags", "links", "enabled", "internal"}), // column needed to be updated
		}).Create(&obj).Error
		if err != nil {
			return fmt.Errorf("failure in insert: %v", err)
		}
	}

	return nil
}
