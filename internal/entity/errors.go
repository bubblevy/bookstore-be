package entity

import "errors"

var (
	ErrInvalidCredentials          = errors.New("invalid credentials")
	ErrAccountInactive             = errors.New("account is inactive")
	ErrEmailAlreadyRegistered      = errors.New("email already registered")
	ErrUserNotFound                = errors.New("user not found")
	ErrInvalidUserId               = errors.New("invalid user id")
	ErrInvalidToken                = errors.New("invalid or expired token")
	ErrInvalidTokenClaims          = errors.New("invalid token claims")
	ErrInvalidTokenType            = errors.New("invalid token type")
	ErrInvalidTokenIssuer          = errors.New("invalid token issuer")
	ErrInvalidRefreshToken         = errors.New("invalid or expired refresh token")
	ErrFailedHashPassword          = errors.New("failed to hash password")
	ErrFailedGenerateToken         = errors.New("failed to generate token")
	ErrActivationTokenRequired     = errors.New("activation token is required")
	ErrInvalidActivationToken      = errors.New("invalid activation token")
	ErrActivationTokenExpired      = errors.New("activation token has expired")
	ErrChangePasswordTokenRequired = errors.New("change password token is required")
	ErrChangePasswordTokenExpired  = errors.New("change password token has expired")
	ErrPasswordMismatch            = errors.New("password confirmation mismatch")
	ErrResetPasswordTokenRequired  = errors.New("reset password token is required")
	ErrResetPasswordTokenExpired   = errors.New("reset password token has expired")
	ErrInvalidCurrentPassword      = errors.New("current password is incorrect")
	ErrAccountAlreadyActive        = errors.New("account is already active")
)
