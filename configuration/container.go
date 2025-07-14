package configuration

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
	"paper/purgatory/controller"
	"paper/purgatory/service"
)

type Container struct {
	Database            *gorm.DB
	PurgatoryService    service.PurgatoryService
	PurgatoryController controller.PurgatoryController
}

func InitContainer(config *Config) Container {
	database := initDatabase(config.Postgres)
	purgatoryService := service.Init(database)
	purgatoryController := controller.Init(purgatoryService)

	return Container{
		Database:            database,
		PurgatoryService:    purgatoryService,
		PurgatoryController: purgatoryController,
	}
}

func initDatabase(config Postgres) *gorm.DB {
	database, err := gorm.Open(postgres.Open(config.Dsn()), &gorm.Config{})
	if err != nil {
		fmt.Println("Failed to connect to postgres:", err)
		os.Exit(1)
	}
	return database
}
