package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"paper/purgatory/configuration"
)

func main() {
	config := configuration.LoadConfig()
	container := configuration.InitContainer(config)

	router := gin.Default()
	router.Use(configuration.AuthMiddleware(config.Sign.Key, container.Database))

	router.GET("/purgatory", container.PurgatoryController.Get)

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
