package validation

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

type Validator struct {
	validate *validator.Validate
}

func New() *Validator {
	instance := validator.New(validator.WithRequiredStructEnabled())
	instance.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
	return &Validator{validate: instance}
}

func (v *Validator) Struct(value any) map[string]string {
	err := v.validate.Struct(value)
	if err == nil {
		return nil
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return map[string]string{"_": "Data tidak dapat divalidasi"}
	}

	fields := make(map[string]string)
	for _, fieldError := range validationErrors {
		fields[fieldError.Field()] = message(fieldError)
	}
	return fields
}

func message(field validator.FieldError) string {
	switch field.Tag() {
	case "required":
		return "Wajib diisi"
	case "email":
		return "Format email tidak valid"
	case "min":
		return "Nilai terlalu pendek"
	case "max":
		return "Nilai terlalu panjang"
	default:
		return "Nilai tidak valid"
	}
}
