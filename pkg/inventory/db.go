package inventory

import (
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Database struct {
	orm *gorm.DB
}

func (db Database) Initialize() error {
	err := db.orm.AutoMigrate(
		&SmartQuery{},
	)
	if err != nil {
		return err
	}

	return nil
}

// AddQuery adding a query
func (db Database) AddQuery(q *SmartQuery) error {
	tx := db.orm.
		Model(&SmartQuery{}).
		Create(q)

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// GetQueries gets list of all queries
func (db Database) GetQueries() ([]SmartQuery, error) {
	var s []SmartQuery
	tx := db.orm.Find(&s)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return s, nil
}

// GetQuery gets a query with matching id
func (db Database) GetQuery(id uuid.UUID) (SmartQuery, error) {
	var s SmartQuery
	tx := db.orm.First(&s, "id = ?", id.String())

	if tx.Error != nil {
		return SmartQuery{}, tx.Error
	} else if tx.RowsAffected != 1 {
		return SmartQuery{}, fmt.Errorf("get query: specified id was not found")
	}

	return s, nil
}
