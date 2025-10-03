package models

import (
	"github.com/apppanel/durablelinks-core/utils"
	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()

	validate.RegisterValidation("url_scheme", func(fl validator.FieldLevel) bool {
		return utils.ValidateURLScheme(fl.Field().String()) == nil
	})
}

func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}
