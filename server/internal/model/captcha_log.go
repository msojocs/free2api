package model

type CaptchaLog struct {
	BaseModel
	TaskBatchID uint    `json:"task_batch_id"`
	Email       string  `json:"email"`
	Type        string  `json:"type"`
	Status      string  `json:"status"`
	Provider    string  `json:"provider"`
	Cost        float64 `json:"cost"`
}
