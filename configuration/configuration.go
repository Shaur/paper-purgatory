package configuration

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"strconv"
)

type Postgres struct {
	Host     string
	Port     int32
	User     string
	Password string
	Database string
}

type Sign struct {
	Key string
}

func (postgres *Postgres) Dsn() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d",
		postgres.Host,
		postgres.User,
		postgres.Password,
		postgres.Database,
		postgres.Port,
	)
}

type Config struct {
	Postgres Postgres
	Sign     Sign
}

func LoadConfig() *Config {
	file, err := os.ReadFile("application.yaml")
	if err != nil {
		os.Exit(1)
	}

	config := &Config{}
	err = yaml.Unmarshal(file, config)
	if err != nil {
		log.Printf("Can't open properties file")
		os.Exit(1)
	}

	enrichPostgresConfig(config)

	return config
}

func enrichPostgresConfig(config *Config) {
	value, isPresent := os.LookupEnv("POSTGRES_HOST")
	if isPresent {
		config.Postgres.Host = value
	}

	value, isPresent = os.LookupEnv("POSTGRES_PORT")
	if isPresent {
		port, err := strconv.ParseInt(value, 10, 32)
		if err == nil {
			config.Postgres.Port = int32(port)
		}
	}

	value, isPresent = os.LookupEnv("POSTGRES_USER")
	if isPresent {
		config.Postgres.User = value
	}

	value, isPresent = os.LookupEnv("POSTGRES_PASSWORD")
	if isPresent {
		config.Postgres.Password = value
	}

	value, isPresent = os.LookupEnv("POSTGRES_DATABASE")
	if isPresent {
		config.Postgres.Database = value
	}
}
