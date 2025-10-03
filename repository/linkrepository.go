package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/apppanel/durablelinks-core/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type LinkRepository interface {
	GetLinkByHostAndPath(ctx context.Context, host, path string, projectID *uuid.UUID) (*models.DurableLink, error)
	FindExistingShortLink(ctx context.Context, host string, link *models.DurableLink, projectID *uuid.UUID) (string, error)
	CreateShortLink(ctx context.Context, link *models.DurableLinkDB, projectID *uuid.UUID) error
}

type linkRepository struct {
	db *sql.DB
}

func NewLinkRepository(db *sql.DB) LinkRepository {
	return &linkRepository{
		db: db,
	}
}

func (r *linkRepository) GetLinkByHostAndPath(ctx context.Context, host, path string, projectID *uuid.UUID) (*models.DurableLink, error) {
	var dbLink models.DurableLinkDB

	query := `SELECT * FROM apppanel_durable_links WHERE host = $1 AND path = $2`
	args := []interface{}{host, path}

	if projectID != nil {
		query += ` AND project_id = $3`
		args = append(args, projectID.String())
	}

	row := r.db.QueryRowContext(ctx, query, args...)
	err := row.Scan(
		&dbLink.ID,
		&dbLink.Host,
		&dbLink.Path,
		&dbLink.Link,
		&dbLink.IsUnguessablePath,
		&dbLink.ProjectID,
		&dbLink.AndroidPackageName,
		&dbLink.AndroidFallbackLink,
		&dbLink.AndroidMinVersion,
		&dbLink.IOSFallbackLink,
		&dbLink.IOSIpadFallbackLink,
		&dbLink.IOSAppStoreID,
		&dbLink.SocialTitle,
		&dbLink.SocialDescription,
		&dbLink.SocialImageLink,
		&dbLink.UtmSource,
		&dbLink.UtmMedium,
		&dbLink.UtmCampaign,
		&dbLink.UtmTerm,
		&dbLink.UtmContent,
		&dbLink.ItunesPt,
		&dbLink.ItunesAt,
		&dbLink.ItunesCt,
		&dbLink.ItunesMt,
		&dbLink.OtherFallbackURL,
		&dbLink.ParamsHash,
		&dbLink.CreatedAt,
		&dbLink.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Debug().
				Str("host", host).
				Str("path", path).
				Msg("Link not found in database")
			return nil, ErrLinkNotFound
		}
		log.Error().
			Err(err).
			Str("host", host).
			Str("path", path).
			Msg("Failed to retrieve link from database")
		return nil, fmt.Errorf("database error: %w", err)
	}

	dl := dbLink.ToDurableLink()
	return &dl, nil
}

func (r *linkRepository) FindExistingShortLink(ctx context.Context, host string, link *models.DurableLink, projectID *uuid.UUID) (string, error) {
	var path string
	dbLink := models.FromDurableLink(*link, "", "", false, nil)
	paramsHash := dbLink.ComputeParamsHash()
	query := `
		SELECT path FROM apppanel_durable_links
		WHERE host = $1
		AND link = $2
		AND params_hash = $3
		AND is_unguessable_path = FALSE
	`
	args := []interface{}{host, link.Link, paramsHash}

	if projectID != nil {
		query += ` AND project_id = $4`
		args = append(args, projectID.String())
	} else {
		query += ` AND project_id IS NULL`
	}
	query += ` LIMIT 1`
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&path)
	return path, err
}

func (r *linkRepository) CreateShortLink(ctx context.Context, link *models.DurableLinkDB, projectID *uuid.UUID) error {
	query := `
		INSERT INTO apppanel_durable_links (
			host, path, link, is_unguessable_path, project_id,
			android_package_name, android_fallback_link, android_min_version,
			ios_fallback_link, ios_ipad_fallback_link, ios_app_store_id,
			social_title, social_description, social_image_link,
			utm_source, utm_medium, utm_campaign, utm_term, utm_content,
			itunes_pt, itunes_at, itunes_ct, itunes_mt,
			other_fallback_url, params_hash
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11,
			$12, $13, $14,
			$15, $16, $17, $18, $19,
			$20, $21, $22, $23,
			$24, $25
		)`

	var projectIDStr *string
	if projectID != nil {
		s := projectID.String()
		projectIDStr = &s
	}

	_, err := r.db.ExecContext(ctx, query,
		link.Host,
		link.Path,
		link.Link,
		link.IsUnguessablePath,
		projectIDStr,
		link.AndroidPackageName,
		link.AndroidFallbackLink,
		link.AndroidMinVersion,
		link.IOSFallbackLink,
		link.IOSIpadFallbackLink,
		link.IOSAppStoreID,
		link.SocialTitle,
		link.SocialDescription,
		link.SocialImageLink,
		link.UtmSource,
		link.UtmMedium,
		link.UtmCampaign,
		link.UtmTerm,
		link.UtmContent,
		link.ItunesPt,
		link.ItunesAt,
		link.ItunesCt,
		link.ItunesMt,
		link.OtherFallbackURL,
		link.ParamsHash,
	)
	return err
}
