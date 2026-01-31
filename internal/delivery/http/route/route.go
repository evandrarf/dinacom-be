package route

import (
	"github.com/evandrarf/dinacom-be/internal/delivery/http/handler"
	"github.com/evandrarf/dinacom-be/internal/delivery/http/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

type RouteConfig struct {
	Api                     *fiber.App
	Middleware              *middleware.Middleware
	DyslexiaQuestionHandler handler.DyslexiaQuestionHandler
}

func Setup(c *RouteConfig) {
	c.Api.Use(recover.New())
	c.Api.Use(logger.New(logger.Config{
		Format: "[${ip}]:${port} ${status} - ${method} ${path}\n",
	}))
	c.Api.Use(c.Middleware.CorsMiddleware())

	SetupDyslexiaQuestionRoute(c.Api, c.DyslexiaQuestionHandler, c.Middleware)
}
