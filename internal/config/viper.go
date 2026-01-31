package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

func NewViper() *viper.Viper {
	config := viper.New()

	if os.Getenv("ENV") == "production" {
		config.SetConfigName("config.prod")
	} else {
		config.SetConfigName("config")
	}

	config.SetConfigType("yaml")
	config.AddConfigPath(".")

	if err := config.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	return config
}
