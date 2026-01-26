package main

import (
	"fmt"
	"net/http"
	"paper/purgatory/configuration"

	"github.com/gin-gonic/gin"
)

func main() {
	config := configuration.LoadConfig()
	container := configuration.InitContainer(config)

	router := gin.Default()
	router.Use(configuration.CORSMiddleware())
	router.Use(configuration.AuthMiddleware(config.Sign.Key, container.Database))
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{SkipPaths: []string{"/actuator"}}))

	router.GET("/purgatory", container.PurgatoryController.Get)
	router.POST("/purgatory", container.PurgatoryController.UploadFile)
	router.POST("/purgatory/meta", container.PurgatoryController.AddMeta)

	actuatorGroup := router.Group("/actuator")
	{
		actuatorGroup.GET("/health", func(context *gin.Context) {
			status := struct{ Status string }{Status: "Up"}

			context.JSON(http.StatusOK, status)
		})
	}

	err := router.Run(":8080")
	if err != nil {
		fmt.Println("Failed to start server:", err)
	}
}
