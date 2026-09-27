package response

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type Response struct {
	Code    int    `json:"code"`
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func Success(c fiber.Ctx, code int, message string, data any) error {
	return c.Status(code).JSON(Response{
		Code:    code,
		Status:  true,
		Message: message,
		Data:    data,
	})
}

func Error(c fiber.Ctx, code int, message string, data any) error {
	return c.Status(code).JSON(Response{
		Code:    code,
		Status:  false,
		Message: message,
		Data:    data,
	})
}

func FormatValidationErrors(err error) map[string]string {
	errorsMap := make(map[string]string)

	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrs {
			var msg string
			switch e.Tag() {
			case "required":
				msg = fmt.Sprintf("field '%s' is required", e.Field())
			case "email":
				msg = fmt.Sprintf("field '%s' must be a valid email address", e.Field())
			case "min":
				msg = fmt.Sprintf("field '%s' must be at least %s characters", e.Field(), e.Param())
			case "strong_password":
				msg = fmt.Sprintf("field '%s' must be at least 8 characters long and include uppercase, lowercase, number, and special character", e.Field())
			default:
				msg = fmt.Sprintf("field '%s' is invalid (%s)", e.Field(), e.Tag())
			}

			errorsMap[e.Field()] = msg
		}
	}

	return errorsMap
}
