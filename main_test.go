package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"paper/purgatory/configuration"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	postgresContainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	Username string `gorm:"primaryKey"`
}

func (User) TableName() string {
	return "user_data"
}

type AuthMiddlewareTestSuite struct {
	suite.Suite
	pgContainer testcontainers.Container
	db          *gorm.DB
	signingKey  string
}

func (s *AuthMiddlewareTestSuite) SetupSuite() {
	ctx := context.Background()

	var err error
	s.pgContainer, err = postgresContainer.Run(
		ctx,
		"postgres:16-alpine",
		postgresContainer.WithDatabase("testdb"),
		postgresContainer.WithUsername("testuser"),
		postgresContainer.WithPassword("testpassword"),
		postgresContainer.BasicWaitStrategies(),
	)

	s.Require().NoError(err, "Failed to start PostgreSQL container")

	mappedPort, err := s.pgContainer.MappedPort(ctx, "5432")
	s.Require().NoError(err, "Failed to get mapped port")

	host, err := s.pgContainer.Host(ctx)
	s.Require().NoError(err, "Failed to get host")

	dsn := fmt.Sprintf("host=%s user=testuser password=testpassword dbname=testdb port=%s sslmode=disable",
		host, mappedPort.Port())

	s.db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	s.Require().NoError(err, "Failed to connect to database")

	err = s.db.AutoMigrate(&User{})
	s.Require().NoError(err, "Failed to migrate database schema")

	s.signingKey = "07NGeiQj5vJbnrLKZzukZK8gYQamCA54xx0VAdnhlZqm6xfkwS2Z9rhRm3sOdr0C"
}

func (s *AuthMiddlewareTestSuite) TearDownSuite() {
	ctx := context.Background()
	s.Require().NoError(s.pgContainer.Terminate(ctx), "Failed to terminate container")
}

func (s *AuthMiddlewareTestSuite) SetupTest() {
	s.db.Exec("TRUNCATE TABLE user_data")

	testUser := &User{Username: "testuser"}
	s.db.Create(testUser)
}

func (s *AuthMiddlewareTestSuite) createTestToken(username string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.MapClaims{
		"sub": username,
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	data, err := base64.StdEncoding.DecodeString(s.signingKey)
	if err != nil {
		os.Exit(-1)
	}

	tokenString, _ := token.SignedString(data)
	return tokenString
}

func createTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	return c, w
}

func (s *AuthMiddlewareTestSuite) TestValidTokenWithExistingUser() {
	ctx, httpRecorder := createTestContext()

	token := s.createTestToken("testuser")
	ctx.Request = httptest.NewRequest("GET", "/", nil)
	ctx.Request.Header.Set("Authorization", "Bearer "+token)

	middleware := configuration.AuthMiddleware(s.signingKey, s.db)
	middleware(ctx)

	s.Assert().Equal(http.StatusOK, httpRecorder.Code)
	s.Assert().False(ctx.IsAborted())
}

func (s *AuthMiddlewareTestSuite) TestMissingAuthorizationHeader() {
	ctx, httpRecorder := createTestContext()
	ctx.Request = httptest.NewRequest("GET", "/", nil)

	middleware := configuration.AuthMiddleware(s.signingKey, s.db)
	middleware(ctx)

	s.Assert().Equal(http.StatusUnauthorized, httpRecorder.Code)
	s.Assert().True(ctx.IsAborted())
}

func (s *AuthMiddlewareTestSuite) TestEmptyToken() {
	ctx, httpRecorder := createTestContext()
	ctx.Request = httptest.NewRequest("GET", "/", nil)
	ctx.Request.Header.Set("Authorization", "Bearer ")

	middleware := configuration.AuthMiddleware(s.signingKey, s.db)
	middleware(ctx)

	s.Assert().Equal(http.StatusUnauthorized, httpRecorder.Code)
	s.Assert().True(ctx.IsAborted())
}

func (s *AuthMiddlewareTestSuite) TestInvalidToken() {
	ctx, httpRecorder := createTestContext()
	ctx.Request = httptest.NewRequest("GET", "/", nil)
	ctx.Request.Header.Set("Authorization", "Bearer invalid.token.here")

	middleware := configuration.AuthMiddleware(s.signingKey, s.db)
	middleware(ctx)

	s.Assert().Equal(http.StatusUnauthorized, httpRecorder.Code)
	s.Assert().True(ctx.IsAborted())
}

func (s *AuthMiddlewareTestSuite) TestValidTokenWithNonExistingUser() {
	ctx, httpRecorder := createTestContext()
	token := s.createTestToken("nonexistentuser")
	ctx.Request = httptest.NewRequest("GET", "/", nil)
	ctx.Request.Header.Set("Authorization", "Bearer "+token)

	middleware := configuration.AuthMiddleware(s.signingKey, s.db)
	middleware(ctx)

	s.Assert().Equal(http.StatusUnauthorized, httpRecorder.Code)
	s.Assert().True(ctx.IsAborted())
}

func (s *AuthMiddlewareTestSuite) TestTokenWithWrongSigningKey() {
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

func (s *AuthMiddlewareTestSuite) TestExpiredToken() {
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

func (s *AuthMiddlewareTestSuite) TestTokenWithoutSubject() {
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
	suite.Run(t, new(AuthMiddlewareTestSuite))
}
