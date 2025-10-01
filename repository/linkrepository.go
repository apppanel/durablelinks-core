package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type LinkRepository interface {
	GetQueryParamsByHostAndPath(ctx context.Context, host, path string, projectID *uuid.UUID) (string, error)
	FindExistingShortLink(ctx context.Context, host, rawQS string, projectID *uuid.UUID) (string, error)
	CreateShortLink(ctx context.Context, host, path, rawQS string, unguessable bool, projectID *uuid.UUID) error
}

type linkRepository struct {
	db *sql.DB
}

func NewLinkRepository(db *sql.DB) LinkRepository {
	return &linkRepository{
		db: db,
	}
}

func (r *linkRepository) GetQueryParamsByHostAndPath(ctx context.Context, host, path string, projectID *uuid.UUID) (string, error) {
	var rawQueryStr string

	query := `SELECT query_params FROM apppanel_durable_links WHERE host = $1 AND path = $2`
	args := []interface{}{host, path}

	if projectID != nil {
		query += ` AND project_id = $3`
		args = append(args, *projectID)
	}

	row := r.db.QueryRowContext(ctx, query, args...)
	if err := row.Scan(&rawQueryStr); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Debug().
				Str("path", path).
				Msg("Link not found in database")
			return "", ErrLinkNotFound
		}
		log.Error().
			Err(err).
			Str("path", path).
			Msg("Failed to retrieve link from database")
		return "", fmt.Errorf("database error: %w", err)
	}

	return rawQueryStr, nil
}

func (r *linkRepository) FindExistingShortLink(ctx context.Context, host, rawQS string, projectID *uuid.UUID) (string, error) {
	var path string

	query := `SELECT path FROM apppanel_durable_links WHERE host = $1 AND query_params = $2 AND is_unguessable_path = FALSE`
	args := []interface{}{host, rawQS}

	if projectID != nil {
		query += ` AND project_id = $3`
		args = append(args, *projectID)
	}

	query += ` LIMIT 1`
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&path)
	return path, err
}

func (r *linkRepository) CreateShortLink(ctx context.Context, host, path, rawQS string, unguessable bool, projectID *uuid.UUID) error {
	var query string
	var args []interface{}

	if projectID != nil {
		query = `INSERT INTO apppanel_durable_links (host, path, query_params, is_unguessable_path, project_id) VALUES ($1, $2, $3, $4, $5)`
		args = []interface{}{host, path, rawQS, unguessable, *projectID}
	} else {
		query = `INSERT INTO apppanel_durable_links (host, path, query_params, is_unguessable_path) VALUES ($1, $2, $3, $4)`
		args = []interface{}{host, path, rawQS, unguessable}
	}

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}
