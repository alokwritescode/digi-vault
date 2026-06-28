package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/alokwritescode/digi-vault/auth-service/internal/handler"
	"github.com/alokwritescode/digi-vault/auth-service/internal/usecase"
	apperrors "github.com/alokwritescode/digi-vault/shared/pkg/errors"
)

// ─── Mock ────────────────────────────────────────────────────────────────────

type MockAuthUsecase struct{ mock.Mock }

func (m *MockAuthUsecase) Register(ctx context.Context, phone, email, password string) error {
	return m.Called(ctx, phone, email, password).Error(0)
}
func (m *MockAuthUsecase) SendOTP(ctx context.Context, phone string) error {
	return m.Called(ctx, phone).Error(0)
}
func (m *MockAuthUsecase) VerifyOTP(ctx context.Context, phone, otp string) error {
	return m.Called(ctx, phone, otp).Error(0)
}
func (m *MockAuthUsecase) Login(ctx context.Context, phone, password string) (*usecase.TokenPair, error) {
	args := m.Called(ctx, phone, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.TokenPair), args.Error(1)
}
func (m *MockAuthUsecase) Refresh(ctx context.Context, refreshToken string) (*usecase.TokenPair, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.TokenPair), args.Error(1)
}
func (m *MockAuthUsecase) Logout(ctx context.Context, jti string) error {
	return m.Called(ctx, jti).Error(0)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func jsonBody(t *testing.T, v any) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("jsonBody marshal: %v", err)
	}
	return bytes.NewReader(b)
}

func newRequest(t *testing.T, method, path string, body *bytes.Reader) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func newRouter(h *handler.AuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/auth/register", h.Register)
	r.POST("/auth/send-otp", h.SendOTP)
	r.POST("/auth/verify-otp", h.VerifyOTP)
	r.POST("/auth/login", h.Login)
	r.POST("/auth/refresh", h.Refresh)
	r.POST("/auth/logout", h.Logout)
	return r
}

// ─── Register ────────────────────────────────────────────────────────────────

func TestRegister_Success_Returns201(t *testing.T) {
	uc := &MockAuthUsecase{}
	uc.On("Register", mock.Anything, "+919876543210", "alice@example.com", "password123").
		Return(nil)

	h := handler.NewAuthHandler(uc)
	w := httptest.NewRecorder()
	newRouter(h).ServeHTTP(w, newRequest(t, http.MethodPost, "/auth/register", jsonBody(t, map[string]string{
		"phone": "+919876543210", "email": "alice@example.com", "password": "password123",
	})))

	assert.Equal(t, http.StatusCreated, w.Code)
	uc.AssertExpectations(t)
}

func TestRegister_DuplicateUser_Returns409(t *testing.T) {
	uc := &MockAuthUsecase{}
	uc.On("Register", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(apperrors.ErrUserAlreadyExists)

	h := handler.NewAuthHandler(uc)
	w := httptest.NewRecorder()
	newRouter(h).ServeHTTP(w, newRequest(t, http.MethodPost, "/auth/register", jsonBody(t, map[string]string{
		"phone": "+919876543210", "email": "alice@example.com", "password": "password123",
	})))

	assert.Equal(t, http.StatusConflict, w.Code)
	uc.AssertExpectations(t)
}

func TestRegister_InvalidJSON_Returns400(t *testing.T) {
	uc := &MockAuthUsecase{}
	h := handler.NewAuthHandler(uc)
	w := httptest.NewRecorder()

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	newRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	uc.AssertNotCalled(t, "Register")
}

func TestRegister_MissingField_Returns400(t *testing.T) {
	uc := &MockAuthUsecase{}
	h := handler.NewAuthHandler(uc)
	w := httptest.NewRecorder()

	// email is missing — binding:"required" should reject
	newRouter(h).ServeHTTP(w, newRequest(t, http.MethodPost, "/auth/register", jsonBody(t, map[string]string{
		"phone": "+919876543210", "password": "password123",
	})))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	uc.AssertNotCalled(t, "Register")
}

// ─── SendOTP ─────────────────────────────────────────────────────────────────

func TestSendOTP_Success_Returns200(t *testing.T) {
	uc := &MockAuthUsecase{}
	uc.On("SendOTP", mock.Anything, "+919876543210").Return(nil)

	h := handler.NewAuthHandler(uc)
	w := httptest.NewRecorder()
	newRouter(h).ServeHTTP(w, newRequest(t, http.MethodPost, "/auth/send-otp", jsonBody(t, map[string]string{
		"phone": "+919876543210",
	})))

	assert.Equal(t, http.StatusOK, w.Code)
	uc.AssertExpectations(t)
}

func TestSendOTP_UserNotFound_Returns404(t *testing.T) {
	uc := &MockAuthUsecase{}
	uc.On("SendOTP", mock.Anything, mock.Anything).Return(apperrors.ErrUserNotFound)

	h := handler.NewAuthHandler(uc)
	w := httptest.NewRecorder()
	newRouter(h).ServeHTTP(w, newRequest(t, http.MethodPost, "/auth/send-otp", jsonBody(t, map[string]string{
		"phone": "+919876543210",
	})))

	assert.Equal(t, http.StatusNotFound, w.Code)
	uc.AssertExpectations(t)
}

func TestSendOTP_UserAlreadyActive_Returns400(t *testing.T) {
	uc := &MockAuthUsecase{}
	uc.On("SendOTP", mock.Anything, mock.Anything).Return(apperrors.ErrUserAlreadyActive)

	h := handler.NewAuthHandler(uc)
	w := httptest.NewRecorder()
	newRouter(h).ServeHTTP(w, newRequest(t, http.MethodPost, "/auth/send-otp", jsonBody(t, map[string]string{
		"phone": "+919876543210",
	})))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	uc.AssertExpectations(t)
}

func TestSendOTP_MissingPhone_Returns400(t *testing.T) {
	uc := &MockAuthUsecase{}
	h := handler.NewAuthHandler(uc)
	w := httptest.NewRecorder()

	newRouter(h).ServeHTTP(w, newRequest(t, http.MethodPost, "/auth/send-otp", jsonBody(t, map[string]string{})))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	uc.AssertNotCalled(t, "SendOTP")
}

// ─── VerifyOTP ───────────────────────────────────────────────────────────────

func TestVerifyOTP_Success_Returns200(t *testing.T) {
	uc := &MockAuthUsecase{}
	uc.On("VerifyOTP", mock.Anything, "+919876543210", "123456").Return(nil)

	h := handler.NewAuthHandler(uc)
	w := httptest.NewRecorder()
	newRouter(h).ServeHTTP(w, newRequest(t, http.MethodPost, "/auth/verify-otp", jsonBody(t, map[string]string{
		"phone": "+919876543210", "otp": "123456",
	})))

	assert.Equal(t, http.StatusOK, w.Code)
	uc.AssertExpectations(t)
}

func TestVerifyOTP_OTPExpired_Returns410(t *testing.T) {
	uc := &MockAuthUsecase{}
	uc.On("VerifyOTP", mock.Anything, mock.Anything, mock.Anything).Return(apperrors.ErrOTPExpired)

	h := handler.NewAuthHandler(uc)
	w := httptest.NewRecorder()
	newRouter(h).ServeHTTP(w, newRequest(t, http.MethodPost, "/auth/verify-otp", jsonBody(t, map[string]string{
		"phone": "+919876543210", "otp": "000000",
	})))

	assert.Equal(t, http.StatusGone, w.Code)
	uc.AssertExpectations(t)
}

func TestVerifyOTP_OTPInvalid_Returns400(t *testing.T) {
	uc := &MockAuthUsecase{}
	uc.On("VerifyOTP", mock.Anything, mock.Anything, mock.Anything).Return(apperrors.ErrOTPInvalid)

	h := handler.NewAuthHandler(uc)
	w := httptest.NewRecorder()
	newRouter(h).ServeHTTP(w, newRequest(t, http.MethodPost, "/auth/verify-otp", jsonBody(t, map[string]string{
		"phone": "+919876543210", "otp": "999999",
	})))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	uc.AssertExpectations(t)
}

func TestVerifyOTP_MissingFields_Returns400(t *testing.T) {
	uc := &MockAuthUsecase{}
	h := handler.NewAuthHandler(uc)
	w := httptest.NewRecorder()

	// otp field missing
	newRouter(h).ServeHTTP(w, newRequest(t, http.MethodPost, "/auth/verify-otp", jsonBody(t, map[string]string{
		"phone": "+919876543210",
	})))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	uc.AssertNotCalled(t, "VerifyOTP")
}

// ─── Login ───────────────────────────────────────────────────────────────────

func TestLogin_Success_Returns200WithTokens(t *testing.T) {
	uc := &MockAuthUsecase{}
	uc.On("Login", mock.Anything, "+919876543210", "password123").
		Return(&usecase.TokenPair{AccessToken: "access.tok", RefreshToken: "refresh.tok"}, nil)

	h := handler.NewAuthHandler(uc)
	w := httptest.NewRecorder()
	newRouter(h).ServeHTTP(w, newRequest(t, http.MethodPost, "/auth/login", jsonBody(t, map[string]string{
		"phone": "+919876543210", "password": "password123",
	})))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "access_token")
	assert.Contains(t, w.Body.String(), "refresh_token")
	uc.AssertExpectations(t)
}

func TestLogin_InvalidCredentials_Returns401(t *testing.T) {
	uc := &MockAuthUsecase{}
	uc.On("Login", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, apperrors.ErrInvalidCredentials)

	h := handler.NewAuthHandler(uc)
	w := httptest.NewRecorder()
	newRouter(h).ServeHTTP(w, newRequest(t, http.MethodPost, "/auth/login", jsonBody(t, map[string]string{
		"phone": "+919876543210", "password": "wrongpass",
	})))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	uc.AssertExpectations(t)
}

func TestLogin_UserNotVerified_Returns403(t *testing.T) {
	uc := &MockAuthUsecase{}
	uc.On("Login", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, apperrors.ErrUserNotVerified)

	h := handler.NewAuthHandler(uc)
	w := httptest.NewRecorder()
	newRouter(h).ServeHTTP(w, newRequest(t, http.MethodPost, "/auth/login", jsonBody(t, map[string]string{
		"phone": "+919876543210", "password": "password123",
	})))

	assert.Equal(t, http.StatusForbidden, w.Code)
	uc.AssertExpectations(t)
}

func TestLogin_MissingFields_Returns400(t *testing.T) {
	uc := &MockAuthUsecase{}
	h := handler.NewAuthHandler(uc)
	w := httptest.NewRecorder()

	newRouter(h).ServeHTTP(w, newRequest(t, http.MethodPost, "/auth/login", jsonBody(t, map[string]string{
		"phone": "+919876543210",
	})))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	uc.AssertNotCalled(t, "Login")
}

// ─── Refresh ─────────────────────────────────────────────────────────────────

func TestRefresh_Success_Returns200WithTokens(t *testing.T) {
	uc := &MockAuthUsecase{}
	uc.On("Refresh", mock.Anything, "old.refresh.tok").
		Return(&usecase.TokenPair{AccessToken: "new.access", RefreshToken: "new.refresh"}, nil)

	h := handler.NewAuthHandler(uc)
	w := httptest.NewRecorder()
	newRouter(h).ServeHTTP(w, newRequest(t, http.MethodPost, "/auth/refresh", jsonBody(t, map[string]string{
		"refresh_token": "old.refresh.tok",
	})))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "access_token")
	assert.Contains(t, w.Body.String(), "refresh_token")
	uc.AssertExpectations(t)
}

func TestRefresh_InvalidToken_Returns401(t *testing.T) {
	uc := &MockAuthUsecase{}
	uc.On("Refresh", mock.Anything, mock.Anything).Return(nil, apperrors.ErrInvalidToken)

	h := handler.NewAuthHandler(uc)
	w := httptest.NewRecorder()
	newRouter(h).ServeHTTP(w, newRequest(t, http.MethodPost, "/auth/refresh", jsonBody(t, map[string]string{
		"refresh_token": "bad.token",
	})))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	uc.AssertExpectations(t)
}

func TestRefresh_TokenRevoked_Returns401(t *testing.T) {
	uc := &MockAuthUsecase{}
	uc.On("Refresh", mock.Anything, mock.Anything).Return(nil, apperrors.ErrTokenRevoked)

	h := handler.NewAuthHandler(uc)
	w := httptest.NewRecorder()
	newRouter(h).ServeHTTP(w, newRequest(t, http.MethodPost, "/auth/refresh", jsonBody(t, map[string]string{
		"refresh_token": "revoked.token",
	})))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	uc.AssertExpectations(t)
}

func TestRefresh_MissingToken_Returns400(t *testing.T) {
	uc := &MockAuthUsecase{}
	h := handler.NewAuthHandler(uc)
	w := httptest.NewRecorder()

	newRouter(h).ServeHTTP(w, newRequest(t, http.MethodPost, "/auth/refresh", jsonBody(t, map[string]string{})))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	uc.AssertNotCalled(t, "Refresh")
}

// ─── Logout ──────────────────────────────────────────────────────────────────

func TestLogout_Success_Returns200(t *testing.T) {
	uc := &MockAuthUsecase{}
	uc.On("Logout", mock.Anything, "raw.access.token").Return(nil)

	h := handler.NewAuthHandler(uc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer raw.access.token")
	newRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	uc.AssertExpectations(t)
}

func TestLogout_MissingAuthHeader_Returns401(t *testing.T) {
	uc := &MockAuthUsecase{}
	h := handler.NewAuthHandler(uc)
	w := httptest.NewRecorder()

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	// No Authorization header
	newRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	uc.AssertNotCalled(t, "Logout")
}

func TestLogout_InvalidToken_Returns401(t *testing.T) {
	uc := &MockAuthUsecase{}
	uc.On("Logout", mock.Anything, mock.Anything).Return(apperrors.ErrInvalidToken)

	h := handler.NewAuthHandler(uc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer bad.token.here")
	newRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	uc.AssertExpectations(t)
}
