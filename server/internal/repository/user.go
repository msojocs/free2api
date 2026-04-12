package repository

import (
	"errors"

	"github.com/msojocs/ai-auto-register/server/internal/model"
	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository returns a new UserRepository backed by the given DB.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *userRepository) Update(user *model.User) error {
	return r.db.Save(user).Error
}

func (r *userRepository) FindByUsername(username string) (*model.User, error) {
	var user model.User
	result := r.db.Where("username = ?", username).Limit(1).Find(&user)
	err := result.Error
	if err == nil && result.RowsAffected == 0 {
		return nil, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *userRepository) FindByID(id uint) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *userRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&model.User{}).Count(&count).Error
	return count, err
}
