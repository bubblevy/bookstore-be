package entity

import "errors"

var (
	ErrInvalidCredentials     = errors.New("Invalid credentials")
	ErrAccountInactive        = errors.New("Account is inactive")
	ErrEmailAlreadyRegistered = errors.New("Email already registered")
	ErrUserNotFound           = errors.New("User not found")
	ErrInvalidToken           = errors.New("Invalid or expired token")
)
