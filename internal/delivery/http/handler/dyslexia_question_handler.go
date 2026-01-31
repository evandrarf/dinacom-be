package handler

import (
	"strconv"
	"strings"

	"github.com/evandrarf/dinacom-be/internal/delivery/http/domain"
	"github.com/evandrarf/dinacom-be/internal/delivery/http/entity"
	"github.com/evandrarf/dinacom-be/internal/delivery/http/usecase"
	"github.com/evandrarf/dinacom-be/internal/pkg/response"
	"github.com/evandrarf/dinacom-be/internal/pkg/validate"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

type (
	DyslexiaQuestionHandler interface {
		Generate(ctx *fiber.Ctx) error
		SubmitAnswer(ctx *fiber.Ctx) error
		GetSessionAnswers(ctx *fiber.Ctx) error
		GetSessionReport(ctx *fiber.Ctx) error
	}

	dyslexiaQuestionHandler struct {
		validator *validate.Validator
		logger    *logrus.Logger
		usecase   usecase.DyslexiaQuestionUsecase
	}
)

func NewDyslexiaQuestionHandler(validator *validate.Validator, logger *logrus.Logger, usecase usecase.DyslexiaQuestionUsecase) DyslexiaQuestionHandler {
	return &dyslexiaQuestionHandler{
		validator: validator,
		logger:    logger,
		usecase:   usecase,
	}
}

// GET /questions/generate?difficulty=easy|medium|hard&count=1&includeAnswer=false
func (h *dyslexiaQuestionHandler) Generate(ctx *fiber.Ctx) error {
	_ = h.validator

	count := 1
	if v := strings.TrimSpace(ctx.Query("count")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			count = n
		}
	}

	includeAnswer := false
	if v := strings.TrimSpace(ctx.Query("includeAnswer")); v != "" {
		includeAnswer = (v == "1" || strings.EqualFold(v, "true"))
	}

	difficulty := entity.DifficultyEasy
	if d := strings.TrimSpace(ctx.Query("difficulty")); d != "" {
		difficulty = entity.Difficulty(strings.ToLower(d))
		switch difficulty {
		case entity.DifficultyEasy, entity.DifficultyMedium, entity.DifficultyHard:
			// ok
		default:
			return response.NewFailed(domain.DYSLEXIA_QUESTION_GENERATE_FAILED, fiber.NewError(fiber.StatusBadRequest, "invalid difficulty"), h.logger).Send(ctx)
		}
	}

	questions, err := h.usecase.Generate(ctx.UserContext(), difficulty, count, includeAnswer)
	if err != nil {
		return response.NewFailed(domain.DYSLEXIA_QUESTION_GENERATE_FAILED, fiber.NewError(fiber.StatusBadRequest, err.Error()), h.logger).Send(ctx)
	}

	return response.NewSuccess(domain.DYSLEXIA_QUESTION_GENERATE_SUCCESS, questions, nil).Send(ctx)
}

// POST /questions/answer
func (h *dyslexiaQuestionHandler) SubmitAnswer(ctx *fiber.Ctx) error {
	var req entity.SubmitAnswerRequest

	if err := h.validator.ParseAndValidate(ctx, &req); err != nil {
		return response.NewFailed(domain.DYSLEXIA_QUESTION_SUBMIT_ANSWER_FAILED, fiber.NewError(fiber.StatusBadRequest, err.Error()), h.logger).Send(ctx)
	}

	result, err := h.usecase.SubmitAnswer(ctx.UserContext(), req)
	if err != nil {
		return response.NewFailed(domain.DYSLEXIA_QUESTION_SUBMIT_ANSWER_FAILED, fiber.NewError(fiber.StatusBadRequest, err.Error()), h.logger).Send(ctx)
	}

	return response.NewSuccess(domain.DYSLEXIA_QUESTION_SUBMIT_ANSWER_SUCCESS, result, nil).Send(ctx)
}

// GET /questions/sessions/:session_id
func (h *dyslexiaQuestionHandler) GetSessionAnswers(ctx *fiber.Ctx) error {
	sessionID := ctx.Params("session_id")
	if sessionID == "" {
		return response.NewFailed(domain.DYSLEXIA_QUESTION_GET_SESSION_FAILED, fiber.NewError(fiber.StatusBadRequest, "session_id is required"), h.logger).Send(ctx)
	}

	answers, err := h.usecase.GetSessionAnswers(ctx.UserContext(), sessionID)
	if err != nil {
		return response.NewFailed(domain.DYSLEXIA_QUESTION_GET_SESSION_FAILED, fiber.NewError(fiber.StatusBadRequest, err.Error()), h.logger).Send(ctx)
	}

	return response.NewSuccess(domain.DYSLEXIA_QUESTION_GET_SESSION_SUCCESS, answers, nil).Send(ctx)
}

// GET /report/sessions/:session_id
func (h *dyslexiaQuestionHandler) GetSessionReport(ctx *fiber.Ctx) error {
	sessionID := ctx.Params("session_id")
	if sessionID == "" {
		return response.NewFailed(domain.DYSLEXIA_QUESTION_GET_REPORT_FAILED, fiber.NewError(fiber.StatusBadRequest, "session_id is required"), h.logger).Send(ctx)
	}

	report, err := h.usecase.GenerateSessionReport(ctx.UserContext(), sessionID)
	if err != nil {
		return response.NewFailed(domain.DYSLEXIA_QUESTION_GET_REPORT_FAILED, fiber.NewError(fiber.StatusBadRequest, err.Error()), h.logger).Send(ctx)
	}

	return response.NewSuccess(domain.DYSLEXIA_QUESTION_GET_REPORT_SUCCESS, report, nil).Send(ctx)
}
