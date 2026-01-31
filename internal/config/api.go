package config

import (
	"github.com/evandrarf/dinacom-be/internal/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func NewAPI(config *viper.Viper, log *logrus.Logger) *fiber.App {
	api := fiber.New(fiber.Config{
		AppName:      config.GetString("app.name"),
		ErrorHandler: ErrorHandler(log),
		Prefork:      config.GetBool("api.prefork"),
	})
	return api
}

func ErrorHandler(log *logrus.Logger) fiber.ErrorHandler {
	return func(ctx *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}

		if code >= 500 {
			log.Error(err)
			return response.NewInternalServerError().Send(ctx)
		}

		return response.NewFailed(err.Error(), fiber.NewError(code, ""), log).Send(ctx)
	}
}
