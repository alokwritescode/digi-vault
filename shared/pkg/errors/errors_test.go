package errors_test

import (
	stderrors "errors"
	"fmt"
	"testing"

	"github.com/alokwritescode/digi-vault/shared/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestSentinelErrors_NotNil(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		// auth
		{"ErrUserAlreadyExists", errors.ErrUserAlreadyExists},
		{"ErrUserNotFound", errors.ErrUserNotFound},
		{"ErrUserAlreadyActive", errors.ErrUserAlreadyActive},
		{"ErrUserNotVerified", errors.ErrUserNotVerified},
		{"ErrOTPExpired", errors.ErrOTPExpired},
		{"ErrOTPInvalid", errors.ErrOTPInvalid},
		{"ErrInvalidCredentials", errors.ErrInvalidCredentials},
		{"ErrInvalidToken", errors.ErrInvalidToken},
		{"ErrTokenRevoked", errors.ErrTokenRevoked},
		{"ErrTokenExpired", errors.ErrTokenExpired},
		{"ErrMissingToken", errors.ErrMissingToken},
		{"ErrValidation", errors.ErrValidation},
		// wallet
		{"ErrWalletAlreadyExists", errors.ErrWalletAlreadyExists},
		{"ErrWalletNotFound", errors.ErrWalletNotFound},
		{"ErrInsufficientBalance", errors.ErrInsufficientBalance},
		{"ErrWalletLocked", errors.ErrWalletLocked},
		// txn
		{"ErrSelfTransfer", errors.ErrSelfTransfer},
		{"ErrInvalidAmount", errors.ErrInvalidAmount},
		{"ErrTransactionNotFound", errors.ErrTransactionNotFound},
		// general
		{"ErrInternalServer", errors.ErrInternalServer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
		})
	}
}

func TestSentinelErrors_ErrorsIs(t *testing.T) {
	tests := []struct {
		name   string
		target error
	}{
		{"ErrUserNotFound wrapped", errors.ErrUserNotFound},
		{"ErrInvalidToken wrapped", errors.ErrInvalidToken},
		{"ErrInsufficientBalance wrapped", errors.ErrInsufficientBalance},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := fmt.Errorf("layer: %w", tt.target)
			assert.True(t, stderrors.Is(wrapped, tt.target))
		})
	}
}

func TestSentinelErrors_UniqueMessages(t *testing.T) {
	allErrors := []struct {
		name string
		err  error
	}{
		{"ErrUserAlreadyExists", errors.ErrUserAlreadyExists},
		{"ErrUserNotFound", errors.ErrUserNotFound},
		{"ErrUserAlreadyActive", errors.ErrUserAlreadyActive},
		{"ErrUserNotVerified", errors.ErrUserNotVerified},
		{"ErrOTPExpired", errors.ErrOTPExpired},
		{"ErrOTPInvalid", errors.ErrOTPInvalid},
		{"ErrInvalidCredentials", errors.ErrInvalidCredentials},
		{"ErrInvalidToken", errors.ErrInvalidToken},
		{"ErrTokenRevoked", errors.ErrTokenRevoked},
		{"ErrTokenExpired", errors.ErrTokenExpired},
		{"ErrMissingToken", errors.ErrMissingToken},
		{"ErrValidation", errors.ErrValidation},
		{"ErrWalletAlreadyExists", errors.ErrWalletAlreadyExists},
		{"ErrWalletNotFound", errors.ErrWalletNotFound},
		{"ErrInsufficientBalance", errors.ErrInsufficientBalance},
		{"ErrWalletLocked", errors.ErrWalletLocked},
		{"ErrSelfTransfer", errors.ErrSelfTransfer},
		{"ErrInvalidAmount", errors.ErrInvalidAmount},
		{"ErrTransactionNotFound", errors.ErrTransactionNotFound},
		{"ErrInternalServer", errors.ErrInternalServer},
	}

	seen := make(map[string]string)
	for _, e := range allErrors {
		msg := e.err.Error()
		if prev, exists := seen[msg]; exists {
			t.Errorf("duplicate error message %q: used by %s and %s", msg, prev, e.name)
		}
		seen[msg] = e.name
	}
}
