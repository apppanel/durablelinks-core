package repository

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()
	_ = log
	os.Exit(m.Run())
}

func setupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, LinkRepository) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %s", err)
	}
	repo := NewLinkRepository(db)
	return db, mock, repo
}

func TestGetQueryParamsByHostAndPath_Success(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	host := "example.com"
	path := "test"
	expected := "apn=com.app&amv=1"

	mock.ExpectQuery(`SELECT query_params FROM apppanel_durable_links`).
		WithArgs(host, path).
		WillReturnRows(sqlmock.NewRows([]string{"query_params"}).AddRow(expected))

	result, err := repo.GetQueryParamsByHostAndPath(context.Background(), host, path, nil)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestGetQueryParamsByHostAndPath_NotFound(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectQuery(`SELECT query_params FROM apppanel_durable_links`).
		WithArgs("unknown.com", "notfound").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.GetQueryParamsByHostAndPath(context.Background(), "unknown.com", "notfound", nil)
	assert.True(t, errors.Is(err, ErrLinkNotFound))
}

func TestFindExistingShortLink_Found(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	host := "example.com"
	rawQS := "apn=com.app&amv=1"
	path := "abc123"

	mock.ExpectQuery(`SELECT path FROM apppanel_durable_links`).
		WithArgs(host, rawQS).
		WillReturnRows(sqlmock.NewRows([]string{"path"}).AddRow(path))

	result, err := repo.FindExistingShortLink(context.Background(), host, rawQS, nil)
	assert.NoError(t, err)
	assert.Equal(t, path, result)
}

func TestCreateShortLink(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectExec(`INSERT INTO apppanel_durable_links`).
		WithArgs("example.com", "abc123", "apn=com.app&amv=1", true).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateShortLink(context.Background(), "example.com", "abc123", "apn=com.app&amv=1", true, nil)
	assert.NoError(t, err)
}

func TestFindExistingShortLink_NotFound(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectQuery(`SELECT path FROM apppanel_durable_links`).
		WithArgs("example.com", "apn=com.app&amv=1").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.FindExistingShortLink(context.Background(), "example.com", "apn=com.app&amv=1", nil)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, sql.ErrNoRows))
}

func TestCreateShortLink_DBError(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectExec(`INSERT INTO apppanel_durable_links`).
		WithArgs("example.com", "abc123", "apn=com.app&amv=1", true).
		WillReturnError(errors.New("insert failed"))

	err := repo.CreateShortLink(context.Background(), "example.com", "abc123", "apn=com.app&amv=1", true, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insert failed")
}

func TestGetQueryParamsByHostAndPath_DBError(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectQuery(`SELECT query_params FROM apppanel_durable_links`).
		WithArgs("example.com", "test").
		WillReturnError(errors.New("connection lost"))

	_, err := repo.GetQueryParamsByHostAndPath(context.Background(), "example.com", "test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection lost")
}

func TestGetQueryParamsByHostAndPath_WithProjectID(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	host := "example.com"
	path := "test"
	projectID := uuid.New()
	expected := "apn=com.app&amv=1"

	mock.ExpectQuery(`SELECT query_params FROM apppanel_durable_links`).
		WithArgs(host, path, projectID).
		WillReturnRows(sqlmock.NewRows([]string{"query_params"}).AddRow(expected))

	result, err := repo.GetQueryParamsByHostAndPath(context.Background(), host, path, &projectID)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestFindExistingShortLink_WithProjectID(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	host := "example.com"
	rawQS := "apn=com.app&amv=1"
	projectID := uuid.New()
	path := "abc123"

	mock.ExpectQuery(`SELECT path FROM apppanel_durable_links`).
		WithArgs(host, rawQS, projectID).
		WillReturnRows(sqlmock.NewRows([]string{"path"}).AddRow(path))

	result, err := repo.FindExistingShortLink(context.Background(), host, rawQS, &projectID)
	assert.NoError(t, err)
	assert.Equal(t, path, result)
}

func TestCreateShortLink_WithProjectID(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	projectID := uuid.New()
	mock.ExpectExec(`INSERT INTO apppanel_durable_links`).
		WithArgs("example.com", "abc123", "apn=com.app&amv=1", true, projectID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateShortLink(context.Background(), "example.com", "abc123", "apn=com.app&amv=1", true, &projectID)
	assert.NoError(t, err)
}
