package validate

import (
	"reflect"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/gofiber/fiber/v2"
)

type Validator struct {
	validate *validator.Validate
	trans    ut.Translator
}

func NewValidator() *Validator {
	validator := validator.New(validator.WithRequiredStructEnabled())

	// Registering english translator
	english := en.New()
	uni := ut.New(english, english)
	trans, _ := uni.GetTranslator("en")
	en_translations.RegisterDefaultTranslations(validator, trans)

	// Registering field name translation
	validator.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return &Validator{
		validate: validator,
		trans:    trans,
	}
}

func (v *Validator) ParseAndValidate(ctx *fiber.Ctx, req interface{}) error {
	if err := ctx.BodyParser(req); err != nil {
		return err
	}

	err := v.validate.Struct(req)
	if err == nil {
		return nil
	}

	errors, ok := err.(validator.ValidationErrors)
	if !ok {
		return fiber.NewError(fiber.StatusBadRequest, "Request body is not valid")
	}

	fields := v.translateError(errors)
	return NewFieldsError(fields)
}

func (v *Validator) translateError(errs validator.ValidationErrors) (fields map[string]string) {
	fields = make(map[string]string)
	for _, e := range errs {
		fields[e.Field()] = e.Translate(v.trans)
	}
	return fields
}
