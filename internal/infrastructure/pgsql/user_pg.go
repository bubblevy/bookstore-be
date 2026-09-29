package pgsql

import (
	"context"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) repository.UserRepository {
	return &userRepository{
		db: db,
	}
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	db := extractDB(ctx, r.db)
	err := db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
	db := extractDB(ctx, r.db)
	err := db.Create(user).Error
	if err != nil {
		return err
	}
	return nil
}

func (r *userRepository) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	var user entity.User
	db := extractDB(ctx, r.db)
	err := db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByActivationToken(ctx context.Context, token string) (*entity.User, error) {
	var user entity.User
	db := extractDB(ctx, r.db)
	err := db.Where("activation_token = ?", token).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	db := extractDB(ctx, r.db)
	return db.Save(user).Error
}

func (r *userRepository) FindByResetPasswordToken(ctx context.Context, token string) (*entity.User, error) {
	var user entity.User
	db := extractDB(ctx, r.db)
	err := db.Where("reset_password_token = ?", token).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
