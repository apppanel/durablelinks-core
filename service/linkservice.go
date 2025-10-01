package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/apppanel/durablelinks-core/models"
	"github.com/apppanel/durablelinks-core/repository"
	"github.com/apppanel/durablelinks-core/utils"
	"github.com/google/uuid"

	"github.com/rs/zerolog/log"
)

type TenantConfig struct {
	URLScheme             string
	DomainAllowList       []string
	ShortPathLength       int
	UnguessablePathLength int
}

type LinkService interface {
	CreateDurableLink(ctx context.Context, params models.CreateDurableLinkRequest, projectID *uuid.UUID, tenantCfg TenantConfig) (*models.ShortLinkResponse, error)
	ParseLongDurableLink(longLink string) (models.CreateDurableLinkRequest, error)
	ResolveShortPath(ctx context.Context, rawURL string, projectID *uuid.UUID, tenantCfg TenantConfig) (*models.LongLinkResponse, error)
	PrepareDurableLinkRequest(input map[string]any) (models.CreateDurableLinkRequest, error)
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
	tenantCfg TenantConfig,
) (*models.LongLinkResponse, error) {
	rawQueryStr, err := s.repo.GetQueryParamsByHostAndPath(ctx, host, path, projectID)
	if err != nil {
		return nil, err
	}

	longLink := fmt.Sprintf("%s://%s/%s", tenantCfg.URLScheme, host, path)
	if rawQueryStr != "" {
		longLink += "?" + rawQueryStr
	}

	log.Debug().
		Str("path", path).
		Str("long_link", longLink).
		Msg("Link retrieved from service")

	return &models.LongLinkResponse{
		LongLink: longLink,
	}, nil
}

func (s *linkService) CreateDurableLink(ctx context.Context, params models.CreateDurableLinkRequest, projectID *uuid.UUID, tenantCfg TenantConfig) (*models.ShortLinkResponse, error) {
	warnings := []models.Warning{}

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

	isi := params.DurableLinkInfo.IosParameters.IOSAppStoreId

	if isi != "" {
		if _, err := strconv.ParseUint(isi, 10, 64); err != nil {
			return nil, ErrInvalidAppStoreID
		}
	}

	queryParams := url.Values{}
	queryParams.Add("link", params.DurableLinkInfo.Link)

	addParam := func(key, value string) {
		if value != "" {
			queryParams.Add(key, value)
		}
	}

	addParam("apn", params.DurableLinkInfo.AndroidParameters.AndroidPackageName)
	addParam("afl", params.DurableLinkInfo.AndroidParameters.AndroidFallbackLink)
	addParam("amv", params.DurableLinkInfo.AndroidParameters.AndroidMinPackageVersionCode)

	addParam("ifl", params.DurableLinkInfo.IosParameters.IOSFallbackLink)
	addParam("ipfl", params.DurableLinkInfo.IosParameters.IOSIpadFallbackLink)
	addParam("isi", isi)

	addParam("ofl", params.DurableLinkInfo.OtherPlatformParameters.FallbackURL)

	addParam("st", params.DurableLinkInfo.SocialMetaTagInfo.SocialTitle)
	addParam("sd", params.DurableLinkInfo.SocialMetaTagInfo.SocialDescription)

	si := params.DurableLinkInfo.SocialMetaTagInfo.SocialImageLink

	addParam("si", si)

	if si != "" {
		if !utils.IsURL(si) {
			warnings = append(warnings, models.Warning{
				WarningCode:    "MALFORMED_PARAM",
				WarningMessage: "Param 'si' is not a valid URL",
			})
		}
	}

	addParam("utm_source", params.DurableLinkInfo.AnalyticsInfo.MarketingParameters.UtmSource)
	addParam("utm_medium", params.DurableLinkInfo.AnalyticsInfo.MarketingParameters.UtmMedium)
	addParam("utm_campaign", params.DurableLinkInfo.AnalyticsInfo.MarketingParameters.UtmCampaign)
	addParam("utm_term", params.DurableLinkInfo.AnalyticsInfo.MarketingParameters.UtmTerm)
	addParam("utm_content", params.DurableLinkInfo.AnalyticsInfo.MarketingParameters.UtmContent)

	itunesAnalytics := params.DurableLinkInfo.AnalyticsInfo.ItunesConnectAnalytics
	pt := itunesAnalytics.Pt
	addParam("pt", pt)

	addUnrecognizedWarning := func(paramName, paramValue, missingParam string) {
		if paramValue != "" {
			warnings = append(warnings, models.Warning{
				WarningCode:    "UNRECOGNIZED_PARAM",
				WarningMessage: fmt.Sprintf("Param '%s' is not needed, since '%s' is not specified.", paramName, missingParam),
			})
		}
	}

	if isi == "" {
		addUnrecognizedWarning("at", itunesAnalytics.At, "isi")
		addUnrecognizedWarning("ct", itunesAnalytics.Ct, "isi")
		addUnrecognizedWarning("mt", itunesAnalytics.Mt, "isi")
		addUnrecognizedWarning("pt", pt, "isi")
	}

	if pt == "" {
		addUnrecognizedWarning("at", itunesAnalytics.At, "pt")
		addUnrecognizedWarning("ct", itunesAnalytics.Ct, "pt")
		addUnrecognizedWarning("mt", itunesAnalytics.Mt, "pt")
	}

	addParam("at", itunesAnalytics.At)
	addParam("ct", itunesAnalytics.Ct)
	addParam("mt", itunesAnalytics.Mt)

	shortPath := params.Suffix.Option == "SHORT"
	response, err := s.createOrGetShortLink(ctx, host, queryParams, shortPath, projectID, tenantCfg)
	if err != nil {
		return nil, err
	}

	response.Warnings = warnings
	return response, nil
}

func (s *linkService) parseLongDurableLink(longDurableLink string) (models.CreateDurableLinkRequest, error) {
	var req models.CreateDurableLinkRequest

	log.Debug().
		Str("long_link", longDurableLink).
		Msg("Parsing long dynamic link")

	u, err := url.Parse(longDurableLink)
	if err != nil {
		return req, ErrInvalidURLFormat
	}

	if u.Host == "" {
		return req, ErrHostInvalid
	}

	req.DurableLinkInfo.Host = u.Host

	params := u.Query()

	req.DurableLinkInfo.Link = params.Get("link")

	log.Debug().
		Str("link", req.DurableLinkInfo.Link).
		Msg("Parsed link")

	setParam := func(paramKey string, dest *string) {
		if value := params.Get(paramKey); value != "" {
			*dest = value
		}
	}

	setParam("apn", &req.DurableLinkInfo.AndroidParameters.AndroidPackageName)
	setParam("afl", &req.DurableLinkInfo.AndroidParameters.AndroidFallbackLink)
	setParam("amv", &req.DurableLinkInfo.AndroidParameters.AndroidMinPackageVersionCode)

	setParam("isi", &req.DurableLinkInfo.IosParameters.IOSAppStoreId)
	setParam("ifl", &req.DurableLinkInfo.IosParameters.IOSFallbackLink)
	setParam("ipfl", &req.DurableLinkInfo.IosParameters.IOSIpadFallbackLink)

	setParam("ofl", &req.DurableLinkInfo.OtherPlatformParameters.FallbackURL)

	setParam("utm_source", &req.DurableLinkInfo.AnalyticsInfo.MarketingParameters.UtmSource)
	setParam("utm_medium", &req.DurableLinkInfo.AnalyticsInfo.MarketingParameters.UtmMedium)
	setParam("utm_campaign", &req.DurableLinkInfo.AnalyticsInfo.MarketingParameters.UtmCampaign)
	setParam("utm_term", &req.DurableLinkInfo.AnalyticsInfo.MarketingParameters.UtmTerm)

	setParam("utm_content", &req.DurableLinkInfo.AnalyticsInfo.MarketingParameters.UtmContent)
	setParam("at", &req.DurableLinkInfo.AnalyticsInfo.ItunesConnectAnalytics.At)
	setParam("ct", &req.DurableLinkInfo.AnalyticsInfo.ItunesConnectAnalytics.Ct)
	setParam("mt", &req.DurableLinkInfo.AnalyticsInfo.ItunesConnectAnalytics.Mt)
	setParam("pt", &req.DurableLinkInfo.AnalyticsInfo.ItunesConnectAnalytics.Pt)

	setParam("st", &req.DurableLinkInfo.SocialMetaTagInfo.SocialTitle)
	setParam("sd", &req.DurableLinkInfo.SocialMetaTagInfo.SocialDescription)
	setParam("si", &req.DurableLinkInfo.SocialMetaTagInfo.SocialImageLink)

	setParam("path", &req.Suffix.Option)

	log.Debug().
		Str("req", fmt.Sprintf("%+v", req)).
		Msg("Parsed long dynamic link")

	return req, nil
}

func (s *linkService) createOrGetShortLink(
	ctx context.Context,
	host string,
	queryParams url.Values,
	shortPath bool,
	projectID *uuid.UUID,
	tenantCfg TenantConfig,
) (*models.ShortLinkResponse, error) {
	rawQS := queryParams.Encode()
	if shortPath {
		if path, err := s.findExistingShortLink(ctx, host, rawQS, projectID); err == nil {
			full := fmt.Sprintf("%s://%s/%s", tenantCfg.URLScheme, host, path)
			log.Debug().
				Str("path", path).
				Str("query_params", rawQS).
				Msg("Re‑using existing short link")
			return &models.ShortLinkResponse{ShortLink: full, Warnings: []models.Warning{}}, nil

		} else if err != sql.ErrNoRows {
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

	if err := s.createShortLink(ctx, host, path, rawQS, !shortPath, projectID); err != nil {
		return nil, fmt.Errorf("failed to store link: %w", err)
	}

	full := fmt.Sprintf("%s://%s/%s", tenantCfg.URLScheme, host, path)
	log.Debug().
		Str("path", path).
		Str("query_params", rawQS).
		Msg("New link stored in database")

	return &models.ShortLinkResponse{ShortLink: full, Warnings: []models.Warning{}}, nil
}

func (s *linkService) findExistingShortLink(
	ctx context.Context,
	host, rawQS string,
	projectID *uuid.UUID,
) (string, error) {
	return s.repo.FindExistingShortLink(ctx, host, rawQS, projectID)
}

func (s *linkService) createShortLink(
	ctx context.Context,
	host, path, rawQS string,
	unguessable bool,
	projectID *uuid.UUID,
) error {
	return s.repo.CreateShortLink(ctx, host, path, rawQS, unguessable, projectID)
}

func (s *linkService) ResolveShortPath(ctx context.Context, rawURL string, projectID *uuid.UUID, tenantCfg TenantConfig) (*models.LongLinkResponse, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, ErrInvalidRequestedLink
	}

	normalizedHost := removePreviewFromHost(u.Host)

	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(pathParts) != 1 {
		return nil, fmt.Errorf("unexpected path format: %w", ErrInvalidPathFormat)
	}

	return s.getLongLinkFromHostAndPath(ctx, normalizedHost, pathParts[0], projectID, tenantCfg)
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
	if strings.HasPrefix(host, "preview.") {
		return strings.TrimPrefix(host, "preview.")
	}
	parts := strings.SplitN(host, ".", 2) // ["acme-preview", "short.link"]
	if len(parts) == 2 && strings.HasSuffix(parts[0], "-preview") {
		app := strings.TrimSuffix(parts[0], "-preview")
		return app + "." + parts[1]
	}

	return host
}

func (s *linkService) PrepareDurableLinkRequest(input map[string]any) (models.CreateDurableLinkRequest, error) {
	var req models.CreateDurableLinkRequest

	if longLink, ok := input["longDurableLink"].(string); ok && longLink != "" {
		parsedReq, err := s.parseLongDurableLink(longLink)
		if err != nil {
			return models.CreateDurableLinkRequest{}, err
		}
		req = parsedReq
	} else {
		reqBytes, err := json.Marshal(input)
		if err != nil {
			return models.CreateDurableLinkRequest{}, ErrInvalidFormat
		}
		if err := json.Unmarshal(reqBytes, &req); err != nil {
			return models.CreateDurableLinkRequest{}, ErrInvalidFormat
		}
	}

	if req.DurableLinkInfo.Host == "" {
		return models.CreateDurableLinkRequest{}, ErrMissingHost
	}
	if req.DurableLinkInfo.Link == "" {
		return models.CreateDurableLinkRequest{}, ErrMissingLink
	}
	if err := utils.ValidateURLScheme(req.DurableLinkInfo.Link); err != nil {
		return models.CreateDurableLinkRequest{}, err
	}

	return req, nil
}
