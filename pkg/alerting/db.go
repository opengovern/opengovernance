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
	err := db.orm.Model(&Rule{}).Find(&listRules).Error
	if err != nil {
		return nil, err
	}

	return listRules, nil
}

func (db Database) CreateRule(rule Rule) error {
	return db.orm.Model(&Rule{}).Create(rule).Error
}

func (db Database) DeleteRule(ruleId uint) error {
	return db.orm.Model(&Action{}).Where("ID = ?", ruleId).Delete(Rule{}).Error
}

func (db Database) UpdateRule(rule Rule) error {
	var inputs map[string]interface{}

	if &rule.EventType != nil {
		inputs["eventType"] = rule.EventType
	}
	if &rule.Scope != nil {
		inputs["scope"] = rule.Scope
	}
	if &rule.Operator != nil {
		inputs["operator"] = rule.Operator
	}
	if &rule.Value != nil {
		inputs["value"] = rule.Value
	}
	if &rule.ActionID != nil {
		inputs["actionId"] = rule.ActionID
	}

	return db.orm.Model(&Action{}).Where("ID = ?", rule.ID).Updates(inputs).Error
}

func (db Database) ListAction() ([]Action, error) {
	var actions []Action
	err := db.orm.Model(&Action{}).Find(&actions).Error
	if err != nil {
		return nil, err
	}

	return actions, nil
}

func (db Database) CreateAction(action Action) error {
	return db.orm.Model(&Action{}).Create(&action).Error
}

func (db Database) DeleteAction(actionId uint) error {
	return db.orm.Model(&Action{}).Where("ID = ?", actionId).Delete(&Action{}).Error
}

func (db Database) UpdateAction(action Action) error {
	var inputs map[string]interface{}

	if &action.Headers != nil {
		inputs["header"] = action.Headers
	}
	if &action.Body != nil {
		inputs["body"] = action.Body
	}
	if &action.Url != nil {
		inputs["url"] = action.Url
	}
	if &action.Method != nil {
		inputs["method"] = action.Method
	}

	return db.orm.Model(&Action{}).Where("ID = ?", action.ID).Updates(inputs).Error
}
