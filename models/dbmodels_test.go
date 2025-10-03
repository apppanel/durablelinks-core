package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestParamsHashAutoComputed(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&DurableLinkDB{})
	require.NoError(t, err)

	// Create a link without manually setting ParamsHash
	link := &DurableLinkDB{
		Host:               "example.com",
		Path:               "test123",
		Link:               "https://example.com/target",
		IsUnguessablePath:  false,
		AndroidPackageName: stringPtr("com.example.app"),
		IOSAppStoreID:      int64Ptr(123456789),
	}

	// ParamsHash should be empty before saving
	assert.Empty(t, link.ParamsHash)

	// Save the link
	err = db.Create(link).Error
	require.NoError(t, err)

	// ParamsHash should be automatically computed
	assert.NotEmpty(t, link.ParamsHash)
	assert.Equal(t, 64, len(link.ParamsHash), "Hash should be 64 characters (SHA256 hex)")

	// Verify the hash matches what ComputeParamsHash would return
	expectedHash := link.ComputeParamsHash()
	assert.Equal(t, expectedHash, link.ParamsHash)
}

func TestParamsHashAutoComputedOnUpdate(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&DurableLinkDB{})
	require.NoError(t, err)

	// Create a link
	link := &DurableLinkDB{
		Host:              "example.com",
		Path:              "test456",
		Link:              "https://example.com/target",
		IsUnguessablePath: false,
	}

	err = db.Create(link).Error
	require.NoError(t, err)

	originalHash := link.ParamsHash

	// Update the link with new parameters
	link.AndroidPackageName = stringPtr("com.newapp.package")
	err = db.Save(link).Error
	require.NoError(t, err)

	// ParamsHash should be different now
	assert.NotEqual(t, originalHash, link.ParamsHash)

	// Verify the hash matches what ComputeParamsHash would return
	expectedHash := link.ComputeParamsHash()
	assert.Equal(t, expectedHash, link.ParamsHash)
}

func TestFromDurableLink_AutoComputesHash(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&DurableLinkDB{})
	require.NoError(t, err)

	// Create a DurableLink
	dl := DurableLink{
		Host: "example.com",
		Link: "https://example.com/target",
		AndroidParameters: AndroidParameters{
			AndroidPackageName: stringPtr("com.example.app"),
		},
		IosParameters: IOSParameters{
			IOSAppStoreId: int64Ptr(987654321),
		},
	}

	// Convert to DB model
	dbLink := FromDurableLink(dl, "example.com", "abc789", false, nil)

	// ParamsHash should be empty (not computed yet)
	assert.Empty(t, dbLink.ParamsHash)

	// Save it
	err = db.Create(dbLink).Error
	require.NoError(t, err)

	// ParamsHash should now be auto-computed
	assert.NotEmpty(t, dbLink.ParamsHash)

	// Verify it matches
	expectedHash := dbLink.ComputeParamsHash()
	assert.Equal(t, expectedHash, dbLink.ParamsHash)
}

func stringPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}
