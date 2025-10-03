package models

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

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

// ValidateStruct validates a struct using the validator
func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}

// ParseAndValidateCreateRequest parses JSON and validates a CreateDurableLinkRequest
func ParseAndValidateCreateRequest(body io.Reader) (*CreateDurableLinkRequest, error) {
	var req CreateDurableLinkRequest
	var allErrors []ValidationError

	// Decode the request
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		// Check if it's a type mismatch error (e.g., string instead of int)
		if jsonErr, ok := err.(*json.UnmarshalTypeError); ok {
			allErrors = append(allErrors, ValidationError{
				Field:   jsonErr.Field,
				Tag:     "type",
				Message: fmt.Sprintf("Invalid type for field '%s': expected %s but got %s", jsonErr.Field, jsonErr.Type.String(), jsonErr.Value),
			})
			// Continue to validate other fields even after type error
		} else {
			// For other JSON errors, return a single generic error
			allErrors = append(allErrors, ValidationError{
				Field:   "",
				Tag:     "json",
				Message: "Invalid request body: " + err.Error(),
			})
			return nil, ValidationErrors{Errors: allErrors}
		}
	}

	// Run struct validation to catch missing/invalid fields
	if err := ValidateStruct(&req); err != nil {
		validationErrors := parseValidationErrors(err)
		allErrors = append(allErrors, validationErrors...)
	}

	if len(allErrors) > 0 {
		return nil, ValidationErrors{Errors: allErrors}
	}

	return &req, nil
}

func parseValidationErrors(err error) []ValidationError {
	var validationErrors []ValidationError

	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		for _, fieldErr := range validationErrs {
			validationError := ValidationError{
				Field:   getJSONFieldName(fieldErr),
				Tag:     fieldErr.Tag(),
				Message: getValidationMessage(fieldErr),
			}
			validationErrors = append(validationErrors, validationError)
		}
	}

	return validationErrors
}

func getJSONFieldName(fieldErr validator.FieldError) string {
	// Convert struct field path to JSON field path
	// e.g., "DurableLinkInfo.Host" -> "durableLinkInfo.host"
	namespace := fieldErr.Namespace()
	parts := strings.Split(namespace, ".")

	if len(parts) <= 1 {
		return fieldErr.Field()
	}

	// Skip the first part (struct name) and convert to camelCase
	jsonParts := make([]string, 0, len(parts)-1)
	for i := 1; i < len(parts); i++ {
		jsonParts = append(jsonParts, toLowerFirst(parts[i]))
	}

	return strings.Join(jsonParts, ".")
}

func toLowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func getValidationMessage(fieldErr validator.FieldError) string {
	switch fieldErr.Tag() {
	case "required":
		return fmt.Sprintf("Field '%s' is required", getJSONFieldName(fieldErr))
	case "url":
		return fmt.Sprintf("Field '%s' must be a valid URL", getJSONFieldName(fieldErr))
	case "url_scheme":
		return fmt.Sprintf("Field '%s' has an invalid URL scheme", getJSONFieldName(fieldErr))
	default:
		return fmt.Sprintf("Field '%s' failed validation on '%s' tag", getJSONFieldName(fieldErr), fieldErr.Tag())
	}
}
