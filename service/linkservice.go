package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/apppanel/durablelinks-core/models"
	"github.com/apppanel/durablelinks-core/repository"
	"github.com/apppanel/durablelinks-core/utils"
	"github.com/google/uuid"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type TenantConfig struct {
	URLScheme             string
	DomainAllowList       []string
	ShortPathLength       int
	UnguessablePathLength int
	DefaultIOSAppStoreId  *int64
	DefaultAndroidPackage *string
}

type LinkService interface {
	CreateDurableLink(ctx context.Context, params models.CreateDurableLinkRequest, projectID *uuid.UUID, tenantCfg TenantConfig) (*models.ShortLinkResponse, error)
	ParseLongDurableLink(longLink string) (models.CreateDurableLinkRequest, error)
	ResolveShortPath(ctx context.Context, rawURL string, projectID *uuid.UUID, tenantCfg TenantConfig) (*models.LongLinkResponse, error)
}

type linkService struct {
	repo repository.LinkRepository
}

func NewLinkService(repo repository.LinkRepository) *linkService {
	return &linkService{
		repo: repo,
	}
}

func (s *linkService) getLongLinkFromHostAndPath(
	ctx context.Context,
	host string,
	path string,
	projectID *uuid.UUID,
) (*models.LongLinkResponse, error) {
	link, err := s.repo.GetLinkByHostAndPath(ctx, host, path, projectID)
	if err != nil {
		return nil, err
	}

	log.Debug().
		Str("path", path).
		Str("long_link", link.Link).
		Msg("Link retrieved from service")

	return &models.LongLinkResponse{
		LongLink: link.Link,
	}, nil
}

func (s *linkService) CreateDurableLink(ctx context.Context, params models.CreateDurableLinkRequest, projectID *uuid.UUID, tenantCfg TenantConfig) (*models.ShortLinkResponse, error) {
	log.Debug().
		Str("params", fmt.Sprintf("%+v", params)).
		Msg("Dynamic link parameters")

	host, err := utils.CleanHost(log.Logger, params.DurableLinkInfo.Host)
	if err != nil {
		log.Error().
			Str("host", params.DurableLinkInfo.Host).
			Msg("Invalid host")
		return nil, fmt.Errorf("invalid host: %w", err)
	}

	if !utils.IsDomainAllowed(log.Logger, tenantCfg.DomainAllowList, params.DurableLinkInfo.Link) {
		log.Error().
			Str("link", params.DurableLinkInfo.Link).
			Msg("Domain link not in allow list")
		return nil, ErrDomainLinkNotAllowed
	}

	warnings := []models.Warning{}

	// Apply defaults from tenant config if not provided
	if params.DurableLinkInfo.IosParameters.IOSAppStoreId == nil && tenantCfg.DefaultIOSAppStoreId != nil {
		params.DurableLinkInfo.IosParameters.IOSAppStoreId = tenantCfg.DefaultIOSAppStoreId
		warnings = append(warnings, models.Warning{
			WarningCode:    "DEFAULT_APPLIED",
			WarningMessage: fmt.Sprintf("Using default iOS App Store ID: %d", *tenantCfg.DefaultIOSAppStoreId),
		})
	}

	if params.DurableLinkInfo.AndroidParameters.AndroidPackageName == nil && tenantCfg.DefaultAndroidPackage != nil {
		params.DurableLinkInfo.AndroidParameters.AndroidPackageName = tenantCfg.DefaultAndroidPackage
		warnings = append(warnings, models.Warning{
			WarningCode:    "DEFAULT_APPLIED",
			WarningMessage: fmt.Sprintf("Using default Android package name: %s", *tenantCfg.DefaultAndroidPackage),
		})
	}

	validationWarnings := s.validateLinkParameters(&params.DurableLinkInfo)
	warnings = append(warnings, validationWarnings...)

	shortPath, suffixWarning := s.validateSuffixOption(params.Suffix)
	if suffixWarning != nil {
		warnings = append(warnings, *suffixWarning)
	}
	response, err := s.createOrGetShortLink(ctx, host, params.DurableLinkInfo, shortPath, projectID, tenantCfg)
	if err != nil {
		return nil, err
	}

	response.Warnings = warnings
	return response, nil
}

func (s *linkService) validateSuffixOption(suffix models.Suffix) (bool, *models.Warning) {
	option := strings.ToUpper(suffix.Option)
	if option != "SHORT" && option != "UNGUESSABLE" {
		return false, &models.Warning{
			WarningCode:    "INVALID_SUFFIX_OPTION",
			WarningMessage: fmt.Sprintf("Param 'suffix.option' must be 'SHORT' or 'UNGUESSABLE'. Received '%s', defaulting to 'UNGUESSABLE'.", suffix.Option),
		}
	}

	if option == "UNGUESSABLE" {
		return false, nil
	}
	return true, nil
}

func (s *linkService) validateLinkParameters(dl *models.DurableLink) []models.Warning {
	warnings := []models.Warning{}

	addUnrecognizedWarning := func(paramName string, paramValue *string, missingParam string) {
		if paramValue != nil && *paramValue != "" {
			warnings = append(warnings, models.Warning{
				WarningCode:    "UNRECOGNIZED_PARAM",
				WarningMessage: fmt.Sprintf("Param '%s' is not needed, since '%s' is not specified.", paramName, missingParam),
			})
		}
	}

	validateAndClearInvalidURL := func(url **string, jsonFieldName string) {
		if *url != nil && **url != "" && !utils.IsURL(**url) {
			warnings = append(warnings, models.Warning{
				WarningCode:    "MALFORMED_PARAM",
				WarningMessage: fmt.Sprintf("Param '%s' is not a valid URL", jsonFieldName),
			})
			// Clear invalid URL - don't save garbage data
			*url = nil
		}
	}

	validateAndClearInvalidURL(&dl.AndroidParameters.AndroidFallbackLink, "androidFallbackLink")
	validateAndClearInvalidURL(&dl.IosParameters.IOSFallbackLink, "iosFallbackLink")
	validateAndClearInvalidURL(&dl.IosParameters.IOSIpadFallbackLink, "iosIpadFallbackLink")
	validateAndClearInvalidURL(&dl.OtherPlatformParameters.FallbackURL, "fallbackUrl")
	validateAndClearInvalidURL(&dl.SocialMetaTagInfo.SocialImageLink, "socialImageLink")

	isi := dl.IosParameters.IOSAppStoreId
	itunes := dl.AnalyticsInfo.ItunesConnectAnalytics
	pt := itunes.Pt

	if isi == nil {
		addUnrecognizedWarning("at", itunes.At, "isi")
		addUnrecognizedWarning("ct", itunes.Ct, "isi")
		addUnrecognizedWarning("mt", itunes.Mt, "isi")
		addUnrecognizedWarning("pt", pt, "isi")
	}

	if pt == nil || *pt == "" {
		addUnrecognizedWarning("at", itunes.At, "pt")
		addUnrecognizedWarning("ct", itunes.Ct, "pt")
		addUnrecognizedWarning("mt", itunes.Mt, "pt")
	}

	return warnings
}

func (s *linkService) createOrGetShortLink(
	ctx context.Context,
	host string,
	link models.DurableLink,
	shortPath bool,
	projectID *uuid.UUID,
	tenantCfg TenantConfig,
) (*models.ShortLinkResponse, error) {
	if shortPath {
		if path, err := s.repo.FindExistingShortLink(ctx, host, &link, projectID); err == nil {
			full := fmt.Sprintf("%s://%s/%s", tenantCfg.URLScheme, host, path)
			log.Debug().
				Str("path", path).
				Str("link", link.Link).
				Msg("Re-using existing short link")
			return &models.ShortLinkResponse{ShortLink: full, Warnings: []models.Warning{}}, nil

		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Error().
				Err(err).
				Msg("Error querying for existing short link")
			return nil, err
		}
	}

	length := tenantCfg.ShortPathLength
	if !shortPath {
		length = tenantCfg.UnguessablePathLength
	}
	path := generateDurableLinkPath(length)

	var projectIDStr *string
	if projectID != nil {
		idStr := projectID.String()
		projectIDStr = &idStr
	}

	dbLink := models.FromDurableLink(link, host, path, !shortPath, projectIDStr)
	if err := s.repo.CreateShortLink(ctx, dbLink, projectID); err != nil {
		return nil, fmt.Errorf("failed to store link: %w", err)
	}

	full := fmt.Sprintf("%s://%s/%s", tenantCfg.URLScheme, host, path)
	log.Debug().
		Str("path", path).
		Str("link", link.Link).
		Msg("New link stored in database")

	return &models.ShortLinkResponse{ShortLink: full, Warnings: []models.Warning{}}, nil
}

func (s *linkService) ResolveShortPath(ctx context.Context, rawURL string, projectID *uuid.UUID, tenantCfg TenantConfig) (*models.LongLinkResponse, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, ErrInvalidRequestedLink
	}

	normalizedHost := removePreviewFromHost(u.Host)

	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(pathParts) != 1 || pathParts[0] == "" {
		return nil, ErrInvalidPathFormat
	}

	return s.getLongLinkFromHostAndPath(ctx, normalizedHost, pathParts[0], projectID)
}

func generateDurableLinkPath(length int) string {
	const alphanumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		log.Panic().Err(err).Msg("Failed to generate random bytes")
	}

	for i := range b {
		b[i] = alphanumeric[b[i]%byte(len(alphanumeric))]
	}

	id := string(b)

	log.Debug().
		Str("short_code", id).
		Msg("Generated alphanumeric short ID")

	return id
}

func removePreviewFromHost(host string) string {
	if after, ok := strings.CutPrefix(host, "preview."); ok {
		return after
	}
	parts := strings.SplitN(host, ".", 2) // ["acme-preview", "short.link"]
	if len(parts) == 2 && strings.HasSuffix(parts[0], "-preview") {
		app := strings.TrimSuffix(parts[0], "-preview")
		return app + "." + parts[1]
	}

	return host
}
