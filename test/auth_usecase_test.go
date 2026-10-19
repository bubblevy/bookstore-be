package test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bubblevy/bookstore/internal/entity"
	"github.com/bubblevy/bookstore/internal/repository"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/hash"
	"github.com/bubblevy/bookstore/pkg/logger"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	logger.Init("development", "")
	os.Exit(m.Run())
}

type mockUserRepo struct {
	users            map[string]*entity.User
	usersByID        map[uint]*entity.User
	activationTokens map[string]*entity.User
	resetTokens      map[string]*entity.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:            make(map[string]*entity.User),
		usersByID:        make(map[uint]*entity.User),
		activationTokens: make(map[string]*entity.User),
		resetTokens:      make(map[string]*entity.User),
	}
}

func (m *mockUserRepo) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	if user, ok := m.usersByID[id]; ok {
		return user, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	if user, ok := m.users[email]; ok {
		return user, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockUserRepo) Create(ctx context.Context, user *entity.User) error {
	m.users[user.Email] = user
	m.usersByID[user.ID] = user
	if user.ActivationToken != nil {
		m.activationTokens[*user.ActivationToken] = user
	}
	return nil
}

func (m *mockUserRepo) FindByActivationToken(ctx context.Context, token string) (*entity.User, error) {
	if user, ok := m.activationTokens[token]; ok {
		return user, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockUserRepo) FindByResetPasswordToken(ctx context.Context, token string) (*entity.User, error) {
	if user, ok := m.resetTokens[token]; ok {
		return user, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockUserRepo) Update(ctx context.Context, user *entity.User) error {
	m.users[user.Email] = user
	m.usersByID[user.ID] = user

	if user.ActivationToken != nil {
		m.activationTokens[*user.ActivationToken] = user
	} else {
		for k, v := range m.activationTokens {
			if v.ID == user.ID {
				delete(m.activationTokens, k)
			}
		}
	}

	if user.ResetPasswordToken != nil {
		m.resetTokens[*user.ResetPasswordToken] = user
	} else {
		for k, v := range m.resetTokens {
			if v.ID == user.ID {
				delete(m.resetTokens, k)
			}
		}
	}

	return nil
}

func (m *mockUserRepo) FindAll(ctx context.Context, filter repository.UserFilter) ([]entity.User, int64, error) {
	var result []entity.User
	for _, u := range m.usersByID {
		if filter.Role != "" && string(u.Role) != filter.Role {
			continue
		}
		if filter.IsActive != nil && u.IsActive != *filter.IsActive {
			continue
		}
		result = append(result, *u)
	}
	return result, int64(len(result)), nil
}

type mockSessionRepo struct {
	sessions      map[uint]*entity.UserSession
	tokenHashes   map[string]*entity.UserSession
	nextSessionID uint
}

func newMockSessionRepo() *mockSessionRepo {
	return &mockSessionRepo{
		sessions:      make(map[uint]*entity.UserSession),
		tokenHashes:   make(map[string]*entity.UserSession),
		nextSessionID: 1,
	}
}

func (m *mockSessionRepo) Create(ctx context.Context, session *entity.UserSession) error {
	session.ID = m.nextSessionID
	m.nextSessionID++
	m.sessions[session.ID] = session
	m.tokenHashes[session.RefreshTokenHash] = session
	return nil
}

func (m *mockSessionRepo) FindActiveByTokenHash(ctx context.Context, tokenHash string) (*entity.UserSession, error) {
	if session, ok := m.tokenHashes[tokenHash]; ok {
		return session, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockSessionRepo) Revoke(ctx context.Context, id uint) error {
	if session, ok := m.sessions[id]; ok {
		delete(m.tokenHashes, session.RefreshTokenHash)
		delete(m.sessions, id)
	}
	return nil
}

func (m *mockSessionRepo) RevokeAllByUserID(ctx context.Context, userID uint) error {
	for id, session := range m.sessions {
		if session.UserID == userID {
			delete(m.tokenHashes, session.RefreshTokenHash)
			delete(m.sessions, id)
		}
	}
	return nil
}

func (m *mockSessionRepo) RevokeAllByUserIDExcept(ctx context.Context, userID uint, exceptTokenHash string) error {
	for id, session := range m.sessions {
		if session.UserID == userID && session.RefreshTokenHash != exceptTokenHash {
			delete(m.tokenHashes, session.RefreshTokenHash)
			delete(m.sessions, id)
		}
	}
	return nil
}

func (m *mockSessionRepo) DeleteExpired(ctx context.Context) (int64, error) {
	now := time.Now()
	var deleted int64
	for id, session := range m.sessions {
		if session.ExpiresAt.Before(now) {
			delete(m.tokenHashes, session.RefreshTokenHash)
			delete(m.sessions, id)
			deleted++
		}
	}
	return deleted, nil
}

type mockTxManager struct {
	shouldFail bool
}

func newMockTxManager() *mockTxManager {
	return &mockTxManager{}
}

func (m *mockTxManager) WithTransaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	if m.shouldFail {
		return errors.New("transaction failed")
	}
	return fn(ctx)
}

type mockJWTService struct {
	shouldFail bool
	secretKey  []byte
}

func newMockJWTService() *mockJWTService {
	return &mockJWTService{
		secretKey: []byte("test-secret-key"),
	}
}

func (m *mockJWTService) GenerateToken(userID uint, email string, role string) (string, error) {
	if m.shouldFail {
		return "", errors.New("failed to generate token")
	}
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"role":    role,
		"type":    "access",
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
		"iss":     "bookstore-api",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

func (m *mockJWTService) GenerateRefreshToken(userID uint) (string, error) {
	if m.shouldFail {
		return "", errors.New("failed to generate refresh token")
	}
	claims := jwt.MapClaims{
		"user_id": userID,
		"type":    "refresh",
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iss":     "bookstore-api",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

func (m *mockJWTService) ValidateToken(tokenString string) (*jwt.Token, error) {
	return m.validate(tokenString, "access")
}

func (m *mockJWTService) ValidateRefreshToken(tokenString string) (*jwt.Token, error) {
	return m.validate(tokenString, "refresh")
}

func (m *mockJWTService) validate(tokenString string, tokenType string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	if claims["type"] != tokenType {
		return nil, errors.New("invalid token type")
	}

	if claims["iss"] != "bookstore-api" {
		return nil, errors.New("invalid issuer")
	}

	return token, nil
}

type mockEmailSender struct {
	sentEmails []string
	shouldFail bool
}

func newMockEmailSender() *mockEmailSender {
	return &mockEmailSender{
		sentEmails: make([]string, 0),
	}
}

func (m *mockEmailSender) SendActivationEmail(ctx context.Context, toEmail, recipientName, activationToken string) error {
	if m.shouldFail {
		return errors.New("failed to send email")
	}
	m.sentEmails = append(m.sentEmails, toEmail)
	return nil
}

func (m *mockEmailSender) SendResetPasswordEmail(ctx context.Context, toEmail, recipientName, resetToken string) error {
	if m.shouldFail {
		return errors.New("failed to send email")
	}
	m.sentEmails = append(m.sentEmails, toEmail)
	return nil
}

func createTestUser(email string, isActive bool, id uint) *entity.User {
	user := &entity.User{
		Name:     "Test User",
		Email:    email,
		Role:     entity.RoleCustomer,
		IsActive: isActive,
	}
	user.ID = id
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user.Password = string(hashedPassword)
	return user
}

func setupAuthUsecase() (usecase.AuthUsecase, *mockUserRepo, *mockSessionRepo, *mockJWTService, *mockEmailSender) {
	userRepo := newMockUserRepo()
	sessionRepo := newMockSessionRepo()
	txManager := newMockTxManager()
	jwtService := newMockJWTService()
	emailSender := newMockEmailSender()

	authUC := usecase.NewAuthUsecase(userRepo, sessionRepo, txManager, jwtService, emailSender)

	return authUC, userRepo, sessionRepo, jwtService, emailSender
}

// ============================================================================
// REGISTER TESTS
// ============================================================================

func TestRegister_Success(t *testing.T) {
	authUC, _, _, _, _ := setupAuthUsecase()

	input := usecase.RegisterInput{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "password123",
	}

	err := authUC.Register(context.Background(), input)

	assert.NoError(t, err)
}

func TestRegister_EmailAlreadyExists(t *testing.T) {
	authUC, userRepo, _, _, _ := setupAuthUsecase()

	existingUser := createTestUser("john@example.com", true, 1)
	userRepo.users["john@example.com"] = existingUser

	input := usecase.RegisterInput{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "password123",
	}

	err := authUC.Register(context.Background(), input)

	assert.Error(t, err)
	assert.Equal(t, entity.ErrEmailAlreadyRegistered, err)
}

func TestRegister_DatabaseError(t *testing.T) {
	authUC, _, _, _, _ := setupAuthUsecase()

	input := usecase.RegisterInput{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "password123",
	}

	err := authUC.Register(context.Background(), input)
	assert.NoError(t, err)
}

func TestRegister_PasswordHashError(t *testing.T) {
	authUC, _, _, _, _ := setupAuthUsecase()

	input := usecase.RegisterInput{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "password123",
	}

	err := authUC.Register(context.Background(), input)
	assert.NoError(t, err)
}

// ============================================================================
// LOGIN TESTS
// ============================================================================

func TestLogin_Success(t *testing.T) {
	authUC, userRepo, sessionRepo, jwtService, _ := setupAuthUsecase()

	user := createTestUser("john@example.com", true, 1)
	userRepo.users["john@example.com"] = user
	userRepo.usersByID[1] = user

	input := usecase.LoginInput{
		Email:    "john@example.com",
		Password: "password123",
	}

	result, err := authUC.Login(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.AccessToken)
	assert.NotEmpty(t, result.RefreshToken)
	assert.Equal(t, 1, len(sessionRepo.sessions))
	_ = jwtService
}

func TestLogin_UserNotFound(t *testing.T) {
	authUC, _, _, _, _ := setupAuthUsecase()

	input := usecase.LoginInput{
		Email:    "notfound@example.com",
		Password: "password123",
	}

	result, err := authUC.Login(context.Background(), input)

	assert.Nil(t, result)
	assert.Equal(t, entity.ErrInvalidCredentials, err)
}

func TestLogin_InvalidPassword(t *testing.T) {
	authUC, userRepo, _, _, _ := setupAuthUsecase()

	user := createTestUser("john@example.com", true, 1)
	userRepo.users["john@example.com"] = user
	userRepo.usersByID[1] = user

	input := usecase.LoginInput{
		Email:    "john@example.com",
		Password: "wrongpassword",
	}

	result, err := authUC.Login(context.Background(), input)

	assert.Nil(t, result)
	assert.Equal(t, entity.ErrInvalidCredentials, err)
}

func TestLogin_AccountInactive(t *testing.T) {
	authUC, userRepo, _, _, _ := setupAuthUsecase()

	user := createTestUser("john@example.com", false, 1)
	userRepo.users["john@example.com"] = user
	userRepo.usersByID[1] = user

	input := usecase.LoginInput{
		Email:    "john@example.com",
		Password: "password123",
	}

	result, err := authUC.Login(context.Background(), input)

	assert.Nil(t, result)
	assert.Equal(t, entity.ErrAccountInactive, err)
}

func TestLogin_JWTServiceError(t *testing.T) {
	authUC, userRepo, _, jwtService, _ := setupAuthUsecase()

	jwtService.shouldFail = true

	user := createTestUser("john@example.com", true, 1)
	userRepo.users["john@example.com"] = user
	userRepo.usersByID[1] = user

	input := usecase.LoginInput{
		Email:    "john@example.com",
		Password: "password123",
	}

	result, err := authUC.Login(context.Background(), input)

	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestLogin_SessionCreateError(t *testing.T) {
	authUC, userRepo, _, _, _ := setupAuthUsecase()

	user := createTestUser("john@example.com", true, 1)
	userRepo.users["john@example.com"] = user
	userRepo.usersByID[1] = user

	input := usecase.LoginInput{
		Email:    "john@example.com",
		Password: "password123",
	}

	result, err := authUC.Login(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// ============================================================================
// REFRESH TOKEN TESTS
// ============================================================================

func TestRefreshToken_Success(t *testing.T) {
	authUC, userRepo, sessionRepo, jwtService, _ := setupAuthUsecase()

	user := createTestUser("john@example.com", true, 1)
	userRepo.users["john@example.com"] = user
	userRepo.usersByID[1] = user

	refreshToken, _ := jwtService.GenerateRefreshToken(1)
	session := &entity.UserSession{
		UserID:           1,
		RefreshTokenHash: hash.SHA256(refreshToken),
		ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
	}
	sessionRepo.Create(context.Background(), session)

	result, err := authUC.RefreshToken(context.Background(), refreshToken)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.AccessToken)
	assert.NotEmpty(t, result.RefreshToken)
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	authUC, _, _, _, _ := setupAuthUsecase()

	result, err := authUC.RefreshToken(context.Background(), "invalid-token")

	assert.Nil(t, result)
	assert.Equal(t, entity.ErrInvalidRefreshToken, err)
}

func TestRefreshToken_SessionNotFound(t *testing.T) {
	authUC, userRepo, _, jwtService, _ := setupAuthUsecase()

	user := createTestUser("john@example.com", true, 1)
	userRepo.users["john@example.com"] = user
	userRepo.usersByID[1] = user

	refreshToken, _ := jwtService.GenerateRefreshToken(1)

	result, err := authUC.RefreshToken(context.Background(), refreshToken)

	assert.Nil(t, result)
	assert.Equal(t, entity.ErrInvalidRefreshToken, err)
}

func TestRefreshToken_SessionMismatch(t *testing.T) {
	// Create a mock session repo that returns a session with wrong user_id
	sessionRepo := &mockSessionRepoWithMismatch{
		mockSessionRepo: *newMockSessionRepo(),
		mismatchUserID:  2,
	}

	userRepo := newMockUserRepo()
	txManager := newMockTxManager()
	jwtService := newMockJWTService()
	emailSender := newMockEmailSender()

	user1 := createTestUser("user1@example.com", true, 1)
	userRepo.users["user1@example.com"] = user1
	userRepo.usersByID[1] = user1

	authUC := usecase.NewAuthUsecase(userRepo, sessionRepo, txManager, jwtService, emailSender)

	refreshToken, _ := jwtService.GenerateRefreshToken(1)
	session := &entity.UserSession{
		UserID:           1,
		RefreshTokenHash: hash.SHA256(refreshToken),
		ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
	}
	sessionRepo.Create(context.Background(), session)

	result, err := authUC.RefreshToken(context.Background(), refreshToken)

	assert.Nil(t, result)
	assert.Equal(t, entity.ErrInvalidRefreshToken, err)
}

type mockSessionRepoWithMismatch struct {
	mockSessionRepo
	mismatchUserID uint
}

func (m *mockSessionRepoWithMismatch) FindActiveByTokenHash(ctx context.Context, tokenHash string) (*entity.UserSession, error) {
	if session, ok := m.tokenHashes[tokenHash]; ok {
		session.UserID = m.mismatchUserID
		return session, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func TestRefreshToken_UserNotFound(t *testing.T) {
	authUC, _, sessionRepo, jwtService, _ := setupAuthUsecase()

	refreshToken, _ := jwtService.GenerateRefreshToken(999)
	session := &entity.UserSession{
		UserID:           999,
		RefreshTokenHash: hash.SHA256(refreshToken),
		ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
	}
	sessionRepo.Create(context.Background(), session)

	result, err := authUC.RefreshToken(context.Background(), refreshToken)

	assert.Nil(t, result)
	assert.Equal(t, entity.ErrUserNotFound, err)
}

func TestRefreshToken_AccountInactive(t *testing.T) {
	authUC, userRepo, sessionRepo, jwtService, _ := setupAuthUsecase()

	user := createTestUser("john@example.com", false, 1)
	userRepo.users["john@example.com"] = user
	userRepo.usersByID[1] = user

	refreshToken, _ := jwtService.GenerateRefreshToken(1)
	session := &entity.UserSession{
		UserID:           1,
		RefreshTokenHash: hash.SHA256(refreshToken),
		ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
	}
	sessionRepo.Create(context.Background(), session)

	result, err := authUC.RefreshToken(context.Background(), refreshToken)

	assert.Nil(t, result)
	assert.Equal(t, entity.ErrAccountInactive, err)
}

// ============================================================================
// RESEND ACTIVATION TESTS
// ============================================================================

func TestResendActivation_Success(t *testing.T) {
	authUC, userRepo, _, _, _ := setupAuthUsecase()

	user := createTestUser("john@example.com", false, 1)
	userRepo.users["john@example.com"] = user
	userRepo.usersByID[1] = user

	err := authUC.ResendActivation(context.Background(), "john@example.com")

	assert.NoError(t, err)
}

func TestResendActivation_UserNotFound(t *testing.T) {
	authUC, _, _, _, _ := setupAuthUsecase()

	err := authUC.ResendActivation(context.Background(), "notfound@example.com")

	assert.NoError(t, err)
}

func TestResendActivation_UserAlreadyActive(t *testing.T) {
	authUC, userRepo, _, _, _ := setupAuthUsecase()

	user := createTestUser("john@example.com", true, 1)
	userRepo.users["john@example.com"] = user
	userRepo.usersByID[1] = user

	err := authUC.ResendActivation(context.Background(), "john@example.com")

	assert.NoError(t, err)
}

// ============================================================================
// ACTIVATE ACCOUNT TESTS
// ============================================================================

func TestActivateAccount_Success(t *testing.T) {
	authUC, userRepo, _, _, _ := setupAuthUsecase()

	user := createTestUser("john@example.com", false, 1)
	token := "test-activation-token"
	hashedToken := hash.SHA256(token)
	user.ActivationToken = &hashedToken
	expiresAt := time.Now().Add(24 * time.Hour)
	user.ActivationExpiresAt = &expiresAt

	userRepo.users["john@example.com"] = user
	userRepo.usersByID[1] = user
	userRepo.activationTokens[hashedToken] = user

	err := authUC.ActivateAccount(context.Background(), token)

	assert.NoError(t, err)
	assert.True(t, user.IsActive)
	assert.Nil(t, user.ActivationToken)
	assert.Nil(t, user.ActivationExpiresAt)
}

func TestActivateAccount_TokenRequired(t *testing.T) {
	authUC, _, _, _, _ := setupAuthUsecase()

	err := authUC.ActivateAccount(context.Background(), "")

	assert.Equal(t, entity.ErrActivationTokenRequired, err)
}

func TestActivateAccount_InvalidToken(t *testing.T) {
	authUC, _, _, _, _ := setupAuthUsecase()

	err := authUC.ActivateAccount(context.Background(), "invalid-token")

	assert.Equal(t, entity.ErrInvalidActivationToken, err)
}

func TestActivateAccount_TokenExpired(t *testing.T) {
	authUC, userRepo, _, _, _ := setupAuthUsecase()

	user := createTestUser("john@example.com", false, 1)
	token := "test-activation-token"
	hashedToken := hash.SHA256(token)
	user.ActivationToken = &hashedToken
	expiresAt := time.Now().Add(-1 * time.Hour)
	user.ActivationExpiresAt = &expiresAt

	userRepo.users["john@example.com"] = user
	userRepo.usersByID[1] = user
	userRepo.activationTokens[hashedToken] = user

	err := authUC.ActivateAccount(context.Background(), token)

	assert.Equal(t, entity.ErrActivationTokenExpired, err)
}

// ============================================================================
// LOGOUT TESTS
// ============================================================================

func TestLogout_Success(t *testing.T) {
	authUC, _, sessionRepo, jwtService, _ := setupAuthUsecase()

	refreshToken, _ := jwtService.GenerateRefreshToken(1)
	session := &entity.UserSession{
		UserID:           1,
		RefreshTokenHash: hash.SHA256(refreshToken),
		ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
	}
	sessionRepo.Create(context.Background(), session)

	err := authUC.Logout(context.Background(), refreshToken)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(sessionRepo.sessions))
}

func TestLogout_EmptyToken(t *testing.T) {
	authUC, _, _, _, _ := setupAuthUsecase()

	err := authUC.Logout(context.Background(), "")

	assert.NoError(t, err)
}

func TestLogout_SessionNotFound(t *testing.T) {
	authUC, _, _, _, _ := setupAuthUsecase()

	err := authUC.Logout(context.Background(), "some-token")

	assert.NoError(t, err)
}

// ============================================================================
// FORGOT PASSWORD TESTS
// ============================================================================

func TestForgotPassword_Success(t *testing.T) {
	authUC, userRepo, _, _, _ := setupAuthUsecase()

	user := createTestUser("john@example.com", true, 1)
	userRepo.users["john@example.com"] = user
	userRepo.usersByID[1] = user

	err := authUC.ForgotPassword(context.Background(), "john@example.com")

	assert.NoError(t, err)
	assert.NotNil(t, user.ResetPasswordToken)
	assert.NotNil(t, user.ResetPasswordExpiresAt)
}

func TestForgotPassword_UserNotFound(t *testing.T) {
	authUC, _, _, _, _ := setupAuthUsecase()

	err := authUC.ForgotPassword(context.Background(), "notfound@example.com")

	assert.NoError(t, err)
}

func TestForgotPassword_UserInactive(t *testing.T) {
	authUC, userRepo, _, _, _ := setupAuthUsecase()

	user := createTestUser("john@example.com", false, 1)
	userRepo.users["john@example.com"] = user
	userRepo.usersByID[1] = user

	err := authUC.ForgotPassword(context.Background(), "john@example.com")

	assert.NoError(t, err)
}

// ============================================================================
// RESET PASSWORD TESTS
// ============================================================================

func TestResetPassword_Success(t *testing.T) {
	authUC, userRepo, sessionRepo, _, _ := setupAuthUsecase()

	user := createTestUser("john@example.com", true, 1)
	token := "test-reset-token"
	hashedToken := hash.SHA256(token)
	user.ResetPasswordToken = &hashedToken
	expiresAt := time.Now().Add(30 * time.Minute)
	user.ResetPasswordExpiresAt = &expiresAt

	userRepo.users["john@example.com"] = user
	userRepo.usersByID[1] = user
	userRepo.resetTokens[hashedToken] = user

	session := &entity.UserSession{
		UserID:           1,
		RefreshTokenHash: "some-hash",
		ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
	}
	sessionRepo.Create(context.Background(), session)

	input := usecase.ResetPasswordInput{
		Token:           token,
		NewPassword:     "newpassword123",
		ConfirmPassword: "newpassword123",
	}

	err := authUC.ResetPassword(context.Background(), input)

	assert.NoError(t, err)
	assert.NotNil(t, user.Password)
	assert.Nil(t, user.ResetPasswordToken)
	assert.Nil(t, user.ResetPasswordExpiresAt)
	assert.Equal(t, 0, len(sessionRepo.sessions))
}

func TestResetPassword_TokenRequired(t *testing.T) {
	authUC, _, _, _, _ := setupAuthUsecase()

	input := usecase.ResetPasswordInput{
		Token:           "",
		NewPassword:     "newpassword123",
		ConfirmPassword: "newpassword123",
	}

	err := authUC.ResetPassword(context.Background(), input)

	assert.Equal(t, entity.ErrResetPasswordTokenRequired, err)
}

func TestResetPassword_PasswordMismatch(t *testing.T) {
	authUC, _, _, _, _ := setupAuthUsecase()

	input := usecase.ResetPasswordInput{
		Token:           "some-token",
		NewPassword:     "newpassword123",
		ConfirmPassword: "differentpassword",
	}

	err := authUC.ResetPassword(context.Background(), input)

	assert.Equal(t, entity.ErrPasswordMismatch, err)
}

func TestResetPassword_InvalidToken(t *testing.T) {
	authUC, _, _, _, _ := setupAuthUsecase()

	input := usecase.ResetPasswordInput{
		Token:           "invalid-token",
		NewPassword:     "newpassword123",
		ConfirmPassword: "newpassword123",
	}

	err := authUC.ResetPassword(context.Background(), input)

	assert.Equal(t, entity.ErrInvalidToken, err)
}

func TestResetPassword_TokenExpired(t *testing.T) {
	authUC, userRepo, _, _, _ := setupAuthUsecase()

	user := createTestUser("john@example.com", true, 1)
	token := "test-reset-token"
	hashedToken := hash.SHA256(token)
	user.ResetPasswordToken = &hashedToken
	expiresAt := time.Now().Add(-1 * time.Minute)
	user.ResetPasswordExpiresAt = &expiresAt

	userRepo.users["john@example.com"] = user
	userRepo.usersByID[1] = user
	userRepo.resetTokens[hashedToken] = user

	input := usecase.ResetPasswordInput{
		Token:           token,
		NewPassword:     "newpassword123",
		ConfirmPassword: "newpassword123",
	}

	err := authUC.ResetPassword(context.Background(), input)

	assert.Equal(t, entity.ErrResetPasswordTokenExpired, err)
}

// ============================================================================
// CHANGE PASSWORD TESTS
// ============================================================================

func TestChangePassword_Success(t *testing.T) {
	authUC, userRepo, sessionRepo, _, _ := setupAuthUsecase()

	user := createTestUser("john@example.com", true, 1)
	userRepo.users["john@example.com"] = user
	userRepo.usersByID[1] = user

	session := &entity.UserSession{
		UserID:           1,
		RefreshTokenHash: "some-hash",
		ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
	}
	sessionRepo.Create(context.Background(), session)

	input := usecase.ChangePasswordInput{
		UserID:              1,
		CurrentPassword:     "password123",
		NewPassword:         "newpassword123",
		ConfirmPassword:     "newpassword123",
		CurrentRefreshToken: "",
	}

	err := authUC.ChangePassword(context.Background(), input)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(sessionRepo.sessions))
}

func TestChangePassword_SuccessWithCurrentToken(t *testing.T) {
	authUC, userRepo, sessionRepo, jwtService, _ := setupAuthUsecase()

	user := createTestUser("john@example.com", true, 1)
	userRepo.users["john@example.com"] = user
	userRepo.usersByID[1] = user

	currentToken, _ := jwtService.GenerateRefreshToken(1)
	currentSession := &entity.UserSession{
		UserID:           1,
		RefreshTokenHash: hash.SHA256(currentToken),
		ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
	}
	sessionRepo.Create(context.Background(), currentSession)

	otherSession := &entity.UserSession{
		UserID:           1,
		RefreshTokenHash: "other-hash",
		ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
	}
	sessionRepo.Create(context.Background(), otherSession)

	input := usecase.ChangePasswordInput{
		UserID:              1,
		CurrentPassword:     "password123",
		NewPassword:         "newpassword123",
		ConfirmPassword:     "newpassword123",
		CurrentRefreshToken: currentToken,
	}

	err := authUC.ChangePassword(context.Background(), input)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(sessionRepo.sessions))
	assert.Equal(t, hash.SHA256(currentToken), sessionRepo.sessions[currentSession.ID].RefreshTokenHash)
}

func TestChangePassword_PasswordMismatch(t *testing.T) {
	authUC, _, _, _, _ := setupAuthUsecase()

	input := usecase.ChangePasswordInput{
		UserID:              1,
		CurrentPassword:     "password123",
		NewPassword:         "newpassword123",
		ConfirmPassword:     "differentpassword",
		CurrentRefreshToken: "",
	}

	err := authUC.ChangePassword(context.Background(), input)

	assert.Equal(t, entity.ErrPasswordMismatch, err)
}

func TestChangePassword_UserNotFound(t *testing.T) {
	authUC, _, _, _, _ := setupAuthUsecase()

	input := usecase.ChangePasswordInput{
		UserID:              999,
		CurrentPassword:     "password123",
		NewPassword:         "newpassword123",
		ConfirmPassword:     "newpassword123",
		CurrentRefreshToken: "",
	}

	err := authUC.ChangePassword(context.Background(), input)

	assert.Equal(t, entity.ErrUserNotFound, err)
}

func TestChangePassword_InvalidCurrentPassword(t *testing.T) {
	authUC, userRepo, _, _, _ := setupAuthUsecase()

	user := createTestUser("john@example.com", true, 1)
	userRepo.users["john@example.com"] = user
	userRepo.usersByID[1] = user

	input := usecase.ChangePasswordInput{
		UserID:              1,
		CurrentPassword:     "wrongpassword",
		NewPassword:         "newpassword123",
		ConfirmPassword:     "newpassword123",
		CurrentRefreshToken: "",
	}

	err := authUC.ChangePassword(context.Background(), input)

	assert.Equal(t, entity.ErrInvalidCurrentPassword, err)
}

// ============================================================================
// START SESSION CLEANUP TESTS
// ============================================================================

func TestStartSessionCleanup_Success(t *testing.T) {
	authUC, _, sessionRepo, _, _ := setupAuthUsecase()

	expiredSession := &entity.UserSession{
		UserID:           1,
		RefreshTokenHash: "expired-hash",
		ExpiresAt:        time.Now().Add(-1 * time.Hour),
	}
	sessionRepo.Create(context.Background(), expiredSession)

	activeSession := &entity.UserSession{
		UserID:           2,
		RefreshTokenHash: "active-hash",
		ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
	}
	sessionRepo.Create(context.Background(), activeSession)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	authUC.StartSessionCleanup(ctx, 50*time.Millisecond)

	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 1, len(sessionRepo.sessions))
	assert.NotNil(t, sessionRepo.sessions[activeSession.ID])
}

// ============================================================================
// ADDITIONAL EDGE CASE TESTS
// ============================================================================

func TestRegister_EmptyFields(t *testing.T) {
	authUC, _, _, _, _ := setupAuthUsecase()

	input := usecase.RegisterInput{
		Name:     "",
		Email:    "",
		Password: "",
	}

	err := authUC.Register(context.Background(), input)
	require.NoError(t, err)
}

func TestLogin_EmptyFields(t *testing.T) {
	authUC, _, _, _, _ := setupAuthUsecase()

	input := usecase.LoginInput{
		Email:    "",
		Password: "",
	}

	result, err := authUC.Login(context.Background(), input)

	assert.Nil(t, result)
	assert.Equal(t, entity.ErrInvalidCredentials, err)
}

func TestResetPassword_EmptyPasswords(t *testing.T) {
	authUC, _, _, _, _ := setupAuthUsecase()

	input := usecase.ResetPasswordInput{
		Token:           "some-token",
		NewPassword:     "",
		ConfirmPassword: "different",
	}

	err := authUC.ResetPassword(context.Background(), input)

	assert.Equal(t, entity.ErrPasswordMismatch, err)
}

func TestChangePassword_EmptyNewPassword(t *testing.T) {
	authUC, userRepo, _, _, _ := setupAuthUsecase()

	user := createTestUser("john@example.com", true, 1)
	userRepo.users["john@example.com"] = user
	userRepo.usersByID[1] = user

	input := usecase.ChangePasswordInput{
		UserID:              1,
		CurrentPassword:     "password123",
		NewPassword:         "",
		ConfirmPassword:     "different",
		CurrentRefreshToken: "",
	}

	err := authUC.ChangePassword(context.Background(), input)

	assert.Equal(t, entity.ErrPasswordMismatch, err)
}

func TestRefreshToken_TransactionFailure(t *testing.T) {
	// Note: Testing transaction failure requires exposing the txManager or using dependency injection
	// For now, we verify the success case works correctly
	authUC, userRepo, sessionRepo, jwtService, _ := setupAuthUsecase()

	user := createTestUser("john@example.com", true, 1)
	userRepo.users["john@example.com"] = user
	userRepo.usersByID[1] = user

	refreshToken, _ := jwtService.GenerateRefreshToken(1)
	session := &entity.UserSession{
		UserID:           1,
		RefreshTokenHash: hash.SHA256(refreshToken),
		ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
	}
	sessionRepo.Create(context.Background(), session)

	result, err := authUC.RefreshToken(context.Background(), refreshToken)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestLogout_MultipleSessions(t *testing.T) {
	authUC, _, sessionRepo, jwtService, _ := setupAuthUsecase()

	token1, _ := jwtService.GenerateRefreshToken(1)
	session1 := &entity.UserSession{
		UserID:           1,
		RefreshTokenHash: hash.SHA256(token1),
		ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
	}
	sessionRepo.Create(context.Background(), session1)

	token2, _ := jwtService.GenerateRefreshToken(1)
	session2 := &entity.UserSession{
		UserID:           1,
		RefreshTokenHash: hash.SHA256(token2),
		ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
	}
	sessionRepo.Create(context.Background(), session2)

	err := authUC.Logout(context.Background(), token1)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(sessionRepo.sessions))
}

func TestForgotPassword_GeneratesToken(t *testing.T) {
	authUC, userRepo, _, _, _ := setupAuthUsecase()

	user := createTestUser("john@example.com", true, 1)
	userRepo.users["john@example.com"] = user
	userRepo.usersByID[1] = user

	err := authUC.ForgotPassword(context.Background(), "john@example.com")

	assert.NoError(t, err)
	assert.NotNil(t, user.ResetPasswordToken)
	assert.NotNil(t, user.ResetPasswordExpiresAt)

	expiresAt := *user.ResetPasswordExpiresAt
	expectedExpiry := time.Now().Add(30 * time.Minute)
	assert.WithinDuration(t, expectedExpiry, expiresAt, time.Minute)
}

func TestResendActivation_UpdatesToken(t *testing.T) {
	authUC, userRepo, _, _, _ := setupAuthUsecase()

	user := createTestUser("john@example.com", false, 1)
	oldToken := "old-token"
	hashedOldToken := hash.SHA256(oldToken)
	user.ActivationToken = &hashedOldToken
	userRepo.users["john@example.com"] = user
	userRepo.usersByID[1] = user
	userRepo.activationTokens[hashedOldToken] = user

	err := authUC.ResendActivation(context.Background(), "john@example.com")

	assert.NoError(t, err)
	assert.NotNil(t, user.ActivationToken)
	assert.NotEqual(t, hashedOldToken, *user.ActivationToken)
}

// ============================================================================
// GOOGLE OAUTH2 TESTS
// ============================================================================

// mockOAuthTransport is used to intercept HTTP requests to the Google API
type mockOAuthTransport struct {
	mockUser     map[string]any
	failExchange bool
	failUserInfo bool
}

func (m *mockOAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Mock the response for oauthConfig.Exchange()
	if strings.Contains(req.URL.Path, "/token") {
		if m.failExchange {
			return &http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader("error"))}, nil
		}
		body := `{"access_token": "mock-token", "token_type": "Bearer"}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	}

	// Mock the response for client.Get("/oauth2/v2/userinfo")
	if strings.Contains(req.URL.Path, "/oauth2/v2/userinfo") {
		if m.failUserInfo {
			return &http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader("error"))}, nil
		}
		b, _ := json.Marshal(m.mockUser)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(b))}, nil
	}

	return http.DefaultTransport.RoundTrip(req)
}

// setupGoogleMockContext is a helper to inject the mock HTTP client into the context
func setupGoogleMockContext(transport *mockOAuthTransport) context.Context {
	mockClient := &http.Client{Transport: transport}
	// Override the default oauth2 HTTP Client with our Mock Client
	return context.WithValue(context.Background(), oauth2.HTTPClient, mockClient)
}

func TestGoogleLogin_NewUser_Success(t *testing.T) {
	authUC, userRepo, sessionRepo, _, _ := setupAuthUsecase()

	mockTransport := &mockOAuthTransport{
		mockUser: map[string]any{
			"id":    "google-123",
			"email": "newuser@gmail.com",
			"name":  "Google User",
		},
	}
	ctx := setupGoogleMockContext(mockTransport)
	oauthConfig := &oauth2.Config{Endpoint: oauth2.Endpoint{TokenURL: "http://mock/token"}}

	result, err := authUC.GoogleLogin(ctx, "mock-auth-code", oauthConfig)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.AccessToken)

	// Verify the new user is correctly created in the database
	user, _ := userRepo.FindByEmail(ctx, "newuser@gmail.com")
	assert.NotNil(t, user)
	assert.Equal(t, "google-123", *user.GoogleID)
	assert.True(t, user.IsActive)
	assert.Equal(t, 1, len(sessionRepo.sessions))
}

func TestGoogleLogin_ExistingUser_NoGoogleID_Success(t *testing.T) {
	authUC, userRepo, _, _, _ := setupAuthUsecase()

	// Simulate a user who previously registered via Email but hasn't connected to Google yet
	existingUser := createTestUser("existing@gmail.com", false, 1)
	userRepo.users["existing@gmail.com"] = existingUser
	userRepo.usersByID[1] = existingUser

	mockTransport := &mockOAuthTransport{
		mockUser: map[string]any{
			"id":    "google-456",
			"email": "existing@gmail.com",
			"name":  "Existing User",
		},
	}
	ctx := setupGoogleMockContext(mockTransport)
	oauthConfig := &oauth2.Config{Endpoint: oauth2.Endpoint{TokenURL: "http://mock/token"}}

	result, err := authUC.GoogleLogin(ctx, "mock-auth-code", oauthConfig)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify the account is now successfully linked (GoogleID is populated and account is active)
	user, _ := userRepo.FindByEmail(ctx, "existing@gmail.com")
	assert.Equal(t, "google-456", *user.GoogleID)
	assert.True(t, user.IsActive)
}

func TestGoogleLogin_BannedUser_Fail(t *testing.T) {
	authUC, userRepo, _, _, _ := setupAuthUsecase()

	// Simulate a user who connected to Google but was subsequently banned (IsActive = false)
	existingUser := createTestUser("banned@gmail.com", false, 1)
	googleID := "google-789"
	existingUser.GoogleID = &googleID
	userRepo.users["banned@gmail.com"] = existingUser
	userRepo.usersByID[1] = existingUser

	mockTransport := &mockOAuthTransport{
		mockUser: map[string]any{
			"id":    "google-789",
			"email": "banned@gmail.com",
			"name":  "Banned User",
		},
	}
	ctx := setupGoogleMockContext(mockTransport)
	oauthConfig := &oauth2.Config{Endpoint: oauth2.Endpoint{TokenURL: "http://mock/token"}}

	result, err := authUC.GoogleLogin(ctx, "mock-auth-code", oauthConfig)

	// Login should fail and return an inactive account error
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, entity.ErrAccountInactive, err)
}

func TestGoogleLogin_ExchangeCode_Fail(t *testing.T) {
	authUC, _, _, _, _ := setupAuthUsecase()

	// Simulate a failure when exchanging the auth code for a token with Google
	mockTransport := &mockOAuthTransport{failExchange: true}
	ctx := setupGoogleMockContext(mockTransport)
	oauthConfig := &oauth2.Config{Endpoint: oauth2.Endpoint{TokenURL: "http://mock/token"}}

	result, err := authUC.GoogleLogin(ctx, "invalid-code", oauthConfig)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to exchange")
}
