package compliance

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func PopulateDatabase(dbc *gorm.DB) error {
	var benchmarks []db.Benchmark
	var benchmarkTags []db.BenchmarkTag
	var policies []db.Policy
	var policyTags []db.PolicyTag
	var queries []db.Query

	for _, obj := range benchmarkTags {
		dbc.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},                                    // key colume
			DoUpdates: clause.AssignmentColumns([]string{"key", "value", "updated_at"}), // column needed to be updated
		}).Create(&obj)
	}

	for _, obj := range benchmarks {
		dbc.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},                                                                                                                                     // key colume
			DoUpdates: clause.AssignmentColumns([]string{"title", "description", "logo_uri", "category", "document_uri", "enabled", "managed", "auto_assign", "baseline", "updated_at"}), // column needed to be updated
		}).Create(&obj)
	}

	for _, obj := range benchmarks {
		for _, child := range obj.Children {
			dbc.Clauses(clause.OnConflict{
				DoNothing: true,
			}).Create(&db.BenchmarkChild{
				BenchmarkID: obj.ID,
				ChildID:     child.ID,
			})
		}

		for _, tag := range obj.Tags {
			dbc.Clauses(clause.OnConflict{
				DoNothing: true,
			}).Create(&db.BenchmarkTagRel{
				BenchmarkID:    obj.ID,
				BenchmarkTagID: tag.ID,
			})
		}
	}

	for _, obj := range policyTags {
		dbc.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},                                    // key colume
			DoUpdates: clause.AssignmentColumns([]string{"key", "value", "updated_at"}), // column needed to be updated
		}).Create(&obj)
	}

	for _, obj := range queries {
		dbc.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},                                                                                                   // key colume
			DoUpdates: clause.AssignmentColumns([]string{"query_to_execute", "connector", "list_of_tables", "engine", "engine_version", "updated_at"}), // column needed to be updated
		}).Create(&obj)
	}

	for _, obj := range policies {
		dbc.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},                                                                                                          // key colume
			DoUpdates: clause.AssignmentColumns([]string{"title", "description", "document_uri", "severity", "manual_verification", "managed", "updated_at"}), // column needed to be updated
		}).Create(&obj)

		for _, tag := range obj.Tags {
			dbc.Clauses(clause.OnConflict{
				DoNothing: true,
			}).Create(&db.PolicyTagRel{
				PolicyID:    obj.ID,
				PolicyTagID: tag.ID,
			})
		}

		for _, benchmark := range obj.Benchmarks {
			dbc.Clauses(clause.OnConflict{
				DoNothing: true,
			}).Create(&db.BenchmarkPolicies{
				BenchmarkID: benchmark.ID,
				PolicyID:    obj.ID,
			})
		}
	}

	return nil
}
