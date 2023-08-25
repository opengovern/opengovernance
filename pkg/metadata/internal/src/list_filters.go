package src

import (
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/internal/database"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/models"
)

func SetListFilter(db database.DatabaseFilter, name string, keyValue map[string]string) error {
	err := db.SetListFilters(models.Filters{Name: name, KeyValue: keyValue})
	if err != nil {
		return err
	}
	return nil
}

func GetListFilters(db database.DatabaseFilter, name string) (models.Filters, error) {
	keyValue, err := db.GetListFilters(name)
	if err != nil {
		return models.Filters{}, err
	}
	return keyValue, nil
}
