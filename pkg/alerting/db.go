package alerting

import (
	"github.com/kaytu-io/kaytu-engine/pkg/alerting/api"
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

func (db Database) GetRule(id uint) (Rule, error) {
	var rule Rule
	err := db.orm.Model(&Rule{}).Where("id = ? ", id).Find(&rule).Error
	if err != nil {
		return Rule{}, err
	}
	return rule, nil
}

func (db Database) CreateRule(id uint, eventType []byte, scope []byte, operator api.Operator, value int64, actionID uint) error {
	rule := Rule{
		ID:        id,
		EventType: eventType,
		Scope:     scope,
		Operator:  operator,
		Value:     value,
		ActionID:  actionID,
	}
	return db.orm.Model(&Rule{}).Create(&rule).Error
}

func (db Database) DeleteRule(ruleId uint) error {
	return db.orm.Model(&Rule{}).Where("id = ?", ruleId).Delete(&Rule{}).Error
}

func (db Database) UpdateRule(id uint, eventType *[]byte, scope *[]byte, operator *api.Operator, value *int64, actionID *uint) error {
	inputs := make(map[string]interface{})

	if eventType != nil {
		inputs["event_type"] = *eventType
	}
	if scope != nil {
		inputs["scope"] = *scope
	}
	if operator != nil {
		inputs["operator"] = *operator
	}
	if value != nil {
		inputs["value"] = *value
	}
	if actionID != nil {
		inputs["action_id"] = *actionID
	}

	return db.orm.Model(&Rule{}).Where("id = ?", id).Updates(inputs).Error
}

func (db Database) ListAction() ([]Action, error) {
	var actions []Action
	err := db.orm.Model(&Action{}).Find(&actions).Error
	if err != nil {
		return nil, err
	}

	return actions, nil
}

func (db Database) GetAction(id uint) (Action, error) {
	var action Action
	err := db.orm.Model(&Action{}).Where("id = ?", id).Find(&action).Error
	if err != nil {
		return Action{}, err
	}
	return action, nil
}

func (db Database) CreateAction(id uint, method string, url string, headers []byte, body string) error {
	action := Action{
		ID:      id,
		Method:  method,
		Url:     url,
		Headers: headers,
		Body:    body,
	}
	return db.orm.Model(&Action{}).Create(&action).Error
}

func (db Database) DeleteAction(actionId uint) error {
	return db.orm.Model(&Action{}).Where("id = ?", actionId).Delete(&Action{}).Error
}

func (db Database) UpdateAction(id uint, headers *[]byte, url *string, body *string, method *string) error {
	inputs := make(map[string]interface{})

	if headers != nil {
		inputs["headers"] = *headers
	}
	if body != nil {
		inputs["body"] = *body
	}
	if url != nil {
		inputs["url"] = *url
	}
	if method != nil {
		inputs["method"] = *method
	}

	return db.orm.Model(&Action{}).Where("id = ?", id).Updates(inputs).Error
}
