package db

import (
	"github.com/opengovern/opencomply/services/tasks/db/models"
	"gorm.io/gorm"
)

type Database struct {
	Orm *gorm.DB
}

func (db Database) Initialize() error {
	err := db.Orm.AutoMigrate(
		&models.Task{},
		&models.TaskRun{},
	)
	if err != nil {
		return err
	}

	return nil
}

func (db Database) CreateTask(task *models.Task) error {
	tx := db.Orm.FirstOrCreate(task)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (db Database) UpdateTask(id string, task *models.Task) error {
	tx := db.Orm.
		Model(&models.Task{}).
		Where("id = ?", id).
		Updates(task)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

// GetTask retrieves a task by Task name
func (db Database) GetTask(id string) (*models.Task, error) {
	var task models.Task
	tx := db.Orm.Where("id = ?", id).
		First(&task)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return &task, nil
}

// GetTaskRunResult retrieves a task result by Task ID
func (db Database) GetTaskRunResult(id string) (*models.TaskRun, error) {
	var task *models.TaskRun
	tx := db.Orm.Where("id = ?", id).
		Order("created_at desc").
		First(&task)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return task, nil
}

// FetchCreatedTaskRuns retrieves a list of task runs
func (db Database) FetchCreatedTaskRuns() ([]models.TaskRun, error) {
	var tasks []models.TaskRun
	tx := db.Orm.Model(&models.TaskRun{}).Where("status = ?", models.TaskRunStatusCreated).Find(&tasks)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return tasks, nil
}

func (db Database) CreateTaskRun(taskRun *models.TaskRun) error {
	tx := db.Orm.Create(taskRun)
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// UpdateTaskRun creates a task result
func (db Database) UpdateTaskRun(runID uint, status models.TaskRunStatus, result string) error {
	tx := db.Orm.Where("id = ?", runID).Updates(&models.TaskRun{
		Status: status, Result: result,
	})
	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

// GetTaskList retrieves a list of tasks
func (db Database) GetTaskList() ([]models.Task, error) {
	var tasks []models.Task
	tx := db.Orm.Order("created_at desc").Find(&tasks)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return tasks, nil
}
