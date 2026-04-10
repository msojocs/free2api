package model

type Account struct {
	BaseModel
	Email       string `gorm:"index" json:"email"`
	Password    string `json:"password"`
	Type        string `json:"type"`
	Status      string `gorm:"default:'active'" json:"status"`
	TaskBatchID uint   `json:"task_batch_id"`
	Extra       string `gorm:"type:text" json:"extra"`
	Usage       JSONMap `gorm:"type:text" json:"usage"`
}
