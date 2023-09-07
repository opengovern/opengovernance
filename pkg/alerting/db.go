package alerting

import (
	"gorm.io/gorm"
)

type Database struct {
	orm *gorm.DB
}

func NewDatabase(orm *gorm.DB) Database {
	return Database{orm: orm}
}

func (db Database) Initialize() error {
	err := db.orm.AutoMigrate(
		&Action{},
		&Rule{},
	)
	if err != nil {
		return err
	}

	return nil
}

func (db Database) ListRules() ([]Rule, error) {
	var listRules []Rule
	err := db.orm.Model(&Rule{}).First(&listRules).Error
	if err != nil {
		return nil, err
	}

	return listRules, nil
}

func (db Database) CreateRule(rule Rule) error {
	return db.orm.Model(&Rule{}).Create(rule).Error
}

func (db Database) DeleteRule(ruleId *uint) error {
	return db.orm.Model(&Action{}).Where("id = ?", ruleId).Delete(Rule{}).Error
}

func (db Database) UpdateRule(ruleId *uint, operator *string, value *int64) error {
	return db.orm.Model(&Action{}).Where("id = ?", ruleId).Updates(map[string]interface{}{"operator": &operator, "value": &value}).Error
}

func (db Database) ListAction() ([]Action, error) {
	var actions []Action
	err := db.orm.Model(&Action{}).First(&actions).Error
	if err != nil {
		return nil, err
	}
	return actions, nil
}

func (db Database) CreateAction(action Action) error {
	return db.orm.Model(&Action{}).Create(&action).Error
}

func (db Database) DeleteAction(actionId uint) error {
	return db.orm.Model(&Action{}).Where("id = ?", actionId).Delete(&Action{}).Error
}

func (db Database) UpdateAction(actionId *uint, method *string, url *string, body *string) error {
	return db.orm.Model(&Action{}).Where("id = ?", actionId).Updates(map[string]interface{}{"method": method, "url": url, "body": body}).Error
}
