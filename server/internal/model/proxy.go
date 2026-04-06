package model

type Proxy struct {
	BaseModel
	Host         string      `gorm:"not null" json:"host"`
	Port         string      `gorm:"not null" json:"port"`
	ProxyGroupID *uint       `gorm:"index" json:"proxy_group_id,omitempty"`
	ProxyGroup   *ProxyGroup `json:"proxy_group,omitempty"`
	Username     string      `json:"username"`
	Password     string      `json:"password"`
	Protocol     string      `gorm:"default:'http'" json:"protocol"`
	Status       string      `gorm:"default:'active'" json:"status"`
}
