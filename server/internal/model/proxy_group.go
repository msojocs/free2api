package model

type ProxyGroup struct {
	BaseModel
	Name string `gorm:"not null;uniqueIndex" json:"name"`
}
