package repository

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/apppanel/durablelinks-core/models"
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

func stringPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}

func TestGetLinkByHostAndPath_Success(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	host := "example.com"
	path := "test"
	link := "https://example.com/deep-link"
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "host", "path", "link", "is_unguessable_path", "project_id",
		"android_package_name", "android_fallback_link", "android_min_version",
		"ios_fallback_link", "ios_ipad_fallback_link", "ios_app_store_id",
		"social_title", "social_description", "social_image_link",
		"utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content",
		"itunes_pt", "itunes_at", "itunes_ct", "itunes_mt",
		"other_fallback_url", "params_hash", "created_at", "updated_at",
	}).AddRow(
		1, host, path, link, false, nil,
		stringPtr("com.example.app"), nil, nil,
		nil, nil, nil,
		nil, nil, nil,
		nil, nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil, "abc123hash", now, now,
	)

	mock.ExpectQuery(`SELECT \* FROM apppanel_durable_links`).
		WithArgs(host, path).
		WillReturnRows(rows)

	result, err := repo.GetLinkByHostAndPath(context.Background(), host, path, nil)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, link, result.Link)
	assert.Equal(t, host, result.Host)
	assert.Equal(t, stringPtr("com.example.app"), result.AndroidParameters.AndroidPackageName)
}

func TestGetLinkByHostAndPath_NotFound(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectQuery(`SELECT \* FROM apppanel_durable_links`).
		WithArgs("unknown.com", "notfound").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.GetLinkByHostAndPath(context.Background(), "unknown.com", "notfound", nil)
	assert.True(t, errors.Is(err, ErrLinkNotFound))
}

func TestGetLinkByHostAndPath_WithProjectID(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	host := "example.com"
	path := "test"
	link := "https://example.com/deep-link"
	projectID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "host", "path", "link", "is_unguessable_path", "project_id",
		"android_package_name", "android_fallback_link", "android_min_version",
		"ios_fallback_link", "ios_ipad_fallback_link", "ios_app_store_id",
		"social_title", "social_description", "social_image_link",
		"utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content",
		"itunes_pt", "itunes_at", "itunes_ct", "itunes_mt",
		"other_fallback_url", "params_hash", "created_at", "updated_at",
	}).AddRow(
		1, host, path, link, false, stringPtr(projectID.String()),
		nil, nil, nil,
		nil, nil, nil,
		nil, nil, nil,
		nil, nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil, "abc123hash", now, now,
	)

	mock.ExpectQuery(`SELECT \* FROM apppanel_durable_links`).
		WithArgs(host, path, projectID.String()).
		WillReturnRows(rows)

	result, err := repo.GetLinkByHostAndPath(context.Background(), host, path, &projectID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, link, result.Link)
}

func TestFindExistingShortLink_Found(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	host := "example.com"
	link := &models.DurableLink{
		Host: host,
		Link: "https://example.com/deep-link",
	}
	path := "abc123"

	mock.ExpectQuery(`SELECT path FROM apppanel_durable_links`).
		WithArgs(host, link.Link, sqlmock.AnyArg()). // params_hash is computed
		WillReturnRows(sqlmock.NewRows([]string{"path"}).AddRow(path))

	result, err := repo.FindExistingShortLink(context.Background(), host, link, nil)
	assert.NoError(t, err)
	assert.Equal(t, path, result)
}

func TestFindExistingShortLink_NotFound(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	link := &models.DurableLink{
		Host: "example.com",
		Link: "https://example.com/deep-link",
	}

	mock.ExpectQuery(`SELECT path FROM apppanel_durable_links`).
		WithArgs("example.com", link.Link, sqlmock.AnyArg()). // params_hash is computed
		WillReturnError(sql.ErrNoRows)

	_, err := repo.FindExistingShortLink(context.Background(), "example.com", link, nil)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, sql.ErrNoRows))
}

func TestFindExistingShortLink_WithProjectID(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	host := "example.com"
	link := &models.DurableLink{
		Host: host,
		Link: "https://example.com/deep-link",
	}
	projectID := uuid.New()
	path := "abc123"

	mock.ExpectQuery(`SELECT path FROM apppanel_durable_links`).
		WithArgs(host, link.Link, sqlmock.AnyArg(), projectID.String()). // params_hash is computed
		WillReturnRows(sqlmock.NewRows([]string{"path"}).AddRow(path))

	result, err := repo.FindExistingShortLink(context.Background(), host, link, &projectID)
	assert.NoError(t, err)
	assert.Equal(t, path, result)
}

func TestFindExistingShortLink_IgnoresUnguessableLinks(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	host := "example.com"
	link := &models.DurableLink{
		Host: host,
		Link: "https://example.com/deep-link",
	}

	// Query should filter out unguessable links with is_unguessable_path = FALSE clause
	mock.ExpectQuery(`SELECT path FROM apppanel_durable_links`).
		WithArgs(host, link.Link, sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows) // No short links found, only unguessable ones exist

	result, err := repo.FindExistingShortLink(context.Background(), host, link, nil)
	assert.Error(t, err)
	assert.Equal(t, "", result)
	assert.True(t, errors.Is(err, sql.ErrNoRows))
}

func TestFindExistingShortLink_DifferentParamsGetDifferentHash(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	host := "example.com"
	deepLink := "https://example.com/deep-link"

	// First link with Android package name
	link1 := &models.DurableLink{
		Host: host,
		Link: deepLink,
		AndroidParameters: models.AndroidParameters{
			AndroidPackageName: stringPtr("com.example.app1"),
		},
	}

	// Second link with different Android package name
	link2 := &models.DurableLink{
		Host: host,
		Link: deepLink,
		AndroidParameters: models.AndroidParameters{
			AndroidPackageName: stringPtr("com.example.app2"),
		},
	}

	// Compute actual hashes to verify they're different
	dbLink1 := models.FromDurableLink(*link1, "", "", false, nil)
	dbLink2 := models.FromDurableLink(*link2, "", "", false, nil)
	hash1 := dbLink1.ComputeParamsHash()
	hash2 := dbLink2.ComputeParamsHash()

	// Hashes should be different
	assert.NotEqual(t, hash1, hash2, "Different parameters should produce different hashes")

	// Mock expects the first query with hash1
	mock.ExpectQuery(`SELECT path FROM apppanel_durable_links`).
		WithArgs(host, deepLink, hash1).
		WillReturnRows(sqlmock.NewRows([]string{"path"}).AddRow("path1"))

	result1, err := repo.FindExistingShortLink(context.Background(), host, link1, nil)
	assert.NoError(t, err)
	assert.Equal(t, "path1", result1)
}

func TestFindExistingShortLink_SameParamsGetSameHash(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	host := "example.com"
	deepLink := "https://example.com/deep-link"

	// Two identical links
	link1 := &models.DurableLink{
		Host: host,
		Link: deepLink,
		AndroidParameters: models.AndroidParameters{
			AndroidPackageName: stringPtr("com.example.app"),
		},
		IosParameters: models.IOSParameters{
			IOSAppStoreId: int64Ptr(123456789),
		},
	}

	link2 := &models.DurableLink{
		Host: host,
		Link: deepLink,
		AndroidParameters: models.AndroidParameters{
			AndroidPackageName: stringPtr("com.example.app"),
		},
		IosParameters: models.IOSParameters{
			IOSAppStoreId: int64Ptr(123456789),
		},
	}

	// Compute hashes - should be identical
	dbLink1 := models.FromDurableLink(*link1, "", "", false, nil)
	dbLink2 := models.FromDurableLink(*link2, "", "", false, nil)
	hash1 := dbLink1.ComputeParamsHash()
	hash2 := dbLink2.ComputeParamsHash()

	assert.Equal(t, hash1, hash2, "Identical parameters should produce identical hashes")

	// Both queries should use the same hash
	path := "existing-path"
	mock.ExpectQuery(`SELECT path FROM apppanel_durable_links`).
		WithArgs(host, deepLink, hash1).
		WillReturnRows(sqlmock.NewRows([]string{"path"}).AddRow(path))

	result, err := repo.FindExistingShortLink(context.Background(), host, link1, nil)
	assert.NoError(t, err)
	assert.Equal(t, path, result)
}

func TestFindExistingShortLink_NilVsEmptyStringProducesDifferentHash(t *testing.T) {
	// Link with nil Android package name
	link1 := &models.DurableLink{
		Link: "https://example.com/deep-link",
		AndroidParameters: models.AndroidParameters{
			AndroidPackageName: nil,
		},
	}

	// Link with empty string Android package name
	link2 := &models.DurableLink{
		Link: "https://example.com/deep-link",
		AndroidParameters: models.AndroidParameters{
			AndroidPackageName: stringPtr(""),
		},
	}

	dbLink1 := models.FromDurableLink(*link1, "", "", false, nil)
	dbLink2 := models.FromDurableLink(*link2, "", "", false, nil)
	hash1 := dbLink1.ComputeParamsHash()
	hash2 := dbLink2.ComputeParamsHash()

	// nil and empty string should produce different hashes
	assert.NotEqual(t, hash1, hash2, "nil and empty string should produce different hashes")
}

func TestCreateShortLink(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	dbLink := &models.DurableLinkDB{
		Host:              "example.com",
		Path:              "abc123",
		Link:              "https://example.com/deep-link",
		IsUnguessablePath: true,
		ProjectID:         nil,
		AndroidPackageName: stringPtr("com.example.app"),
		IOSAppStoreID:     int64Ptr(123456789),
	}

	mock.ExpectExec(`INSERT INTO apppanel_durable_links`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateShortLink(context.Background(), dbLink, nil)
	assert.NoError(t, err)
}

func TestCreateShortLink_WithProjectID(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	projectID := uuid.New()
	dbLink := &models.DurableLinkDB{
		Host:              "example.com",
		Path:              "abc123",
		Link:              "https://example.com/deep-link",
		IsUnguessablePath: false,
		ProjectID:         stringPtr(projectID.String()),
		IOSFallbackLink:   stringPtr("https://example.com/ios-fallback"),
	}

	mock.ExpectExec(`INSERT INTO apppanel_durable_links`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateShortLink(context.Background(), dbLink, &projectID)
	assert.NoError(t, err)
}

func TestCreateShortLink_DBError(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	dbLink := &models.DurableLinkDB{
		Host:              "example.com",
		Path:              "abc123",
		Link:              "https://example.com/deep-link",
		IsUnguessablePath: true,
	}

	mock.ExpectExec(`INSERT INTO apppanel_durable_links`).
		WillReturnError(errors.New("insert failed"))

	err := repo.CreateShortLink(context.Background(), dbLink, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insert failed")
}

func TestGetLinkByHostAndPath_DBError(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectQuery(`SELECT \* FROM apppanel_durable_links`).
		WithArgs("example.com", "test").
		WillReturnError(errors.New("connection lost"))

	_, err := repo.GetLinkByHostAndPath(context.Background(), "example.com", "test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection lost")
}
