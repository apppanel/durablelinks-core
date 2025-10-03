package repository

import (
	"context"
	"os"
	"testing"

	"github.com/apppanel/durablelinks-core/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()
	_ = log
	os.Exit(m.Run())
}

func setupTestDB(t *testing.T) (*gorm.DB, LinkRepository) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.DurableLinkDB{})
	require.NoError(t, err)

	repo := NewLinkRepository(db)
	return db, repo
}

func stringPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}

func TestGetLinkByHostAndPath_Success(t *testing.T) {
	db, repo := setupTestDB(t)

	host := "example.com"
	path := "test"
	link := "https://example.com/deep-link"

	// Create a link in the database
	dbLink := &models.DurableLinkDB{
		Host:               host,
		Path:               path,
		Link:               link,
		IsUnguessablePath:  false,
		AndroidPackageName: stringPtr("com.example.app"),
		ParamsHash:         "abc123hash",
	}
	db.Create(dbLink)

	result, err := repo.GetLinkByHostAndPath(context.Background(), host, path, nil)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, link, result.Link)
	assert.Equal(t, host, result.Host)
	assert.Equal(t, stringPtr("com.example.app"), result.AndroidParameters.AndroidPackageName)
}

func TestGetLinkByHostAndPath_NotFound(t *testing.T) {
	_, repo := setupTestDB(t)

	_, err := repo.GetLinkByHostAndPath(context.Background(), "unknown.com", "notfound", nil)
	assert.ErrorIs(t, err, ErrLinkNotFound)
}

func TestGetLinkByHostAndPath_WithProjectID(t *testing.T) {
	db, repo := setupTestDB(t)

	host := "example.com"
	path := "test"
	link := "https://example.com/deep-link"
	projectID := uuid.New()
	projectIDStr := projectID.String()

	// Create a link in the database
	dbLink := &models.DurableLinkDB{
		Host:              host,
		Path:              path,
		Link:              link,
		IsUnguessablePath: false,
		ProjectID:         &projectIDStr,
		ParamsHash:        "abc123hash",
	}
	db.Create(dbLink)

	result, err := repo.GetLinkByHostAndPath(context.Background(), host, path, &projectID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, link, result.Link)
}

func TestFindExistingShortLink_Found(t *testing.T) {
	db, repo := setupTestDB(t)

	host := "example.com"
	linkURL := "https://example.com/target"
	existingPath := "abc123"

	link := &models.DurableLink{
		Host: host,
		Link: linkURL,
	}

	// Create existing short link
	dbLink := models.FromDurableLink(*link, host, existingPath, false, nil)
	db.Create(dbLink)

	path, err := repo.FindExistingShortLink(context.Background(), host, link, nil)
	assert.NoError(t, err)
	assert.Equal(t, existingPath, path)
}

func TestFindExistingShortLink_NotFound(t *testing.T) {
	_, repo := setupTestDB(t)

	link := &models.DurableLink{
		Host: "example.com",
		Link: "https://example.com/target",
	}

	_, err := repo.FindExistingShortLink(context.Background(), "example.com", link, nil)
	assert.Error(t, err)
}

func TestCreateShortLink_Success(t *testing.T) {
	db, repo := setupTestDB(t)

	host := "example.com"
	path := "abc123"
	linkURL := "https://example.com/target"

	link := &models.DurableLinkDB{
		Host:              host,
		Path:              path,
		Link:              linkURL,
		IsUnguessablePath: false,
		ParamsHash:        "hash123",
	}

	err := repo.CreateShortLink(context.Background(), link, nil)
	assert.NoError(t, err)

	// Verify it was created
	var result models.DurableLinkDB
	db.Where("host = ? AND path = ?", host, path).First(&result)
	assert.Equal(t, linkURL, result.Link)
}

func TestCreateShortLink_WithProjectID(t *testing.T) {
	db, repo := setupTestDB(t)

	host := "example.com"
	path := "abc123"
	linkURL := "https://example.com/target"
	projectID := uuid.New()

	link := &models.DurableLinkDB{
		Host:              host,
		Path:              path,
		Link:              linkURL,
		IsUnguessablePath: false,
		ParamsHash:        "hash123",
	}

	err := repo.CreateShortLink(context.Background(), link, &projectID)
	assert.NoError(t, err)

	// Verify it was created with projectID
	var result models.DurableLinkDB
	db.Where("host = ? AND path = ?", host, path).First(&result)
	assert.Equal(t, linkURL, result.Link)
	assert.NotNil(t, result.ProjectID)
	assert.Equal(t, projectID.String(), *result.ProjectID)
}
