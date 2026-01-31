package config

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func NewLogger(viper *viper.Viper) *logrus.Logger {
	log := logrus.New()

	log.SetLevel(logrus.Level(viper.GetInt32("log.level")))
	log.SetOutput(os.Stdout)

	format := viper.GetString("log.format")
	if format == "json" {
		log.SetFormatter(&logrus.JSONFormatter{})
	} else if format == "text" {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	} else {
		panic("invalid log format, please specify either `json` or `text`")
	}

	return log
}
