package usecase

import (
	"context"
	"fmt"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/pkg/jwt"

	"golang.org/x/crypto/bcrypt"
)

type AuthUsecase interface {
	Login(ctx context.Context, email, password string) (string, error)
	Register(ctx context.Context, name, email, password string) (string, error)
}

type authUsecase struct {
	repo       repository.UserRepository
	jwtService jwt.JWTService
}

func NewAuthUsecase(repo repository.UserRepository, jwtService jwt.JWTService) AuthUsecase {
	return &authUsecase{
		repo:       repo,
		jwtService: jwtService,
	}
}

func (u *authUsecase) Login(ctx context.Context, email, password string) (string, error) {
	user, err := u.repo.FindByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", fmt.Errorf("invalid credentials")
	}

	return u.jwtService.GenerateToken(user.ID, user.Email)
}

func (u *authUsecase) Register(ctx context.Context, name, email, password string) (string, error) {
	existingUser, _ := u.repo.FindByEmail(ctx, email)
	if existingUser != nil {
		return "", fmt.Errorf("email already registered")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password")
	}

	newUser := &entity.User{
		Name:     name,
		Email:    email,
		Password: string(hashedPassword),
	}

	createdUser, err := u.repo.Create(ctx, newUser)
	if err != nil {
		return "", fmt.Errorf("failed to create user: %v", err)
	}

	return u.jwtService.GenerateToken(createdUser.ID, createdUser.Email)
}
