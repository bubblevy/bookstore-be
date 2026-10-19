package pgsql

import (
	"context"
	"strings"

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

func (r *userRepository) FindAll(ctx context.Context, filter repository.UserFilter) ([]entity.User, int64, error) {
	var users []entity.User
	var total int64
	db := extractDB(ctx, r.db).Model(&entity.User{})

	if filter.Search != "" {
		s := "%" + strings.ToLower(strings.TrimSpace(filter.Search)) + "%"
		db = db.Where("LOWER(name) LIKE ? OR LOWER(email) LIKE ?", s, s)
	}
	if filter.Role != "" {
		db = db.Where("role = ?", filter.Role)
	}
	if filter.IsActive != nil {
		db = db.Where("is_active = ?", *filter.IsActive)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Limit > 0 {
		db = db.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		db = db.Offset(filter.Offset)
	}

	if err := db.Order("id DESC").Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}
