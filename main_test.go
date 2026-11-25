package main

import (
	"net/http"
	"net/http/httptest"
	"paper/purgatory/configuration"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/suite"
)

func (s *MainTestSuite) TestValidTokenWithExistingUser() {
	ctx, httpRecorder := createTestContext()

	token := s.createTestToken("testuser")
	ctx.Request = httptest.NewRequest("GET", "/", nil)
	ctx.Request.Header.Set("Authorization", "Bearer "+token)

	middleware := configuration.AuthMiddleware(s.signingKey, s.db)
	middleware(ctx)

	s.Assert().Equal(http.StatusOK, httpRecorder.Code)
	s.Assert().False(ctx.IsAborted())
}

func (s *MainTestSuite) TestMissingAuthorizationHeader() {
	ctx, httpRecorder := createTestContext()
	ctx.Request = httptest.NewRequest("GET", "/", nil)

	middleware := configuration.AuthMiddleware(s.signingKey, s.db)
	middleware(ctx)

	s.Assert().Equal(http.StatusUnauthorized, httpRecorder.Code)
	s.Assert().True(ctx.IsAborted())
}

func (s *MainTestSuite) TestEmptyToken() {
	ctx, httpRecorder := createTestContext()
	ctx.Request = httptest.NewRequest("GET", "/", nil)
	ctx.Request.Header.Set("Authorization", "Bearer ")

	middleware := configuration.AuthMiddleware(s.signingKey, s.db)
	middleware(ctx)

	s.Assert().Equal(http.StatusUnauthorized, httpRecorder.Code)
	s.Assert().True(ctx.IsAborted())
}

func (s *MainTestSuite) TestInvalidToken() {
	ctx, httpRecorder := createTestContext()
	ctx.Request = httptest.NewRequest("GET", "/", nil)
	ctx.Request.Header.Set("Authorization", "Bearer invalid.token.here")

	middleware := configuration.AuthMiddleware(s.signingKey, s.db)
	middleware(ctx)

	s.Assert().Equal(http.StatusUnauthorized, httpRecorder.Code)
	s.Assert().True(ctx.IsAborted())
}

func (s *MainTestSuite) TestValidTokenWithNonExistingUser() {
	ctx, httpRecorder := createTestContext()
	token := s.createTestToken("nonexistentuser")
	ctx.Request = httptest.NewRequest("GET", "/", nil)
	ctx.Request.Header.Set("Authorization", "Bearer "+token)

	middleware := configuration.AuthMiddleware(s.signingKey, s.db)
	middleware(ctx)

	s.Assert().Equal(http.StatusUnauthorized, httpRecorder.Code)
	s.Assert().True(ctx.IsAborted())
}

func (s *MainTestSuite) TestTokenWithWrongSigningKey() {
	ctx, httpRecorder := createTestContext()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "testuser",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte("wrong-key"))

	ctx.Request = httptest.NewRequest("GET", "/", nil)

	ctx.Request.Header.Set("Authorization", "Bearer "+tokenString)

	middleware := configuration.AuthMiddleware(s.signingKey, s.db)
	middleware(ctx)

	s.Assert().Equal(http.StatusUnauthorized, httpRecorder.Code)
	s.Assert().True(ctx.IsAborted())
}

func (s *MainTestSuite) TestExpiredToken() {
	ctx, httpRecorder := createTestContext()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "testuser",
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte(s.signingKey))

	ctx.Request = httptest.NewRequest("GET", "/", nil)
	ctx.Request.Header.Set("Authorization", "Bearer "+tokenString)

	middleware := configuration.AuthMiddleware(s.signingKey, s.db)
	middleware(ctx)

	s.Assert().Equal(http.StatusUnauthorized, httpRecorder.Code)
	s.Assert().True(ctx.IsAborted())
}

func (s *MainTestSuite) TestTokenWithoutSubject() {
	ctx, httpRecorder := createTestContext()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte(s.signingKey))

	ctx.Request = httptest.NewRequest("GET", "/", nil)
	ctx.Request.Header.Set("Authorization", "Bearer "+tokenString)

	middleware := configuration.AuthMiddleware(s.signingKey, s.db)
	middleware(ctx)

	s.Assert().Equal(http.StatusUnauthorized, httpRecorder.Code)
	s.Assert().True(ctx.IsAborted())
}

func TestAuthMiddleware(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	suite.Run(t, new(MainTestSuite))
}
