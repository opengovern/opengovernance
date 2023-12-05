package compliance

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/migrator/internal"
	"go.uber.org/zap"

	"github.com/kaytu-io/kaytu-engine/pkg/compliance/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func PopulateDatabase(logger *zap.Logger, dbc *gorm.DB) error {
	p := GitParser{}
	if err := p.ExtractQueries(internal.QueriesGitPath); err != nil {
		logger.Error("failed to extract queries", zap.Error(err))
		return err
	}
	logger.Info("extracted queries", zap.Int("count", len(p.queries)))
	if err := p.ExtractCompliance(internal.ComplianceGitPath); err != nil {
		logger.Error("failed to extract controls and benchmarks", zap.Error(err))
		return err
	}
	logger.Info("extracted controls and benchmarks", zap.Int("controls", len(p.controls)), zap.Int("benchmarks", len(p.benchmarks)))

	loadedQueries := make(map[string]bool)
	err := dbc.Transaction(func(tx *gorm.DB) error {
		for _, obj := range p.queries {
			obj.Controls = nil
			err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "id"}}, // key column
				DoUpdates: clause.AssignmentColumns([]string{
					"query_to_execute",
					"connector",
					"list_of_tables",
					"engine",
					"updated_at",
					"primary_table",
				}), // column needed to be updated
			}).Create(&obj).Error
			if err != nil {
				return err
			}
			loadedQueries[obj.ID] = true
		}
		return nil
	})
	if err != nil {
		logger.Error("failed to insert queries", zap.Error(err))
		return err
	}

	missingQueries := make(map[string]bool)
	err = dbc.Transaction(func(tx *gorm.DB) error {
		tx.Model(&db.BenchmarkChild{}).Where("1=1").Unscoped().Delete(&db.BenchmarkChild{})
		tx.Model(&db.BenchmarkControls{}).Where("1=1").Unscoped().Delete(&db.BenchmarkControls{})
		tx.Model(&db.Benchmark{}).Where("1=1").Unscoped().Delete(&db.Benchmark{})
		tx.Model(&db.Control{}).Where("1=1").Unscoped().Delete(&db.Control{})

		for _, obj := range p.controls {
			obj.Benchmarks = nil
			if obj.QueryID != nil && !loadedQueries[*obj.QueryID] {
				missingQueries[*obj.QueryID] = true
				logger.Info("query not found", zap.String("query_id", *obj.QueryID))
				continue
			}
			err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}},                                                                                                          // key column
				DoUpdates: clause.AssignmentColumns([]string{"title", "description", "document_uri", "severity", "manual_verification", "managed", "updated_at"}), // column needed to be updated
			}).Create(&obj).Error
			if err != nil {
				return err
			}
			for _, tag := range obj.Tags {
				err = tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "key"}, {Name: "control_id"}}, // key columns
					DoUpdates: clause.AssignmentColumns([]string{"key", "value"}),   // column needed to be updated
				}).Create(&tag).Error
				if err != nil {
					return fmt.Errorf("failure in control tag insert: %v", err)
				}
			}
		}

		for _, obj := range p.benchmarks {
			obj.Children = nil
			obj.Controls = nil
			err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}},                                                                                                                                     // key column
				DoUpdates: clause.AssignmentColumns([]string{"title", "description", "logo_uri", "category", "document_uri", "enabled", "managed", "auto_assign", "baseline", "updated_at"}), // column needed to be updated
			}).Create(&obj).Error
			if err != nil {
				return err
			}
			for _, tag := range obj.Tags {
				err = tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "key"}, {Name: "benchmark_id"}}, // key columns
					DoUpdates: clause.AssignmentColumns([]string{"key", "value"}),     // column needed to be updated
				}).Create(&tag).Error
				if err != nil {
					return fmt.Errorf("failure in benchmark tag insert: %v", err)
				}
			}
		}

		for _, obj := range p.benchmarks {
			for _, child := range obj.Children {
				err := tx.Clauses(clause.OnConflict{
					DoNothing: true,
				}).Create(&db.BenchmarkChild{
					BenchmarkID: obj.ID,
					ChildID:     child.ID,
				}).Error
				if err != nil {
					logger.Error("inserted controls and benchmarks", zap.Error(err))
					return err
				}
			}

			for _, control := range obj.Controls {
				if control.QueryID != nil && !loadedQueries[*control.QueryID] {
					continue
				}
				err := tx.Clauses(clause.OnConflict{
					DoNothing: true,
				}).Create(&db.BenchmarkControls{
					BenchmarkID: obj.ID,
					ControlID:   control.ID,
				}).Error
				if err != nil {
					logger.Info("inserted controls and benchmarks", zap.Error(err))
					return err
				}
			}
		}

		missingQueriesList := make([]string, 0, len(missingQueries))
		for query := range missingQueries {
			missingQueriesList = append(missingQueriesList, query)
		}
		if len(missingQueriesList) > 0 {
			logger.Warn("missing queries", zap.Strings("queries", missingQueriesList))
		}
		return nil
	})

	if err != nil {
		logger.Info("inserted controls and benchmarks", zap.Error(err))
		return err
	}

	return nil
}
