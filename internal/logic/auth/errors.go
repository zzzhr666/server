package auth

import "errors"

var (
	ErrInvalidUsername    = errors.New("invalid username")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrAccountExists      = errors.New("account already exists")
	ErrAccountNotFound    = errors.New("account not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrSessionNotFound    = errors.New("session not found")
)
