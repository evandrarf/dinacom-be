package route

import (
	"github.com/evandrarf/dinacom-be/internal/delivery/http/handler"
	"github.com/evandrarf/dinacom-be/internal/delivery/http/middleware"
	"github.com/gofiber/fiber/v2"
)

func SetupDyslexiaQuestionRoute(api *fiber.App, handler handler.DyslexiaQuestionHandler, m *middleware.Middleware) {
	router := api.Group("/questions")
	{
		router.Get("/generate", handler.Generate)
		router.Post("/answer", handler.SubmitAnswer)
		router.Get("/sessions/:session_id", handler.GetSessionAnswers)
	}

	reportRouter := api.Group("/report")
	{
		reportRouter.Get("/sessions/:session_id", handler.GetSessionReport)
	}

	chatbotRouter := api.Group("/chatbot")
	{
		chatbotRouter.Post("/sessions/:session_id", handler.ChatWithBot)
		chatbotRouter.Get("/sessions/:session_id/history", handler.GetChatHistory)
	}
}
