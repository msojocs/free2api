package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusPaused    TaskStatus = "paused"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

type JSONMap map[string]interface{}

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return "{}", nil
	}
	b, err := json.Marshal(j)
	return string(b), err
}

func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = JSONMap{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return errors.New("unsupported type for JSONMap")
	}
	return json.Unmarshal(bytes, j)
}

type TaskBatch struct {
	BaseModel
	Name      string     `gorm:"not null" json:"name"`
	Type      string     `gorm:"not null" json:"type"`
	Status    TaskStatus `gorm:"default:'pending'" json:"status"`
	Total     int        `gorm:"default:0" json:"total"`
	Completed int        `gorm:"default:0" json:"completed"`
	Failed    int        `gorm:"default:0" json:"failed"`
	Config    JSONMap    `gorm:"type:text" json:"config"`
	Logs      string     `gorm:"type:text" json:"logs"`
}
