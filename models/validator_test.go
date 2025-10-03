package models

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAndValidateCreateRequest_Success(t *testing.T) {
	jsonBody := `{
		"durableLinkInfo": {
			"host": "example.com",
			"link": "https://example.com/target"
		},
		"suffix": {
			"option": "SHORT"
		}
	}`

	req, err := ParseAndValidateCreateRequest(bytes.NewBufferString(jsonBody))
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, "example.com", req.DurableLinkInfo.Host)
	assert.Equal(t, "https://example.com/target", req.DurableLinkInfo.Link)
	assert.Equal(t, "SHORT", req.Suffix.Option)
}

func TestParseAndValidateCreateRequest_MissingRequiredField(t *testing.T) {
	jsonBody := `{
		"durableLinkInfo": {
			"host": "example.com"
		}
	}`

	req, err := ParseAndValidateCreateRequest(bytes.NewBufferString(jsonBody))
	require.Error(t, err)
	require.Nil(t, req)

	var validationErrs ValidationErrors
	ok := errors.As(err, &validationErrs)
	require.True(t, ok, "Expected ValidationErrors type")
	require.NotEmpty(t, validationErrs.Errors)

	// Check that we got a validation error for the missing 'link' field
	found := false
	for _, ve := range validationErrs.Errors {
		if ve.Field == "durableLinkInfo.link" && ve.Tag == "required" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected validation error for missing 'link' field")
}

func TestParseAndValidateCreateRequest_InvalidURL(t *testing.T) {
	jsonBody := `{
		"durableLinkInfo": {
			"host": "example.com",
			"link": "not-a-valid-url"
		}
	}`

	req, err := ParseAndValidateCreateRequest(bytes.NewBufferString(jsonBody))
	require.Error(t, err)
	require.Nil(t, req)

	var validationErrs ValidationErrors
	ok := errors.As(err, &validationErrs)
	require.True(t, ok, "Expected ValidationErrors type")
	require.NotEmpty(t, validationErrs.Errors)

	// Check that we got a validation error for invalid URL
	found := false
	for _, ve := range validationErrs.Errors {
		if ve.Field == "durableLinkInfo.link" && ve.Tag == "url" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected validation error for invalid URL")
}

func TestParseAndValidateCreateRequest_InvalidJSON(t *testing.T) {
	jsonBody := `{invalid json`

	req, err := ParseAndValidateCreateRequest(bytes.NewBufferString(jsonBody))
	require.Error(t, err)
	require.Nil(t, req)

	var validationErrs ValidationErrors
	ok := errors.As(err, &validationErrs)
	require.True(t, ok, "Expected ValidationErrors type")
	require.NotEmpty(t, validationErrs.Errors)
	assert.Equal(t, "json", validationErrs.Errors[0].Tag)
}

func TestParseAndValidateCreateRequest_TypeMismatch(t *testing.T) {
	jsonBody := `{
		"durableLinkInfo": {
			"host": "example.com",
			"link": "https://example.com/target",
			"iosParameters": {
				"iosAppStoreId": "not-a-number"
			}
		}
	}`

	req, err := ParseAndValidateCreateRequest(bytes.NewBufferString(jsonBody))
	require.Error(t, err)
	require.Nil(t, req)

	var validationErrs ValidationErrors
	ok := errors.As(err, &validationErrs)
	require.True(t, ok, "Expected ValidationErrors type")
	require.NotEmpty(t, validationErrs.Errors)

	// Should have a type error for iosAppStoreId
	found := false
	for _, ve := range validationErrs.Errors {
		if ve.Tag == "type" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected type mismatch error")
}
