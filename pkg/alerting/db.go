package alerting

import (
	"gorm.io/gorm"
	"time"
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
		&Triggers{},
	)
	if err != nil {
		return err
	}

	return nil
}

func (db Database) CreateTrigger(Time time.Time, eventType []byte, scope []byte, value int64, responseStatus int) error {
	trigger := Triggers{
		EventType:      eventType,
		Scope:          scope,
		TriggeredAt:    Time,
		Value:          value,
		ResponseStatus: responseStatus,
	}
	return db.orm.Model(&Triggers{}).Create(&trigger).Error
}

func (db Database) ListTriggers() ([]Triggers, error) {
	var listTriggers []Triggers
	err := db.orm.Model(&Triggers{}).Find(&listTriggers).Error
	if err != nil {
		return nil, err
	}
	return listTriggers, nil
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
	err := db.orm.Model(&Rule{}).Where("id = ?", id).Find(&rule).Error
	if err != nil {
		return Rule{}, err
	}
	return rule, nil
}

func (db Database) CreateRule(eventType []byte, scope []byte, operator []byte, actionID uint, metadata []byte) (uint, error) {
	rule := Rule{
		EventType: eventType,
		Scope:     scope,
		Operator:  operator,
		ActionID:  actionID,
		Metadata:  metadata,
	}
	err := db.orm.Model(&Rule{}).Create(&rule).Error
	return rule.Id, err
}

func (db Database) DeleteRule(ruleId uint) error {
	return db.orm.Model(&Rule{}).Where("id = ?", ruleId).Delete(&Rule{}).Error
}

func (db Database) UpdateRule(id uint, eventType *[]byte, scope *[]byte, metadata *[]byte, operator *[]byte, actionID *uint) error {
	inputs := Rule{}

	if eventType != nil {
		inputs.EventType = *eventType
	}
	if scope != nil {
		inputs.Scope = *scope
	}
	if operator != nil {
		inputs.Operator = *operator
	}
	if actionID != nil {
		inputs.ActionID = *actionID
	}
	if metadata != nil {
		inputs.Metadata = *metadata
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

func (db Database) CreateAction(method string, url string, headers []byte, body string) error {
	action := Action{
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
	inputs := Action{}

	if headers != nil {
		inputs.Headers = *headers
	}
	if body != nil {
		inputs.Body = *body
	}
	if url != nil {
		inputs.Url = *url
	}
	if method != nil {
		inputs.Method = *method
	}

	return db.orm.Model(&Action{}).Where("id = ?", id).Updates(inputs).Error
}
