package repository

import (
	"github.com/msojocs/free2api/server/internal/model"
)

// UserRepository defines operations on the User entity.
type UserRepository interface {
	Create(user *model.User) error
	FindByUsername(username string) (*model.User, error)
	FindByID(id uint) (*model.User, error)
	Count() (int64, error)
}

// TaskRepository defines operations on the TaskBatch entity.
type TaskRepository interface {
	Create(task *model.TaskBatch) error
	FindByID(id uint) (*model.TaskBatch, error)
	List(offset, limit int) ([]model.TaskBatch, int64, error)
	Update(task *model.TaskBatch) error
	UpdateFields(id uint, fields map[string]interface{}) error
	Delete(id uint) error
	Count() (int64, error)
}

// AccountRepository defines operations on the Account entity.
type AccountRepository interface {
	Create(account *model.Account) error
	FindByID(id uint) (*model.Account, error)
	List(offset, limit int, accountType string) ([]model.Account, int64, error)
	ListAll(accountType string) ([]model.Account, error)
	Delete(id uint) error
	CountByStatus(status string) (int64, error)
	Count() (int64, error)
}

// ProxyRepository defines operations on the Proxy entity.
type ProxyRepository interface {
	Create(proxy *model.Proxy) error
	Update(proxy *model.Proxy) error
	FindByID(id uint) (*model.Proxy, error)
	List(offset, limit int) ([]model.Proxy, int64, error)
	ListActive() ([]model.Proxy, error)
	Delete(id uint) error
	CountByStatus(status string) (int64, error)
	CountByGroupID(id uint) (int64, error)
}

type ProxyGroupRepository interface {
	Create(group *model.ProxyGroup) error
	FindByID(id uint) (*model.ProxyGroup, error)
	FindByName(name string) (*model.ProxyGroup, error)
	List() ([]model.ProxyGroup, error)
	Update(group *model.ProxyGroup) error
	Delete(id uint) error
}

// CaptchaRepository defines operations on the CaptchaLog entity.
type CaptchaRepository interface {
	Create(log *model.CaptchaLog) error
	ListByTask(taskBatchID uint) ([]model.CaptchaLog, error)
	SumCostByProvider() (map[string]float64, error)
	Count() (int64, error)
}

// PushTemplateRepository defines operations on the PushTemplate entity.
type PushTemplateRepository interface {
	Create(t *model.PushTemplate) error
	FindByID(id uint) (*model.PushTemplate, error)
	List(offset, limit int) ([]model.PushTemplate, int64, error)
	ListEnabled() ([]model.PushTemplate, error)
	// ListEnabledByType returns enabled templates matching accountType or with an empty AccountType (all types).
	ListEnabledByType(accountType string) ([]model.PushTemplate, error)
	Update(t *model.PushTemplate) error
	Delete(id uint) error
}

// TempMailProviderRepository defines operations on the TempMailProvider entity.
type TempMailProviderRepository interface {
	Create(p *model.TempMailProvider) error
	FindByID(id uint) (*model.TempMailProvider, error)
	List() ([]model.TempMailProvider, error)
	ListEnabled() ([]model.TempMailProvider, error)
	Update(p *model.TempMailProvider) error
	Delete(id uint) error
}
