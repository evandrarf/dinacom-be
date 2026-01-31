package middleware

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type MiddlewareConfig struct {
	Log    *logrus.Logger
	Config *viper.Viper
}

type Middleware struct {
	Log    *logrus.Logger
	Config *viper.Viper
}

func NewMiddleware(c *MiddlewareConfig) *Middleware {
	if c == nil {
		return &Middleware{}
	}

	return &Middleware{
		Log:    c.Log,
		Config: c.Config,
	}
}
