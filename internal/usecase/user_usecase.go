package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/pkg/pagination"
)

// UserProfileResponse represents a safe, non-sensitive view of a user profile.
type UserProfileResponse struct {
	ID        uint            `json:"id"`
	Name      string          `json:"name"`
	Email     string          `json:"email"`
	Role      entity.UserRole `json:"role"`
	IsActive  bool            `json:"is_active"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// UserListItem represents a user summary in the admin paginated list.
type UserListItem struct {
	ID        uint            `json:"id"`
	Name      string          `json:"name"`
	Email     string          `json:"email"`
	Role      entity.UserRole `json:"role"`
	IsActive  bool            `json:"is_active"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type UserUsecase interface {
	GetProfile(ctx context.Context, userID uint) (*UserProfileResponse, error)
	FindAll(ctx context.Context, filter repository.UserFilter, pq pagination.Query) ([]UserListItem, pagination.Meta, error)
}

type userUsecase struct {
	userRepo repository.UserRepository
}

func NewUserUsecase(userRepo repository.UserRepository) UserUsecase {
	return &userUsecase{
		userRepo: userRepo,
	}
}

func (u *userUsecase) GetProfile(ctx context.Context, userID uint) (*UserProfileResponse, error) {
	user, err := u.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	if user == nil {
		return nil, entity.ErrUserNotFound
	}

	return &UserProfileResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

func (u *userUsecase) FindAll(ctx context.Context, filter repository.UserFilter, pq pagination.Query) ([]UserListItem, pagination.Meta, error) {
	filter.Limit = pq.PerPage
	filter.Offset = pq.Offset()

	users, total, err := u.userRepo.FindAll(ctx, filter)
	if err != nil {
		return nil, pagination.Meta{}, fmt.Errorf("failed to fetch users list: %w", err)
	}

	items := make([]UserListItem, 0, len(users))
	for _, user := range users {
		items = append(items, UserListItem{
			ID:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			Role:      user.Role,
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		})
	}

	meta := pagination.BuildMeta(total, pq.Page, pq.PerPage)
	return items, meta, nil
}
