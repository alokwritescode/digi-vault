package jwt_test

import (
	"strings"
	"testing"
	"time"

	"github.com/alokwritescode/digi-vault/shared/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testAccessSecret  = "test-access-secret-32-bytes-long!!"
	testRefreshSecret = "test-refresh-secret-32-bytes-longg"
)

func TestGenerateAccessToken_ReturnsToken(t *testing.T) {
	token, err := jwt.GenerateAccessToken(42, testAccessSecret, 15*time.Minute)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, 3, len(strings.Split(token, ".")), "should be a 3-part JWT")
}

func TestGenerateRefreshToken_ReturnsToken(t *testing.T) {
	token, err := jwt.GenerateRefreshToken(99, testRefreshSecret, 7*24*time.Hour)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, 3, len(strings.Split(token, ".")), "should be a 3-part JWT")
}

func TestGenerateAccessToken_ClaimsRoundTrip(t *testing.T) {
	token, err := jwt.GenerateAccessToken(77, testAccessSecret, 15*time.Minute)
	require.NoError(t, err)

	claims, err := jwt.ParseToken(token, testAccessSecret)
	require.NoError(t, err)

	assert.Equal(t, uint64(77), claims.UserID)
	assert.NotEmpty(t, claims.JTI, "jti must be present")
}

func TestGenerateRefreshToken_ClaimsRoundTrip(t *testing.T) {
	token, err := jwt.GenerateRefreshToken(55, testRefreshSecret, 7*24*time.Hour)
	require.NoError(t, err)

	claims, err := jwt.ParseToken(token, testRefreshSecret)
	require.NoError(t, err)

	assert.Equal(t, uint64(55), claims.UserID)
	assert.NotEmpty(t, claims.JTI)
}

func TestGenerateAccessToken_UniqueJTI(t *testing.T) {
	t1, err1 := jwt.GenerateAccessToken(1, testAccessSecret, time.Minute)
	t2, err2 := jwt.GenerateAccessToken(1, testAccessSecret, time.Minute)
	require.NoError(t, err1)
	require.NoError(t, err2)

	c1, _ := jwt.ParseToken(t1, testAccessSecret)
	c2, _ := jwt.ParseToken(t2, testAccessSecret)
	assert.NotEqual(t, c1.JTI, c2.JTI, "each token must have a unique jti")
}

func TestParseToken_WrongSecret_Fails(t *testing.T) {
	token, err := jwt.GenerateAccessToken(1, testAccessSecret, time.Minute)
	require.NoError(t, err)

	_, err = jwt.ParseToken(token, "wrong-secret")
	assert.Error(t, err)
}

func TestParseToken_ExpiredToken_Fails(t *testing.T) {
	token, err := jwt.GenerateAccessToken(1, testAccessSecret, -time.Second)
	require.NoError(t, err)

	_, err = jwt.ParseToken(token, testAccessSecret)
	assert.Error(t, err)
}

func TestParseToken_MalformedToken_Fails(t *testing.T) {
	_, err := jwt.ParseToken("not.a.valid.jwt", testAccessSecret)
	assert.Error(t, err)
}

func TestParseToken_AlgorithmSwitch_Fails(t *testing.T) {
	// A JWT signed with HS256 verified against a "none" alg claim should fail.
	// This is handled by the signing method check in the keyfunc.
	noneAlgToken := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJ1c2VyX2lkIjo5OX0."
	_, err := jwt.ParseToken(noneAlgToken, testAccessSecret)
	assert.Error(t, err, "algorithm switch attack should be rejected")
}

func TestExtractBearerToken_Valid(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{"standard bearer", "Bearer mytoken123", "mytoken123"},
		{"real JWT", "Bearer eyJ.eyJ.sig", "eyJ.eyJ.sig"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := jwt.ExtractBearerToken(tt.header)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractBearerToken_Invalid(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{"empty header", ""},
		{"no bearer prefix", "mytoken123"},
		{"only Bearer", "Bearer "},
		{"wrong scheme", "Basic mytoken"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := jwt.ExtractBearerToken(tt.header)
			assert.Error(t, err)
		})
	}
}
