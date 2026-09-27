package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/pkg/email"
	"github.com/bubblevy/bookstore/pkg/hash"
	j "github.com/bubblevy/bookstore/pkg/jwt"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"golang.org/x/crypto/bcrypt"
)

type AuthUsecase interface {
	Login(ctx context.Context, email, password string) (*entity.AuthResponse, error)
	Register(ctx context.Context, name, email, password string) error
	GetProfile(ctx context.Context, id uint) (*entity.UserResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*entity.AuthResponse, error)
	ResendActivation(ctx context.Context, email string) error
	ActivateAccount(ctx context.Context, token string) error
}

type authUsecase struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	jwtService  j.JWTService
	emailSender email.EmailSender
}

func NewAuthUsecase(userRepo repository.UserRepository, sessionRepo repository.SessionRepository, jwtService j.JWTService, emailSender email.EmailSender) AuthUsecase {
	return &authUsecase{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		jwtService:  jwtService,
		emailSender: emailSender,
	}
}

// generate random token for activation or password reset
func generateRandomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
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
		ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
	}

	if err := u.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("Failed to create session: %w", err)
	}

	return &entity.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (u *authUsecase) Register(ctx context.Context, name, email, password string) error {
	_, err := u.userRepo.FindByEmail(ctx, email)
	if err == nil {
		return entity.ErrEmailAlreadyRegistered
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("Failed to check email: %w", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("Failed to hash password")
	}

	// generate activation token
	activationToken, err := generateRandomToken()
	if err != nil {
		return fmt.Errorf("Failed to generate activation token")
	}
	expiresAt := time.Now().Add(24 * time.Hour)

	user := &entity.User{
		Name:                name,
		Email:               email,
		Role:                entity.RoleUser,
		Password:            string(hashedPassword),
		ActivationToken:     &activationToken,
		ActivationExpiresAt: &expiresAt,
	}

	if err := u.userRepo.Create(ctx, user); err != nil {
		return fmt.Errorf("Failed to create user: %w", err)
	}

	// send email activation account
	asyncCtx := context.WithoutCancel(ctx)
	timeoutCtx, cancel := context.WithTimeout(asyncCtx, 10*time.Second)
	go func(bgCtx context.Context) {
		defer cancel()
		if err := u.emailSender.SendActivationEmail(bgCtx, user.Email, user.Name, activationToken); err != nil {
			log.Printf("Failed to send activation email: %v", err)
		}
	}(timeoutCtx)

	return nil
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
		ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
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

func (u *authUsecase) ResendActivation(ctx context.Context, email string) error {
	user, err := u.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("Failed to fetch user: %w", err)
	}

	if user == nil || user.IsActive {
		return nil
	}

	newToken, err := generateRandomToken()
	if err != nil {
		return fmt.Errorf("Failed to generate activation token: %w", err)
	}
	expiresAt := time.Now().Add(24 * time.Hour)

	user.ActivationToken = &newToken
	user.ActivationExpiresAt = &expiresAt

	if err := u.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("Failed to update activation token: %w", err)
	}

	// resend email activation account
	asyncCtx := context.WithoutCancel(ctx)
	timeoutCtx, cancel := context.WithTimeout(asyncCtx, 10*time.Second)
	go func(bgCtx context.Context) {
		defer cancel()
		if err := u.emailSender.SendActivationEmail(bgCtx, user.Email, user.Name, newToken); err != nil {
			log.Printf("Failed to send activation email: %v", err)
		}
	}(timeoutCtx)

	return nil
}

func (u *authUsecase) ActivateAccount(ctx context.Context, token string) error {
	if token == "" {
		return fmt.Errorf("Activation token is required")
	}

	user, err := u.userRepo.FindByActivationToken(ctx, token)
	if err != nil || user == nil {
		return fmt.Errorf("Invalid activation token")
	}

	if user.ActivationExpiresAt != nil && time.Now().After(*user.ActivationExpiresAt) {
		return fmt.Errorf("Activation token has expired")
	}

	user.IsActive = true
	user.ActivationToken = nil
	user.ActivationExpiresAt = nil

	if err := u.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("Failed to activate account: %w", err)
	}

	return nil
}
