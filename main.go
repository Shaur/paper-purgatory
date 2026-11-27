package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"paper/purgatory/configuration"
	"paper/purgatory/worker"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	MiB = 1 << 20 // 1048576
	GiB = 1 << 30
)

const maxUploadSize = 2 * GiB

func limitBody(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

func setupRouter(container *configuration.Container, config *configuration.Config) *gin.Engine {
	router := gin.Default()
	router.Use(configuration.CORSMiddleware())
	router.Use(configuration.AuthMiddleware(config.Sign.Key, container.Database))
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{SkipPaths: []string{"/actuator"}}))

	router.GET("/purgatory", container.PurgatoryController.Get)
	router.POST("/purgatory/meta", container.PurgatoryController.AddMeta)
	router.POST("/purgatory", limitBody(maxUploadSize), container.PurgatoryController.UploadFile)

	actuatorGroup := router.Group("/actuator")
	{
		actuatorGroup.GET("/health", func(context *gin.Context) {
			status := struct{ Status string }{Status: "Up"}

			context.JSON(http.StatusOK, status)
		})
	}

	return router
}

func main() {
	config := configuration.LoadConfig()
	container := configuration.InitContainer(config)

	router := setupRouter(&container, config)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	w := worker.NewWorker(container.Database, container.PurgatoryService, 10*time.Second)
	go w.Run(ctx)
	go w.RunGC(ctx)

	err := router.Run(":8080")
	if err != nil {
		fmt.Println("Failed to start server:", err)
	}
}
