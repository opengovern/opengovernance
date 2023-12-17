package repository

import (
	"github.com/kaytu-io/kaytu-engine/services/integration/db"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gorm.io/gorm/clause"
)

type Source struct {
	db *db.Database
}

// ListSources gets list of all source
func (s Source) ListSources() ([]model.Source, error) {
	var sources []model.Source

	tx := s.db.DB.Model(model.Source{}).Preload(clause.Associations).Find(&sources)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return sources, nil
}

// GetSourcesOfType gets list of sources with matching type
func (s Source) GetSourcesOfType(rType source.Type) ([]model.Source, error) {
	var sources []model.Source

	tx := s.db.DB.Preload(clause.Associations).Find(&sources, "type = ?", rType)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return sources, nil
}

// GetSourcesOfTypes gets list of sources with matching types
func (s Source) GetSourcesOfTypes(rTypes []source.Type) ([]model.Source, error) {
	var sources []model.Source
	tx := s.db.DB.Preload(clause.Associations).Find(&sources, "type IN ?", rTypes)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return sources, nil
}
