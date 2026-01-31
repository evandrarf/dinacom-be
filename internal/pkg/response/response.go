package response

import (
	"github.com/evandrarf/dinacom-be/internal/pkg/validate"
	"github.com/gofiber/fiber/v2"

	"github.com/sirupsen/logrus"
)

type Response struct {
	StatusCode int    `json:"-"`
	Success    bool   `json:"success"`
	Message    string `json:"message,omitempty"`
	Error      any    `json:"error,omitempty"`
	Data       any    `json:"data,omitempty"`
	Meta       any    `json:"meta,omitempty"`
}

func NewInternalServerError() *Response {
	res := &Response{
		Success:    false,
		Message:    "Internal Server Error",
		StatusCode: fiber.StatusInternalServerError,
	}
	return res
}

func NewFailed(msg string, err error, logger *logrus.Logger) *Response {
	res := &Response{
		Success:    false,
		Message:    msg,
		StatusCode: fiber.StatusInternalServerError,
	}

	if e, ok := err.(*fiber.Error); ok {
		res.StatusCode = e.Code
		if e.Message != "" {
			res.Error = e.Message
		}
	} else if errors, ok := err.(*validate.FieldsError); ok {
		res.StatusCode = fiber.StatusBadRequest
		res.Error = errors.Fields
	}

	if logger != nil && res.StatusCode >= fiber.StatusInternalServerError {
		logger.Error(err)
	}

	return res
}

func NewSuccess(msg string, data any, meta any) *Response {
	res := &Response{
		Success:    true,
		Message:    msg,
		StatusCode: fiber.StatusOK,
		Data:       data,
		Meta:       meta,
	}

	return res
}

func (r *Response) Send(ctx *fiber.Ctx) error {
	return ctx.Status(r.StatusCode).JSON(r)
}
