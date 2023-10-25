package db

import (
	"errors"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db/model"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (db Database) AddStack(record *model.Stack) error {
	return db.ORM.Model(&model.Stack{}).
		Create(record).Error
}

func (db Database) GetStack(stackID string) (model.Stack, error) {
	var s model.Stack
	tx := db.ORM.Model(&model.Stack{}).
		Where("stack_id = ?", stackID).
		Preload("Tags").
		Preload("Evaluations").
		First(&s)

	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return model.Stack{}, nil
		}
		return model.Stack{}, tx.Error
	}

	return s, nil
}

func (db Database) ListStacks(tags map[string][]string, accountIds []string) ([]model.Stack, error) {
	var s []model.Stack
	query := db.ORM.Model(&model.Stack{}).
		Preload("Tags")
	if len(accountIds) != 0 {
		query = query.Where("EXISTS (SELECT 1 FROM unnest(account_ids) AS account WHERE account IN ?)", accountIds)
	}
	if len(tags) != 0 {
		query = query.Joins("JOIN stack_tags AS tags ON tags.stack_id = stacks.stack_id")
		for key, values := range tags {
			if len(values) != 0 {
				query = query.Where("tags.key = ? AND (tags.value && ?)", key, pq.StringArray(values))
			} else {
				query = query.Where("tags.key = ?", key)
			}
		}
	}

	tx := query.Find(&s)

	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}

	return s, nil
}

func (db Database) DeleteStack(stackID string) error {
	return db.ORM.Model(&model.Stack{}).
		Where("stack_id = ?", stackID).
		Delete(&model.Stack{}).Error
}

func (db Database) UpdateStackResources(stackID string, resources pq.StringArray) error {
	tx := db.ORM.Model(&model.Stack{}).
		Where("stack_id = ?", stackID).
		Update("resources", resources)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) AddEvaluation(record *model.StackEvaluation) error {
	return db.ORM.Model(&model.StackEvaluation{}).
		Create(record).Error
}

func (db Database) GetEvaluation(jobId uint) (model.StackEvaluation, error) {
	var result model.StackEvaluation
	tx := db.ORM.Model(&model.StackEvaluation{}).
		Where("job_id = ?", jobId).
		First(&result)
	if tx.Error != nil {
		return model.StackEvaluation{}, tx.Error
	}

	return result, nil
}

func (db Database) GetResourceStacks(resourceID string) ([]model.Stack, error) {
	var stacks []model.Stack
	tx := db.ORM.Model(&model.Stack{}).
		Where("resources @> ?", pq.StringArray{resourceID}).
		Preload("Tags").
		Find(&stacks)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return stacks, nil
}

func (db Database) CreateStackCredential(a *model.StackCredential) error {
	tx := db.ORM.
		Model(&model.StackCredential{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "stack_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"secret"}),
		}).
		Create(a)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) GetStackCredential(stackId string) (model.StackCredential, error) {
	var stackCredential model.StackCredential
	tx := db.ORM.Model(&model.StackCredential{}).
		Where("stack_id = ?", stackId).
		Find(&stackCredential)
	if tx.Error != nil {
		return model.StackCredential{}, tx.Error
	}
	return stackCredential, nil
}

func (db Database) RemoveStackCredential(stackId string) error {
	tx := db.ORM.Model(&model.StackCredential{}).
		Where("stack_id = ?", stackId).
		Delete(&model.StackCredential{})
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) ListPendingStacks() ([]model.Stack, error) {
	var stacks []model.Stack
	tx := db.ORM.Model(&model.Stack{}).
		Where("status IN (?, ?)", api.StackStatusPending, api.StackStatusStalled).
		Find(&stacks)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return stacks, nil
}

func (db Database) ListCreatedStacks() ([]model.Stack, error) {
	var stacks []model.Stack
	tx := db.ORM.Model(&model.Stack{}).
		Where("status = ?", api.StackStatusCreated).
		Find(&stacks)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return stacks, nil
}

func (db Database) UpdateStackStatus(stackId string, status api.StackStatus) error {
	tx := db.ORM.Model(&model.Stack{}).
		Where("stack_id = ?", stackId).
		Update("status", status)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) UpdateStackFailureMessage(stackId string, message string) error {
	tx := db.ORM.Model(&model.Stack{}).
		Where("stack_id = ?", stackId).
		Update("failure_message", message)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) ListDescribingStacks() ([]model.Stack, error) {
	var stacks []model.Stack
	tx := db.ORM.Model(&model.Stack{}).
		Where("status = ?", api.StackStatusDescribing).
		Find(&stacks)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return stacks, nil
}

func (db Database) ListDescribedStacks() ([]model.Stack, error) {
	var stacks []model.Stack
	tx := db.ORM.Model(&model.Stack{}).
		Where("status = ?", api.StackStatusDescribed).
		Find(&stacks)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return stacks, nil
}

func (db Database) ListEvaluatingStacks() ([]model.Stack, error) {
	var stacks []model.Stack
	tx := db.ORM.Model(&model.Stack{}).
		Where("status = ?", api.StackStatusEvaluating).
		Preload("Evaluations").
		Find(&stacks)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return stacks, nil
}

func (db Database) UpdateEvaluationStatus(jobId uint, status api.StackEvaluationStatus) error {
	tx := db.ORM.Model(&model.StackEvaluation{}).
		Where("job_id = ?", jobId).
		Update("status", status)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (db Database) ListFailedStacks() ([]model.Stack, error) {
	var stacks []model.Stack
	tx := db.ORM.Model(&model.Stack{}).
		Where("status = ?", api.StackStatusFailed).
		Preload("Evaluations").
		Find(&stacks)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return stacks, nil
}
