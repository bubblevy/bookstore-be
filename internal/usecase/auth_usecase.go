package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
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
	"golang.org/x/oauth2"
	"net/http"
	"net/url"
	"strings"
)

type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

type ResetPasswordInput struct {
	Token           string
	NewPassword     string
	ConfirmPassword string
}

type ChangePasswordInput struct {
	UserID              uint
	CurrentPassword     string
	NewPassword         string
	ConfirmPassword     string
	CurrentRefreshToken string
}

type AuthResult struct {
	AccessToken  string
	RefreshToken string
}

type GoogleUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type AuthUsecase interface {
	Login(ctx context.Context, i LoginInput) (*AuthResult, error)
	Register(ctx context.Context, i RegisterInput) error
	RefreshToken(ctx context.Context, refreshToken string) (*AuthResult, error)
	ResendActivation(ctx context.Context, email string) error
	ActivateAccount(ctx context.Context, token string) error
	Logout(ctx context.Context, refreshToken string) error
	StartSessionCleanup(ctx context.Context, interval time.Duration)
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, i ResetPasswordInput) error
	ChangePassword(ctx context.Context, i ChangePasswordInput) error
	GoogleLogin(ctx context.Context, code string, oauthConfig *oauth2.Config) (*AuthResult, error)
	GoogleMobileLogin(ctx context.Context, idToken, code string, oauthConfig *oauth2.Config) (*AuthResult, error)
}

type authUsecase struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	txManager   repository.TransactionManager
	jwtService  j.JWTService
	emailSender email.EmailSender
}

func NewAuthUsecase(userRepo repository.UserRepository, sessionRepo repository.SessionRepository, txManager repository.TransactionManager, jwtService j.JWTService, emailSender email.EmailSender) AuthUsecase {
	return &authUsecase{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		txManager:   txManager,
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

func (u *authUsecase) Register(ctx context.Context, i RegisterInput) error {
	_, err := u.userRepo.FindByEmail(ctx, i.Email)
	if err == nil {
		logger.Warn("register failed - email already exists")
		return entity.ErrEmailAlreadyRegistered
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error("register failed - database error", "error", err)
		return fmt.Errorf("failed to check email: %w", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(i.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("register failed - hash password error", "error", err)
		return entity.ErrFailedHashPassword
	}

	activationToken, err := generateRandomToken()
	if err != nil {
		logger.Error("register failed - token generation error", "error", err)
		return entity.ErrFailedGenerateToken
	}
	expiresAt := time.Now().Add(24 * time.Hour)
	hashedActivationToken := hash.SHA256(activationToken)

	user := &entity.User{
		Name:                i.Name,
		Email:               i.Email,
		Role:                entity.RoleCustomer,
		Password:            string(hashedPassword),
		ActivationToken:     &hashedActivationToken,
		ActivationExpiresAt: &expiresAt,
	}

	if err := u.userRepo.Create(ctx, user); err != nil {
		logger.Error("register failed - create user error", "error", err)
		return fmt.Errorf("failed to create user: %w", err)
	}

	logger.Info("user registered", "user_id", user.ID)

	// send email
	go func(name, email, token string, userID uint) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := u.emailSender.SendActivationEmail(bgCtx, email, name, token); err != nil {
			logger.Error("failed to send activation email", "user_id", userID, "error", err)
		} else {
			logger.Info("activation email sent", "user_id", user.ID)
		}
	}(user.Name, user.Email, activationToken, user.ID)

	return nil
}

func (u *authUsecase) Login(ctx context.Context, i LoginInput) (*AuthResult, error) {
	user, err := u.userRepo.FindByEmail(ctx, i.Email)
	if err != nil {
		logger.Warn("login failed - user not found",
			"reason", "user_not_found",
		)
		return nil, entity.ErrInvalidCredentials
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(i.Password)); err != nil {
		logger.Warn("login failed - invalid password",
			"user_id", user.ID,
			"reason", "invalid_password",
		)
		return nil, entity.ErrInvalidCredentials
	}

	if !user.IsActive {
		logger.Warn("login failed - account inactive",
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
		"user_id", user.ID,
	)

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (u *authUsecase) RefreshToken(ctx context.Context, refreshToken string) (*AuthResult, error) {
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
		logger.Warn("refresh token failed - account inactive", "user_id", userID)
		return nil, entity.ErrAccountInactive
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

	err = u.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := u.sessionRepo.Revoke(txCtx, session.ID); err != nil {
			logger.Error("refresh token failed - revoke refresh token session error", "user_id", userID, "error", err)
			return fmt.Errorf("failed to revoke refresh token session: %w", err)
		}

		if err := u.sessionRepo.Create(txCtx, newSession); err != nil {
			logger.Error("refresh token failed - create refresh token session error", "user_id", userID, "error", err)
			return fmt.Errorf("failed to create new refresh token session: %w", err)
		}
		return nil
	})

	if err != nil {
		logger.Error("refresh token failed - transaction error", "user_id", userID, "error", err)
		return nil, err
	}

	logger.Info("token refreshed", "user_id", userID)

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (u *authUsecase) ResendActivation(ctx context.Context, email string) error {
	user, err := u.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Info("resend activation - email not found (silent)")
			return nil
		}
		logger.Error("resend activation - database error", "error", err)
		return fmt.Errorf("failed to fetch user: %w", err)
	}

	if user == nil || user.IsActive {
		logger.Info("resend activation - account already active or not found (silent)")
		return nil
	}

	newToken, err := generateRandomToken()
	if err != nil {
		logger.Error("resend activation - token generation error", "error", err)
		return fmt.Errorf("failed to generate activation token: %w", err)
	}

	hashedActivationToken := hash.SHA256(newToken)
	expiresAt := time.Now().Add(24 * time.Hour)

	user.ActivationToken = &hashedActivationToken
	user.ActivationExpiresAt = &expiresAt

	if err := u.userRepo.Update(ctx, user); err != nil {
		logger.Error("resend activation - update error", "error", err)
		return fmt.Errorf("failed to update activation token: %w", err)
	}

	logger.Info("activation token regenerated", "user_id", user.ID)

	go func(targetEmail, targetName, token string, userID uint) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := u.emailSender.SendActivationEmail(bgCtx, targetEmail, targetName, token); err != nil {
			logger.Error("resend activation - email send error",
				"user_id", userID,
				"error", err,
			)
		} else {
			logger.Info("activation email resent",
				"user_id", userID,
			)
		}
	}(user.Email, user.Name, newToken, user.ID)

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

	if user.IsActive {
		user.ActivationToken = nil
		user.ActivationExpiresAt = nil
		_ = u.userRepo.Update(ctx, user)

		logger.Info("activate account - user already active", "user_id", user.ID)
		return entity.ErrAccountAlreadyActive
	}

	if user.ActivationExpiresAt != nil && time.Now().After(*user.ActivationExpiresAt) {
		logger.Warn("activate account - token expired", "user_id", user.ID)
		return entity.ErrActivationTokenExpired
	}

	user.IsActive = true
	user.ActivationToken = nil
	user.ActivationExpiresAt = nil

	if err := u.userRepo.Update(ctx, user); err != nil {
		logger.Error("activate account - update error", "user_id", user.ID, "error", err)
		return fmt.Errorf("failed to activate account: %w", err)
	}

	logger.Info("account activated", "user_id", user.ID)

	return nil
}

func (u *authUsecase) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}

	tokenHash := hash.SHA256(refreshToken)
	session, err := u.sessionRepo.FindActiveByTokenHash(ctx, tokenHash)
	if err != nil {
		logger.Warn("logout - refresh token session not found", "reason", "session_not_found")
		return nil
	}

	if err := u.sessionRepo.Revoke(ctx, session.ID); err != nil {
		logger.Error("logout - revoke refresh token session error", "user_id", session.UserID, "session_id", session.ID, "error", err)
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
				logger.Info("refresh token session cleanup stopped")
				return
			case <-ticker.C:
				deleted, err := u.sessionRepo.DeleteExpired(ctx)
				if err != nil {
					logger.Error("refresh token session cleanup error", "error", err)
				} else if deleted > 0 {
					logger.Info("refresh token session cleanup completed", "deleted", deleted)
				}
			}
		}
	}()
}

func (u *authUsecase) ForgotPassword(ctx context.Context, email string) error {
	user, err := u.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Info("forgot password - email not found (silent)")
			return nil
		}
		logger.Error("forgot password - database error", "error", err)
		return fmt.Errorf("failed to fetch user: %w", err)
	}

	if user == nil || !user.IsActive {
		logger.Info("forgot password - account inactive or not found (silent)")
		return nil
	}

	token, err := generateRandomToken()
	if err != nil {
		logger.Error("forgot password - token generation error", "error", err)
		return fmt.Errorf("failed to generate forgot password token: %w", err)
	}
	hashedResetPasswordToken := hash.SHA256(token)
	expiresAt := time.Now().Add(30 * time.Minute)

	user.ResetPasswordToken = &hashedResetPasswordToken
	user.ResetPasswordExpiresAt = &expiresAt

	if err := u.userRepo.Update(ctx, user); err != nil {
		logger.Error("forgot password - update error", "error", err)
		return fmt.Errorf("failed to update forgot password token: %w", err)
	}

	go func(targetEmail, targetName, resetToken string, userID uint) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := u.emailSender.SendResetPasswordEmail(bgCtx, targetEmail, targetName, resetToken); err != nil {
			logger.Error("forgot password - email send error", "user_id", userID, "error", err)
		} else {
			logger.Info("forgot password email sent", "user_id", userID)
		}
	}(user.Email, user.Name, token, user.ID)

	return nil
}

func (u *authUsecase) ResetPassword(ctx context.Context, i ResetPasswordInput) error {
	if i.Token == "" {
		logger.Warn("reset password account - token required")
		return entity.ErrResetPasswordTokenRequired
	}

	if i.NewPassword != i.ConfirmPassword {
		logger.Warn("reset password account - password confirmation mismatch")
		return entity.ErrPasswordMismatch
	}

	hashedToken := hash.SHA256(i.Token)
	user, err := u.userRepo.FindByResetPasswordToken(ctx, hashedToken)
	if err != nil || user == nil {
		logger.Warn("reset password account - invalid token")
		return entity.ErrInvalidToken
	}

	if user.ResetPasswordExpiresAt != nil && time.Now().After(*user.ResetPasswordExpiresAt) {
		logger.Warn("reset password account - token expired", "user_id", user.ID)
		return entity.ErrResetPasswordTokenExpired
	}

	hashedNewPassword, err := bcrypt.GenerateFromPassword([]byte(i.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("reset password failed - hash password error", "error", err)
		return entity.ErrFailedHashPassword
	}

	user.Password = string(hashedNewPassword)
	user.ResetPasswordToken = nil
	user.ResetPasswordExpiresAt = nil

	err = u.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := u.userRepo.Update(txCtx, user); err != nil {
			logger.Error("reset password account - update error", "user_id", user.ID, "error", err)
			return fmt.Errorf("failed to reset password account: %w", err)
		}

		if err := u.sessionRepo.RevokeAllByUserID(txCtx, user.ID); err != nil {
			logger.Error("reset password account - failed to revoke refresh token sessions after password reset", "user_id", user.ID, "error", err)
			return fmt.Errorf("failed to revoke refresh token sessions: %w", err)
		}

		return nil
	})

	if err != nil {
		logger.Error("reset password account - transaction error", "user_id", user.ID, "error", err)
		return err
	}

	logger.Info("reset password account - password changed and refresh token sessions revoked", "user_id", user.ID)
	return nil
}

func (u *authUsecase) ChangePassword(ctx context.Context, i ChangePasswordInput) error {
	if i.NewPassword != i.ConfirmPassword {
		logger.Warn("change password account - password confirmation mismatch")
		return entity.ErrPasswordMismatch
	}

	user, err := u.userRepo.FindByID(ctx, i.UserID)
	if err != nil || user == nil {
		logger.Warn("change password account - user not found")
		return entity.ErrUserNotFound
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(i.CurrentPassword)); err != nil {
		logger.Error("change password account - old password incorrect", "user_id", user.ID)
		return entity.ErrInvalidCurrentPassword
	}

	hashedNewPassword, err := bcrypt.GenerateFromPassword([]byte(i.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("change password failed - hash password error", "error", err)
		return entity.ErrFailedHashPassword
	}

	user.Password = string(hashedNewPassword)

	err = u.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := u.userRepo.Update(txCtx, user); err != nil {
			logger.Error("change password account - update error", "user_id", user.ID, "error", err)
			return fmt.Errorf("failed to change password: %w", err)
		}

		if i.CurrentRefreshToken != "" {
			currentTokenHash := hash.SHA256(i.CurrentRefreshToken)
			if err := u.sessionRepo.RevokeAllByUserIDExcept(txCtx, user.ID, currentTokenHash); err != nil {
				logger.Error("change password account - revoke all refresh token by user id except error", "user_id", user.ID, "error", err)
				return fmt.Errorf("failed to revoke other refresh token sessions: %w", err)
			}
		} else {
			if err := u.sessionRepo.RevokeAllByUserID(txCtx, user.ID); err != nil {
				logger.Error("change password account - revoke all refresh token sessions", "user_id", user.ID, "error", err)
				return fmt.Errorf("failed to revoke all refresh token sessions: %w", err)
			}
		}
		return nil
	})

	if err != nil {
		logger.Error("change password account - transaction error", "user_id", user.ID, "error", err)
		return err
	}

	logger.Info("change password account - password changed", "user_id", user.ID)
	return nil
}

func (u *authUsecase) processGoogleUser(ctx context.Context, googleUser GoogleUser) (*AuthResult, error) {
	var accessToken, refreshToken string
	err := u.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		user, err := u.userRepo.FindByEmail(txCtx, googleUser.Email)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				user = &entity.User{
					Name:     googleUser.Name,
					Email:    googleUser.Email,
					GoogleID: &googleUser.ID,
					Role:     entity.RoleCustomer,
					IsActive: true,
				}
				if err := u.userRepo.Create(txCtx, user); err != nil {
					return fmt.Errorf("failed to create user: %w", err)
				}
				logger.Info("new user registered via google", "user_id", user.ID)
			} else {
				return fmt.Errorf("failed to check user: %w", err)
			}
		} else {
			if user.GoogleID == nil {
				user.GoogleID = &googleUser.ID
				user.IsActive = true
				if err := u.userRepo.Update(txCtx, user); err != nil {
					return fmt.Errorf("failed to update user google id: %w", err)
				}
			} else {
				if !user.IsActive {
					logger.Warn("google login failed - account inactive", "user_id", user.ID)
					return entity.ErrAccountInactive
				}
			}
		}

		accessToken, err = u.jwtService.GenerateToken(user.ID, user.Email, string(user.Role))
		if err != nil {
			return fmt.Errorf("failed to generate access token: %w", err)
		}

		refreshToken, err = u.jwtService.GenerateRefreshToken(user.ID)
		if err != nil {
			return fmt.Errorf("failed to generate refresh token: %w", err)
		}

		session := &entity.UserSession{
			UserID:           user.ID,
			RefreshTokenHash: hash.SHA256(refreshToken),
			ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
		}

		if err := u.sessionRepo.Create(txCtx, session); err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}

		return nil
	})

	if err != nil {
		logger.Error("google login - transaction failed", "error", err)
		return nil, err
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (u *authUsecase) GoogleLogin(ctx context.Context, code string, oauthConfig *oauth2.Config) (*AuthResult, error) {
	if oauthConfig == nil {
		return nil, errors.New("oauth config not initialized")
	}

	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		logger.Error("google login - failed to exchange token", "error", err)
		return nil, errors.New("failed to exchange google token")
	}

	client := oauthConfig.Client(ctx, token)
	response, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		logger.Error("google login - failed to get user info", "error", err)
		return nil, errors.New("failed to get user info from google")
	}
	defer response.Body.Close()

	var googleUser GoogleUser
	if err := json.NewDecoder(response.Body).Decode(&googleUser); err != nil {
		return nil, errors.New("failed to decode google user info")
	}

	return u.processGoogleUser(ctx, googleUser)
}

func (u *authUsecase) GoogleMobileLogin(ctx context.Context, idToken, code string, oauthConfig *oauth2.Config) (*AuthResult, error) {
	if idToken != "" {
		tokenURL := "https://oauth2.googleapis.com/tokeninfo?id_token=" + url.QueryEscape(idToken)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create google request: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			logger.Error("google mobile login - tokeninfo request error", "error", err)
			return nil, errors.New("failed to verify google token with provider")
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			logger.Warn("google mobile login - invalid id_token", "status", resp.StatusCode)
			return nil, errors.New("invalid google id_token")
		}

		var tokenInfo struct {
			Sub   string `json:"sub"`
			Email string `json:"email"`
			Name  string `json:"name"`
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
			return nil, errors.New("failed to parse google token response")
		}

		if tokenInfo.Error != "" || tokenInfo.Email == "" {
			return nil, errors.New("invalid google token payload")
		}

		name := tokenInfo.Name
		if name == "" {
			name = strings.Split(tokenInfo.Email, "@")[0]
		}

		googleUser := GoogleUser{
			ID:    tokenInfo.Sub,
			Email: tokenInfo.Email,
			Name:  name,
		}

		return u.processGoogleUser(ctx, googleUser)
	}

	if code != "" {
		return u.GoogleLogin(ctx, code, oauthConfig)
	}

	return nil, errors.New("either id_token or code is required")
}
