package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func (m *Middleware) CorsMiddleware() fiber.Handler {
	allowOrigins := "*"
	if m != nil && m.Config != nil {
		if v := m.Config.GetString("api.cors.origins"); v != "" {
			allowOrigins = v
		}
	}

	return cors.New(cors.Config{
		AllowHeaders:  "Origin, Content-Type, Accept, Authorization, Content-Length, Accept-Encoding",
		AllowMethods:  "GET, POST, PUT, PATCH, DELETE",
		AllowOrigins:  allowOrigins,
		ExposeHeaders: "Content-Length, Content-Type",
	})
}
