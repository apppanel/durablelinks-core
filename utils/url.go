package utils

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/rs/zerolog"
)

func ValidateURLScheme(urlStr string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %v", err)
	}

	validSchemes := []string{"http", "https"}
	if slices.Contains(validSchemes, u.Scheme) {
		return nil
	}

	return fmt.Errorf("link has invalid scheme. Must have schemes %v", validSchemes)
}

func IsURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func CleanHost(logger zerolog.Logger, raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("host is required")
	}

	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	host := u.Hostname()
	logger.Debug().
		Str("host", host).
		Msg("Cleaned host")

	return host, nil
}

func IsDomainAllowed(logger zerolog.Logger, allowList []string, rawLink string) bool {
	u, err := url.Parse(rawLink)
	if err != nil {
		logger.Error().
			Str("raw_link", rawLink).
			Msg("Invalid link")
		return false
	}
	host := strings.ToLower(u.Hostname())

	for _, allowed := range allowList {
		allowed = strings.ToLower(strings.TrimSpace(allowed))
		if host == allowed {
			return true
		}
	}
	return false
}
