package errors

import "errors"

// Auth errors
var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyActive  = errors.New("user already active")
	ErrUserNotVerified    = errors.New("user not verified")
	ErrOTPExpired         = errors.New("otp expired")
	ErrOTPInvalid         = errors.New("otp invalid")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenRevoked       = errors.New("token revoked")
	ErrTokenExpired       = errors.New("token expired")
	ErrMissingToken       = errors.New("missing token")
	ErrValidation         = errors.New("validation error")
)

// Wallet errors
var (
	ErrWalletAlreadyExists = errors.New("wallet already exists")
	ErrWalletNotFound      = errors.New("wallet not found")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrWalletLocked        = errors.New("wallet locked")
)

// Transaction errors
var (
	ErrSelfTransfer        = errors.New("self transfer not allowed")
	ErrInvalidAmount       = errors.New("invalid amount")
	ErrTransactionNotFound = errors.New("transaction not found")
)

// General errors
var (
	ErrInternalServer = errors.New("internal server error")
)
