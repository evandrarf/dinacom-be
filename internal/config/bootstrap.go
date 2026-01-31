package config

import (
	"github.com/evandrarf/dinacom-be/internal/delivery/http/handler"
	"github.com/evandrarf/dinacom-be/internal/delivery/http/middleware"
	"github.com/evandrarf/dinacom-be/internal/delivery/http/repository"
	"github.com/evandrarf/dinacom-be/internal/delivery/http/route"
	"github.com/evandrarf/dinacom-be/internal/delivery/http/usecase"
	"github.com/evandrarf/dinacom-be/internal/pkg/llm"
	"github.com/evandrarf/dinacom-be/internal/pkg/validate"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type BootstrapConfig struct {
	Api       *fiber.App
	Config    *viper.Viper
	DB        *gorm.DB
	Log       *logrus.Logger
	Validator *validate.Validator
}

func Bootstrap(config *BootstrapConfig) {

	mid := middleware.NewMiddleware(&middleware.MiddlewareConfig{
		Log:    config.Log,
		Config: config.Config,
	})

	apiKey := ""
	model := ""
	baseURL := ""
	promptTemplate := ""
	if config.Config != nil {
		apiKey = config.Config.GetString("llm.gemini.api_key")
		model = config.Config.GetString("llm.gemini.model")
		baseURL = config.Config.GetString("llm.gemini.base_url")
		promptTemplate = config.Config.GetString("llm.gemini.prompt_template")
	}

	gemini := llm.NewGeminiClient(apiKey, model, baseURL)
	dyslexiaQuestionRepo := repository.NewDyslexiaQuestionRepository(config.DB)
	dyslexiaQuestionUsecase := usecase.NewDyslexiaQuestionUsecase(usecase.DyslexiaQuestionConfig{
		DB:             config.DB,
		Gemini:         gemini,
		PromptTemplate: promptTemplate,
		Repository:     dyslexiaQuestionRepo,
		Config:         config.Config,
	})
	dyslexiaQuestionHandler := handler.NewDyslexiaQuestionHandler(config.Validator, config.Log, dyslexiaQuestionUsecase)

	route.Setup(&route.RouteConfig{
		Api:                     config.Api,
		Middleware:              mid,
		DyslexiaQuestionHandler: dyslexiaQuestionHandler,
	})

}
