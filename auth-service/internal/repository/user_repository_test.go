package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/alokwritescode/digi-vault/auth-service/internal/domain"
	"github.com/alokwritescode/digi-vault/auth-service/internal/repository"
	apperrors "github.com/alokwritescode/digi-vault/shared/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestUserDB wires a sqlmock SQL driver into GORM.
//
// SkipInitializeWithVersion — prevents GORM from running SELECT VERSION() on open.
// SkipDefaultTransaction    — prevents auto BEGIN/COMMIT around writes so we mock
//                             one Exec per operation instead of three.
func newTestUserDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return gormDB, mock
}

var userCols = []string{
	"id", "phone", "email", "password", "is_active",
	"created_at", "updated_at", "deleted_at",
}

func userRow(u domain.User) *sqlmock.Rows {
	return sqlmock.NewRows(userCols).AddRow(
		int64(u.ID), u.Phone, u.Email, u.Password, u.IsActive,
		u.CreatedAt, u.UpdatedAt, u.DeletedAt,
	)
}

// ─── Create ──────────────────────────────────────────────────────────────────

func TestCreate_Success(t *testing.T) {
	gormDB, mock := newTestUserDB(t)
	repo := repository.NewUserRepository(gormDB)

	mock.ExpectExec(`INSERT INTO .users.`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), &domain.User{
		Phone:    "+919876543210",
		Email:    "alice@example.com",
		Password: "hashed",
	})
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreate_DuplicateEntry_ReturnsErrUserAlreadyExists(t *testing.T) {
	gormDB, mock := newTestUserDB(t)
	repo := repository.NewUserRepository(gormDB)

	mock.ExpectExec(`INSERT INTO .users.`).
		WillReturnError(errors.New("Error 1062: Duplicate entry '+91...' for key 'idx_users_phone'"))

	err := repo.Create(context.Background(), &domain.User{Phone: "+919876543210"})
	assert.ErrorIs(t, err, apperrors.ErrUserAlreadyExists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ─── FindByPhone ─────────────────────────────────────────────────────────────

func TestFindByPhone_Found(t *testing.T) {
	gormDB, mock := newTestUserDB(t)
	repo := repository.NewUserRepository(gormDB)

	expected := domain.User{
		ID: 1, Phone: "+919876543210", Email: "alice@example.com",
		Password: "hashed", IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	mock.ExpectQuery(`SELECT .+ FROM .users.`).
		WillReturnRows(userRow(expected))

	got, err := repo.FindByPhone(context.Background(), "+919876543210")
	require.NoError(t, err)
	assert.Equal(t, expected.ID, got.ID)
	assert.Equal(t, expected.Phone, got.Phone)
	assert.Equal(t, expected.Email, got.Email)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByPhone_NotFound_ReturnsErrUserNotFound(t *testing.T) {
	gormDB, mock := newTestUserDB(t)
	repo := repository.NewUserRepository(gormDB)

	mock.ExpectQuery(`SELECT .+ FROM .users.`).
		WillReturnRows(sqlmock.NewRows(userCols))

	_, err := repo.FindByPhone(context.Background(), "+919000000000")
	assert.ErrorIs(t, err, apperrors.ErrUserNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ─── FindByEmail ─────────────────────────────────────────────────────────────

func TestFindByEmail_Found(t *testing.T) {
	gormDB, mock := newTestUserDB(t)
	repo := repository.NewUserRepository(gormDB)

	expected := domain.User{
		ID: 2, Phone: "+910000000001", Email: "bob@example.com",
		Password: "hashed2", IsActive: false,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	mock.ExpectQuery(`SELECT .+ FROM .users.`).
		WillReturnRows(userRow(expected))

	got, err := repo.FindByEmail(context.Background(), "bob@example.com")
	require.NoError(t, err)
	assert.Equal(t, expected.Email, got.Email)
	assert.Equal(t, expected.ID, got.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByEmail_NotFound_ReturnsErrUserNotFound(t *testing.T) {
	gormDB, mock := newTestUserDB(t)
	repo := repository.NewUserRepository(gormDB)

	mock.ExpectQuery(`SELECT .+ FROM .users.`).
		WillReturnRows(sqlmock.NewRows(userCols))

	_, err := repo.FindByEmail(context.Background(), "nobody@example.com")
	assert.ErrorIs(t, err, apperrors.ErrUserNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ─── FindByID ────────────────────────────────────────────────────────────────

func TestFindByID_Found(t *testing.T) {
	gormDB, mock := newTestUserDB(t)
	repo := repository.NewUserRepository(gormDB)

	expected := domain.User{
		ID: 99, Phone: "+910000000099", Email: "carol@example.com",
		Password: "hashed3", IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	mock.ExpectQuery(`SELECT .+ FROM .users.`).
		WillReturnRows(userRow(expected))

	got, err := repo.FindByID(context.Background(), 99)
	require.NoError(t, err)
	assert.Equal(t, expected.ID, got.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByID_NotFound_ReturnsErrUserNotFound(t *testing.T) {
	gormDB, mock := newTestUserDB(t)
	repo := repository.NewUserRepository(gormDB)

	mock.ExpectQuery(`SELECT .+ FROM .users.`).
		WillReturnRows(sqlmock.NewRows(userCols))

	_, err := repo.FindByID(context.Background(), 404)
	assert.ErrorIs(t, err, apperrors.ErrUserNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ─── SoftDelete ──────────────────────────────────────────────────────────────

func TestSoftDelete_Success(t *testing.T) {
	gormDB, mock := newTestUserDB(t)
	repo := repository.NewUserRepository(gormDB)

	// GORM soft delete issues UPDATE users SET deleted_at=? WHERE id=?
	mock.ExpectExec(`UPDATE .users.`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.SoftDelete(context.Background(), 1)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSoftDelete_DBError(t *testing.T) {
	gormDB, mock := newTestUserDB(t)
	repo := repository.NewUserRepository(gormDB)

	mock.ExpectExec(`UPDATE .users.`).
		WillReturnError(errors.New("connection refused"))

	err := repo.SoftDelete(context.Background(), 1)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, apperrors.ErrUserNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ─── UpdateIsActive ───────────────────────────────────────────────────────────

func TestUpdateIsActive_Success(t *testing.T) {
	gormDB, mock := newTestUserDB(t)
	repo := repository.NewUserRepository(gormDB)

	mock.ExpectExec(`UPDATE .users.`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateIsActive(context.Background(), 1, true)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateIsActive_DBError(t *testing.T) {
	gormDB, mock := newTestUserDB(t)
	repo := repository.NewUserRepository(gormDB)

	mock.ExpectExec(`UPDATE .users.`).
		WillReturnError(errors.New("connection refused"))

	err := repo.UpdateIsActive(context.Background(), 1, true)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, apperrors.ErrUserNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}
