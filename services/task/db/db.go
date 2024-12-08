package db

import (
	
	"gorm.io/gorm"
)

type Database struct {
	Orm *gorm.DB
}

func (db Database) Initialize() error {
	err := db.Orm.AutoMigrate(
		&Task{},
		&TaskResult{},
		
	)
	if err != nil {
		return err
	}

	return nil
}


func (db Database) CreateTask(task *Task) error {
	tx:=db.Orm.Create(task)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// GetTaskResult retrieves a task result by Task ID
func (db Database) GetTaskResult(id string) ([]TaskResult, error) {
	var task []TaskResult
	tx := db.Orm.Where("task_id = ?", id).
	Order("created_at desc").
	Find(&task)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return task, nil
}

// GetTaskList retrieves a list of tasks
func (db Database) GetTaskList() ([]Task, error) {
	var tasks []Task
	tx := db.Orm.Order("created_at desc").Find(&tasks)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return tasks, nil
}

// CreateTaskResult creates a task result
func (db Database) CreateTaskResult(taskResult *TaskResult) error {
	tx := db.Orm.Create(taskResult)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}



