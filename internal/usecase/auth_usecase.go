package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/pkg/hash"
	j "github.com/bubblevy/bookstore/pkg/jwt"
	"github.com/golang-jwt/jwt/v5"

	"golang.org/x/crypto/bcrypt"
)

type AuthUsecase interface {
	Login(ctx context.Context, email, password string) (*entity.AuthResponse, error)
	Register(ctx context.Context, name, email, password string) (*entity.AuthResponse, error)
	GetProfile(ctx context.Context, id uint) (*entity.UserResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*entity.AuthResponse, error)
}

type authUsecase struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	jwtService  j.JWTService
}

func NewAuthUsecase(userRepo repository.UserRepository, sessionRepo repository.SessionRepository, jwtService j.JWTService) AuthUsecase {
	return &authUsecase{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		jwtService:  jwtService,
	}
}

func (u *authUsecase) Login(ctx context.Context, email, password string) (*entity.AuthResponse, error) {
	user, err := u.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("Invalid credentials")
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, fmt.Errorf("Invalid credentials")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("Account is inactive")
	}

	accessToken, err := u.jwtService.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, err
	}

	refreshToken, err := u.jwtService.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	session := &entity.UserSession{
		UserID:           user.ID,
		RefreshTokenHash: hash.SHA256(refreshToken),
		ExpiresAt:        time.Now().Add(30 * 24 * time.Hour),
	}

	if err := u.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("Failed to create session: %w", err)
	}

	return &entity.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (u *authUsecase) Register(ctx context.Context, name, email, password string) (*entity.AuthResponse, error) {
	_, err := u.userRepo.FindByEmail(ctx, email)
	if err == nil {
		return nil, fmt.Errorf("Email already registered")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("Failed to hash password")
	}

	user := &entity.User{
		Name:     name,
		Email:    email,
		Role:     entity.RoleUser,
		Password: string(hashedPassword),
	}

	newUser, err := u.userRepo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("Failed to create user: %w", err)
	}

	accessToken, err := u.jwtService.GenerateToken(newUser.ID, newUser.Email, string(newUser.Role))
	if err != nil {
		return nil, err
	}

	refreshToken, err := u.jwtService.GenerateRefreshToken(newUser.ID)
	if err != nil {
		return nil, err
	}

	session := &entity.UserSession{
		UserID:           newUser.ID,
		RefreshTokenHash: hash.SHA256(refreshToken),
		ExpiresAt:        time.Now().Add(30 * 24 * time.Hour),
	}

	if err := u.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("Failed to create session: %w", err)
	}

	return &entity.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (u *authUsecase) RefreshToken(ctx context.Context, refreshToken string) (*entity.AuthResponse, error) {
	token, err := u.jwtService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("Invalid refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("Invalid refresh token claims")
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("Invalid user id")
	}

	userID := uint(userIDFloat)

	session, err := u.sessionRepo.FindActiveByTokenHash(ctx, hash.SHA256(refreshToken))
	if err != nil {
		return nil, fmt.Errorf("Invalid or expired refresh token")
	}

	if session.UserID != userID {
		return nil, fmt.Errorf("Invalid or expired refresh token")
	}

	user, err := u.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("User not found")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("Account is inactive")
	}

	// revoke the old refresh token
	if err := u.sessionRepo.Revoke(ctx, session.ID); err != nil {
		return nil, fmt.Errorf("Failed to revoke session: %w", err)
	}

	accessToken, err := u.jwtService.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, err
	}

	newRefreshToken, err := u.jwtService.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	// payload session with new refresh token
	newSession := &entity.UserSession{
		UserID:           user.ID,
		RefreshTokenHash: hash.SHA256(newRefreshToken),
		ExpiresAt:        time.Now().Add(30 * 24 * time.Hour),
	}

	// insert new session into database
	if err := u.sessionRepo.Create(ctx, newSession); err != nil {
		return nil, fmt.Errorf("Failed to create new session: %w", err)
	}

	return &entity.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (u *authUsecase) GetProfile(ctx context.Context, id uint) (*entity.UserResponse, error) {
	user, err := u.userRepo.FindByID(ctx, id)
	if err != nil || user == nil {
		return nil, fmt.Errorf("User not found")
	}

	return &entity.UserResponse{
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}
