package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/pkg/email"
	"github.com/bubblevy/bookstore/pkg/hash"
	j "github.com/bubblevy/bookstore/pkg/jwt"
	"github.com/bubblevy/bookstore/pkg/logger"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"golang.org/x/crypto/bcrypt"
)

type AuthUsecase interface {
	Login(ctx context.Context, email, password string) (*entity.AuthResponse, error)
	Register(ctx context.Context, name, email, password string) error
	RefreshToken(ctx context.Context, refreshToken string) (*entity.AuthResponse, error)
	ResendActivation(ctx context.Context, email string) error
	ActivateAccount(ctx context.Context, token string) error
	Logout(ctx context.Context, refreshToken string) error
	StartSessionCleanup(ctx context.Context, interval time.Duration)
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
		logger.Warn("login failed - user not found",
			"email", email,
			"reason", "user_not_found",
		)
		return nil, entity.ErrInvalidCredentials
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		logger.Warn("login failed - invalid password",
			"email", email,
			"user_id", user.ID,
			"reason", "invalid_password",
		)
		return nil, entity.ErrInvalidCredentials
	}

	if !user.IsActive {
		logger.Warn("login failed - account inactive",
			"email", email,
			"user_id", user.ID,
			"reason", "account_inactive",
		)
		return nil, entity.ErrAccountInactive
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
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	logger.Info("login successful",
		"email", email,
		"user_id", user.ID,
	)

	return &entity.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (u *authUsecase) Register(ctx context.Context, name, email, password string) error {
	_, err := u.userRepo.FindByEmail(ctx, email)
	if err == nil {
		logger.Warn("register failed - email already exists", "email", email)
		return entity.ErrEmailAlreadyRegistered
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error("register failed - database error", "email", email, "error", err)
		return fmt.Errorf("failed to check email: %w", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("register failed - hash password error", "email", email, "error", err)
		return entity.ErrFailedHashPassword
	}

	activationToken, err := generateRandomToken()
	if err != nil {
		logger.Error("register failed - token generation error", "email", email, "error", err)
		return entity.ErrFailedGenerateToken
	}
	expiresAt := time.Now().Add(24 * time.Hour)
	hashedActivationToken := hash.SHA256(activationToken)

	user := &entity.User{
		Name:                name,
		Email:               email,
		Role:                entity.RoleUser,
		Password:            string(hashedPassword),
		ActivationToken:     &hashedActivationToken,
		ActivationExpiresAt: &expiresAt,
	}

	if err := u.userRepo.Create(ctx, user); err != nil {
		logger.Error("register failed - create user error", "email", email, "error", err)
		return fmt.Errorf("failed to create user: %w", err)
	}

	logger.Info("user registered", "email", email, "user_id", user.ID)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	go func(bgCtx context.Context) {
		defer cancel()
		if err := u.emailSender.SendActivationEmail(bgCtx, user.Email, user.Name, activationToken); err != nil {
			logger.Error("failed to send activation email",
				"email", user.Email,
				"user_id", user.ID,
				"error", err,
			)
		} else {
			logger.Info("activation email sent",
				"email", user.Email,
				"user_id", user.ID,
			)
		}
	}(timeoutCtx)

	return nil
}

func (u *authUsecase) RefreshToken(ctx context.Context, refreshToken string) (*entity.AuthResponse, error) {
	token, err := u.jwtService.ValidateRefreshToken(refreshToken)
	if err != nil {
		logger.Warn("refresh token failed - invalid token", "reason", "invalid_token")
		return nil, entity.ErrInvalidRefreshToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		logger.Warn("refresh token failed - invalid claims", "reason", "invalid_claims")
		return nil, entity.ErrInvalidRefreshToken
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		logger.Warn("refresh token failed - invalid user_id", "reason", "invalid_user_id")
		return nil, entity.ErrInvalidUserId
	}

	userID := uint(userIDFloat)

	session, err := u.sessionRepo.FindActiveByTokenHash(ctx, hash.SHA256(refreshToken))
	if err != nil {
		logger.Warn("refresh token failed - session not found", "user_id", userID, "reason", "session_not_found")
		return nil, entity.ErrInvalidRefreshToken
	}

	if session.UserID != userID {
		logger.Warn("refresh token failed - session mismatch", "user_id", userID, "reason", "session_mismatch")
		return nil, entity.ErrInvalidRefreshToken
	}

	user, err := u.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		logger.Warn("refresh token failed - user not found", "user_id", userID)
		return nil, entity.ErrUserNotFound
	}

	if !user.IsActive {
		logger.Warn("refresh token failed - account inactive", "user_id", userID, "email", user.Email)
		return nil, entity.ErrAccountInactive
	}

	if err := u.sessionRepo.Revoke(ctx, session.ID); err != nil {
		logger.Error("refresh token failed - revoke session error", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to revoke session: %w", err)
	}

	accessToken, err := u.jwtService.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		logger.Error("refresh token failed - generate access token error", "user_id", userID, "error", err)
		return nil, err
	}

	newRefreshToken, err := u.jwtService.GenerateRefreshToken(user.ID)
	if err != nil {
		logger.Error("refresh token failed - generate refresh token error", "user_id", userID, "error", err)
		return nil, err
	}

	newSession := &entity.UserSession{
		UserID:           user.ID,
		RefreshTokenHash: hash.SHA256(newRefreshToken),
		ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
	}

	if err := u.sessionRepo.Create(ctx, newSession); err != nil {
		logger.Error("refresh token failed - create session error", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to create new session: %w", err)
	}

	logger.Info("token refreshed", "user_id", userID, "email", user.Email)

	return &entity.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (u *authUsecase) ResendActivation(ctx context.Context, email string) error {
	user, err := u.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Info("resend activation - email not found (silent)", "email", email)
			return nil
		}
		logger.Error("resend activation - database error", "email", email, "error", err)
		return fmt.Errorf("failed to fetch user: %w", err)
	}

	if user == nil || user.IsActive {
		logger.Info("resend activation - account already active or not found (silent)", "email", email)
		return nil
	}

	newToken, err := generateRandomToken()
	hashedActivationToken := hash.SHA256(newToken)
	if err != nil {
		logger.Error("resend activation - token generation error", "email", email, "error", err)
		return fmt.Errorf("failed to generate activation token: %w", err)
	}
	expiresAt := time.Now().Add(24 * time.Hour)

	user.ActivationToken = &hashedActivationToken
	user.ActivationExpiresAt = &expiresAt

	if err := u.userRepo.Update(ctx, user); err != nil {
		logger.Error("resend activation - update error", "email", email, "error", err)
		return fmt.Errorf("failed to update activation token: %w", err)
	}

	logger.Info("activation token regenerated", "email", email, "user_id", user.ID)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	go func(bgCtx context.Context) {
		defer cancel()
		if err := u.emailSender.SendActivationEmail(bgCtx, user.Email, user.Name, newToken); err != nil {
			logger.Error("resend activation - email send error",
				"email", user.Email,
				"user_id", user.ID,
				"error", err,
			)
		} else {
			logger.Info("activation email resent",
				"email", user.Email,
				"user_id", user.ID,
			)
		}
	}(timeoutCtx)

	return nil
}

func (u *authUsecase) ActivateAccount(ctx context.Context, token string) error {
	if token == "" {
		logger.Warn("activate account - token required")
		return entity.ErrActivationTokenRequired
	}

	token = hash.SHA256(token)
	user, err := u.userRepo.FindByActivationToken(ctx, token)
	if err != nil || user == nil {
		logger.Warn("activate account - invalid token")
		return entity.ErrInvalidActivationToken
	}

	if user.ActivationExpiresAt != nil && time.Now().After(*user.ActivationExpiresAt) {
		logger.Warn("activate account - token expired", "user_id", user.ID, "email", user.Email)
		return entity.ErrActivationTokenExpired
	}

	user.IsActive = true
	user.ActivationToken = nil
	user.ActivationExpiresAt = nil

	if err := u.userRepo.Update(ctx, user); err != nil {
		logger.Error("activate account - update error", "user_id", user.ID, "error", err)
		return fmt.Errorf("failed to activate account: %w", err)
	}

	logger.Info("account activated", "user_id", user.ID, "email", user.Email)

	return nil
}

func (u *authUsecase) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}

	tokenHash := hash.SHA256(refreshToken)
	session, err := u.sessionRepo.FindActiveByTokenHash(ctx, tokenHash)
	if err != nil {
		logger.Warn("logout - session not found", "reason", "session_not_found")
		return nil
	}

	if err := u.sessionRepo.Revoke(ctx, session.ID); err != nil {
		logger.Error("logout - revoke session error", "user_id", session.UserID, "session_id", session.ID, "error", err)
		return err
	}

	logger.Info("user logged out", "user_id", session.UserID)

	return nil
}

func (u *authUsecase) StartSessionCleanup(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				logger.Info("session cleanup stopped")
				return
			case <-ticker.C:
				deleted, err := u.sessionRepo.DeleteExpired(ctx)
				if err != nil {
					logger.Error("session cleanup error", "error", err)
				} else if deleted > 0 {
					logger.Info("session cleanup completed", "deleted", deleted)
				}
			}
		}
	}()
}
