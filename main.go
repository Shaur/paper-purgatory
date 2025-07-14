package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"paper/purgatory/configuration"
)

func main() {
	config := configuration.LoadConfig()
	container := configuration.InitContainer(config)

	router := gin.Default()
	router.Use(configuration.AuthMiddleware(config.Sign.Key, container.Database))

	group := router.Group("/purgatory")
	{
		group.GET("/", container.PurgatoryController.Get)
	}

	err := router.Run("localhost:8080")
	if err != nil {
		fmt.Println("Failed to start server:", err)
	}
}
