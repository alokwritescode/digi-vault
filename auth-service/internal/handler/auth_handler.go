package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/alokwritescode/digi-vault/auth-service/internal/usecase"
	apperrors "github.com/alokwritescode/digi-vault/shared/pkg/errors"
	"github.com/alokwritescode/digi-vault/shared/pkg/jwt"
)

type AuthHandler struct {
	uc usecase.AuthUsecase
}

func NewAuthHandler(uc usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{uc: uc}
}

// errorToStatus maps every domain sentinel error to the correct HTTP status.
// Default is 500 — unknown errors must never leak internal details.
func errorToStatus(err error) int {
	switch {
	case errors.Is(err, apperrors.ErrUserAlreadyExists):
		return http.StatusConflict
	case errors.Is(err, apperrors.ErrUserNotFound):
		return http.StatusNotFound
	case errors.Is(err, apperrors.ErrUserAlreadyActive):
		return http.StatusBadRequest
	case errors.Is(err, apperrors.ErrOTPExpired):
		return http.StatusGone
	case errors.Is(err, apperrors.ErrOTPInvalid):
		return http.StatusBadRequest
	case errors.Is(err, apperrors.ErrInvalidCredentials):
		return http.StatusUnauthorized
	case errors.Is(err, apperrors.ErrUserNotVerified):
		return http.StatusForbidden
	case errors.Is(err, apperrors.ErrInvalidToken),
		errors.Is(err, apperrors.ErrTokenRevoked),
		errors.Is(err, apperrors.ErrTokenExpired),
		errors.Is(err, apperrors.ErrMissingToken):
		return http.StatusUnauthorized
	case errors.Is(err, apperrors.ErrValidation):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// ─── Request types ────────────────────────────────────────────────────────────

type registerRequest struct {
	Phone    string `json:"phone"    binding:"required"`
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type sendOTPRequest struct {
	Phone string `json:"phone" binding:"required"`
}

type verifyOTPRequest struct {
	Phone string `json:"phone" binding:"required"`
	OTP   string `json:"otp"   binding:"required"`
}

type loginRequest struct {
	Phone    string `json:"phone"    binding:"required"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ─── Handlers ────────────────────────────────────────────────────────────────

// POST /auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperrors.ErrValidation.Error()})
		return
	}

	if err := h.uc.Register(c.Request.Context(), req.Phone, req.Email, req.Password); err != nil {
		c.JSON(errorToStatus(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "OTP sent to phone"})
}

// POST /auth/send-otp
func (h *AuthHandler) SendOTP(c *gin.Context) {
	var req sendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperrors.ErrValidation.Error()})
		return
	}

	if err := h.uc.SendOTP(c.Request.Context(), req.Phone); err != nil {
		c.JSON(errorToStatus(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OTP sent"})
}

// POST /auth/verify-otp
func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req verifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperrors.ErrValidation.Error()})
		return
	}

	if err := h.uc.VerifyOTP(c.Request.Context(), req.Phone, req.OTP); err != nil {
		c.JSON(errorToStatus(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "phone verified"})
}

// POST /auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperrors.ErrValidation.Error()})
		return
	}

	pair, err := h.uc.Login(c.Request.Context(), req.Phone, req.Password)
	if err != nil {
		c.JSON(errorToStatus(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
	})
}

// POST /auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	rawToken, err := jwt.ExtractBearerToken(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": apperrors.ErrMissingToken.Error()})
		return
	}

	if err := h.uc.Logout(c.Request.Context(), rawToken); err != nil {
		c.JSON(errorToStatus(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// POST /auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperrors.ErrValidation.Error()})
		return
	}

	pair, err := h.uc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(errorToStatus(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
	})
}
