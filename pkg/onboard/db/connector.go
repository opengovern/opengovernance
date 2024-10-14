package db

import (
	"github.com/opengovern/og-util/pkg/source"
	"github.com/opengovern/opengovernance/services/integration/model"
)

// ListConnectors gets list of all connectors
func (db Database) ListConnectors() ([]model.Connector, error) {
	var s []model.Connector
	tx := db.Orm.Find(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

// GetConnector gets connector by name
func (db Database) GetConnector(name source.Type) (model.Connector, error) {
	var c model.Connector
	tx := db.Orm.First(&c, "name = ?", name)

	if tx.Error != nil {
		return model.Connector{}, tx.Error
	}

	return c, nil
}

// ListConnectorsTierFiltered gets list of all connectors
func (db Database) ListConnectorsTierFiltered(tier string) ([]model.Connector, error) {
	var s []model.Connector
	tx := db.Orm.Model(&model.Connector{})
	if tier != "" {
		tx = tx.Where("tier = ?", tier)
	}
	tx = tx.Find(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}
