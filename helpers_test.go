package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http/httptest"
	"os"
	"paper/purgatory/controller"
	"paper/purgatory/model"
	"paper/purgatory/service"
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

type MainTestSuite struct {
	suite.Suite
	pgContainer testcontainers.Container
	db          *gorm.DB
	signingKey  string
	filesPath   string
	service     service.PurgatoryService
	controller  controller.PurgatoryController
}

func (s *MainTestSuite) SetupSuite() {
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

	err = s.db.AutoMigrate(&model.PurgatoryItem{})
	s.Require().NoError(err, "Failed to migrate database schema")

	s.filesPath = s.T().TempDir()

	s.service = service.Init(s.db, s.filesPath)

	s.controller = controller.Init(s.service)

	s.signingKey = "07NGeiQj5vJbnrLKZzukZK8gYQamCA54xx0VAdnhlZqm6xfkwS2Z9rhRm3sOdr0C"
}

func (s *MainTestSuite) TearDownSuite() {
	ctx := context.Background()
	s.Require().NoError(s.pgContainer.Terminate(ctx), "Failed to terminate container")
}

func (s *MainTestSuite) SetupTest() {
	s.db.Exec("TRUNCATE TABLE user_data")

	testUser := &User{Username: "testuser"}
	s.db.Create(testUser)
}

func createTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	return c, w
}

func (s *MainTestSuite) createTestToken(username string) string {
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
