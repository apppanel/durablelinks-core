package repository

import (
	"context"
	"errors"

	"github.com/apppanel/durablelinks-core/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type LinkRepository interface {
	GetLinkByHostAndPath(ctx context.Context, host, path string, projectID *uuid.UUID) (*models.DurableLink, error)
	FindExistingShortLink(ctx context.Context, host string, link *models.DurableLink, projectID *uuid.UUID) (string, error)
	CreateShortLink(ctx context.Context, link *models.DurableLinkDB, projectID *uuid.UUID) error
}

type linkRepository struct {
	db *gorm.DB
}

func NewLinkRepository(db *gorm.DB) LinkRepository {
	return &linkRepository{
		db: db,
	}
}

func (r *linkRepository) GetLinkByHostAndPath(ctx context.Context, host, path string, projectID *uuid.UUID) (*models.DurableLink, error) {
	var dbLink models.DurableLinkDB

	query := r.db.WithContext(ctx).Where("host = ? AND path = ?", host, path)

	if projectID != nil {
		projectIDStr := projectID.String()
		query = query.Where("project_id = ?", projectIDStr)
	}

	err := query.First(&dbLink).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
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
		return nil, err
	}

	dl := dbLink.ToDurableLink()
	return &dl, nil
}

func (r *linkRepository) FindExistingShortLink(ctx context.Context, host string, link *models.DurableLink, projectID *uuid.UUID) (string, error) {
	var result struct {
		Path string
	}

	dbLink := models.FromDurableLink(*link, "", "", false, nil)
	paramsHash := dbLink.ComputeParamsHash()

	query := r.db.WithContext(ctx).
		Model(&models.DurableLinkDB{}).
		Select("path").
		Where("host = ?", host).
		Where("link = ?", link.Link).
		Where("params_hash = ?", paramsHash).
		Where("is_unguessable_path = ?", false)

	if projectID != nil {
		projectIDStr := projectID.String()
		query = query.Where("project_id = ?", projectIDStr)
	} else {
		query = query.Where("project_id IS NULL")
	}

	err := query.Limit(1).First(&result).Error
	return result.Path, err
}

func (r *linkRepository) CreateShortLink(ctx context.Context, link *models.DurableLinkDB, projectID *uuid.UUID) error {
	if projectID != nil {
		projectIDStr := projectID.String()
		link.ProjectID = &projectIDStr
	}

	return r.db.WithContext(ctx).Create(link).Error
}
