package compliance

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func PopulateDatabase(dbc *gorm.DB, compliancePath, queryPath string) error {
	p := GitParser{}
	if err := p.ExtractQueries(queryPath); err != nil {
		return err
	}

	if err := p.ExtractCompliance(compliancePath); err != nil {
		return err
	}

	for _, obj := range p.policyTags {
		obj.Policies = nil
		err := dbc.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},                                    // key colume
			DoUpdates: clause.AssignmentColumns([]string{"key", "value", "updated_at"}), // column needed to be updated
		}).Create(&obj).Error
		if err != nil {
			return err
		}
	}

	for _, obj := range p.benchmarkTags {
		obj.Benchmarks = nil
		err := dbc.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},                                    // key colume
			DoUpdates: clause.AssignmentColumns([]string{"key", "value", "updated_at"}), // column needed to be updated
		}).Create(&obj).Error
		if err != nil {
			return err
		}
	}

	for _, obj := range p.queries {
		obj.Policies = nil
		err := dbc.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},                                                                                 // key colume
			DoUpdates: clause.AssignmentColumns([]string{"query_to_execute", "connector", "list_of_tables", "engine", "updated_at"}), // column needed to be updated
		}).Create(&obj).Error
		if err != nil {
			return err
		}
	}

	for _, obj := range p.policies {
		obj.Tags = nil
		obj.Benchmarks = nil
		err := dbc.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},                                                                                                          // key colume
			DoUpdates: clause.AssignmentColumns([]string{"title", "description", "document_uri", "severity", "manual_verification", "managed", "updated_at"}), // column needed to be updated
		}).Create(&obj).Error
		if err != nil {
			return err
		}
	}

	for _, obj := range p.policies {
		for _, tag := range obj.Tags {
			err := dbc.Clauses(clause.OnConflict{
				DoNothing: true,
			}).Create(&db.PolicyTagRel{
				PolicyID:    obj.ID,
				PolicyTagID: tag.ID,
			}).Error
			if err != nil {
				return err
			}
		}
	}

	for _, obj := range p.benchmarks {
		obj.Children = nil
		obj.Tags = nil
		obj.Policies = nil
		err := dbc.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},                                                                                                                                     // key colume
			DoUpdates: clause.AssignmentColumns([]string{"title", "description", "logo_uri", "category", "document_uri", "enabled", "managed", "auto_assign", "baseline", "updated_at"}), // column needed to be updated
		}).Create(&obj).Error
		if err != nil {
			return err
		}
	}

	for _, obj := range p.benchmarks {
		for _, child := range obj.Children {
			err := dbc.Clauses(clause.OnConflict{
				DoNothing: true,
			}).Create(&db.BenchmarkChild{
				BenchmarkID: obj.ID,
				ChildID:     child.ID,
			}).Error
			if err != nil {
				return err
			}
		}

		for _, tag := range obj.Tags {
			err := dbc.Clauses(clause.OnConflict{
				DoNothing: true,
			}).Create(&db.BenchmarkTagRel{
				BenchmarkID:    obj.ID,
				BenchmarkTagID: tag.ID,
			}).Error
			if err != nil {
				return err
			}
		}

		for _, policy := range obj.Policies {
			err := dbc.Clauses(clause.OnConflict{
				DoNothing: true,
			}).Create(&db.BenchmarkPolicies{
				BenchmarkID: obj.ID,
				PolicyID:    policy.ID,
			}).Error
			if err != nil {
				return err
			}
		}
	}

	return nil
}
