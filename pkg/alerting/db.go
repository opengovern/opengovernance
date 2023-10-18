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
	)
	if err != nil {
		return err
	}

	return nil
}

func (db Database) CreateComplianceTrigger(hour time.Time, benchmarkId string, connectionId string, value int64, responseStatus int) error {
	compliance := TriggerCompliance{
		ComplianceId:   benchmarkId,
		Hour:           hour,
		ConnectionId:   connectionId,
		Value:          value,
		ResponseStatus: responseStatus,
	}

	return db.orm.Model(&TriggerCompliance{}).Create(&compliance).Error
}

func (db Database) CreateInsightTrigger(hour time.Time, insightId int64, connectionId string, value int64, responseStatus int) error {
	insight := TriggerInsight{
		InsightId:      insightId,
		Hour:           hour,
		ConnectionId:   connectionId,
		Value:          value,
		ResponseStatus: responseStatus,
	}
	return db.orm.Model(&TriggerInsight{}).Create(&insight).Error
}

func (db Database) ListInsightTriggers() ([]TriggerInsight, error) {
	var listInsightTriggers []TriggerInsight
	err := db.orm.Model(&TriggerInsight{}).Find(&listInsightTriggers).Error
	if err != nil {
		return nil, err
	}
	return listInsightTriggers, nil
}

func (db Database) ListComplianceTriggers() ([]TriggerCompliance, error) {
	var listComplianceTriggers []TriggerCompliance
	err := db.orm.Model(&TriggerCompliance{}).Find(&listComplianceTriggers).Error
	if err != nil {
		return nil, err
	}
	return listComplianceTriggers, nil
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

func (db Database) CreateRule(eventType []byte, scope []byte, operator []byte, actionID uint) error {
	rule := Rule{
		EventType: eventType,
		Scope:     scope,
		Operator:  operator,
		ActionID:  actionID,
	}
	return db.orm.Model(&Rule{}).Create(&rule).Error
}

func (db Database) DeleteRule(ruleId uint) error {
	return db.orm.Model(&Rule{}).Where("id = ?", ruleId).Delete(&Rule{}).Error
}

func (db Database) UpdateRule(id uint, eventType *[]byte, scope *[]byte, operator *[]byte, actionID *uint) error {
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
