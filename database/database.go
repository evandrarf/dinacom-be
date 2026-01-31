package database

import (
	"fmt"

	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func New(config *viper.Viper) *gorm.DB {
	username := config.GetString("database.username")
	password := config.GetString("database.password")
	host := config.GetString("database.host")
	port := config.GetInt("database.port")
	dbname := config.GetString("database.dbname")
	sslmode := config.GetString("database.sslmode")
	if sslmode == "" {
		sslmode = "disable"
	}
	timezone := config.GetString("database.timezone")
	if timezone == "" {
		timezone = "UTC"
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		host,
		username,
		password,
		dbname,
		port,
		sslmode,
		timezone,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		panic(fmt.Errorf("failed to connect database: %w", err))
	}

	return db
}
