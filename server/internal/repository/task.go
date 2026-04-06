package repository

import (
	"errors"

	"github.com/msojocs/free2api/server/internal/model"
	"gorm.io/gorm"
)

type taskRepository struct {
	db *gorm.DB
}

// NewTaskRepository returns a new TaskRepository backed by the given DB.
func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{db: db}
}

func (r *taskRepository) Create(task *model.TaskBatch) error {
	return r.db.Create(task).Error
}

func (r *taskRepository) FindByID(id uint) (*model.TaskBatch, error) {
	var task model.TaskBatch
	err := r.db.First(&task, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &task, err
}

func (r *taskRepository) List(offset, limit int) ([]model.TaskBatch, int64, error) {
	var tasks []model.TaskBatch
	var total int64
	r.db.Model(&model.TaskBatch{}).Count(&total)
	err := r.db.Order("created_at desc").Offset(offset).Limit(limit).Find(&tasks).Error
	return tasks, total, err
}

func (r *taskRepository) Update(task *model.TaskBatch) error {
	return r.db.Save(task).Error
}

func (r *taskRepository) UpdateFields(id uint, fields map[string]interface{}) error {
	return r.db.Model(&model.TaskBatch{}).Where("id = ?", id).Updates(fields).Error
}

func (r *taskRepository) Delete(id uint) error {
	return r.db.Delete(&model.TaskBatch{}, id).Error
}

func (r *taskRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&model.TaskBatch{}).Count(&count).Error
	return count, err
}
