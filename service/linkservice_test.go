package service


import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/apppanel/durablelinks-core/models"
	"github.com/apppanel/durablelinks-core/repository"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func int64Ptr(i int64) *int64 {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

var defaultTenantCfg = TenantConfig{
	URLScheme:             "https",
	DomainAllowList:       []string{"example.com"},
	ShortPathLength:       8,
	UnguessablePathLength: 17,
}

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()
	_ = log
	os.Exit(m.Run())
}

func setupTestService(t *testing.T) (*linkService, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.DurableLinkDB{})
	require.NoError(t, err)

	repo := repository.NewLinkRepository(db)
	service := NewLinkService(repo)

	return service, db
}

func TestCreateDurableLink_Warnings(t *testing.T) {
	tests := []struct {
		name             string
		params           models.CreateDurableLinkRequest
		expectedWarnings []models.Warning
	}{
		{
			name: "invalid android fallback link should warn",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
					AndroidParameters: models.AndroidParameters{
						AndroidFallbackLink: stringPtr("not-a-valid-url"),
					},
				},
				Suffix: models.Suffix{
					Option: "UNGUESSABLE",
				},
			},
			expectedWarnings: []models.Warning{
				{
					WarningCode:    "MALFORMED_PARAM",
					WarningMessage: "Param 'androidFallbackLink' is not a valid URL",
				},
			},
		},
		{
			name: "invalid ios fallback link should warn",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
					IosParameters: models.IOSParameters{
						IOSFallbackLink: stringPtr("invalid-url"),
					},
				},
				Suffix: models.Suffix{
					Option: "UNGUESSABLE",
				},
			},
			expectedWarnings: []models.Warning{
				{
					WarningCode:    "MALFORMED_PARAM",
					WarningMessage: "Param 'iosFallbackLink' is not a valid URL",
				},
			},
		},
		{
			name: "invalid ios ipad fallback link should warn",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
					IosParameters: models.IOSParameters{
						IOSIpadFallbackLink: stringPtr("bad-url"),
					},
				},
				Suffix: models.Suffix{
					Option: "UNGUESSABLE",
				},
			},
			expectedWarnings: []models.Warning{
				{
					WarningCode:    "MALFORMED_PARAM",
					WarningMessage: "Param 'iosIpadFallbackLink' is not a valid URL",
				},
			},
		},
		{
			name: "invalid other platform fallback url should warn",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
					OtherPlatformParameters: models.OtherPlatformParameters{
						FallbackURL: stringPtr("not-valid"),
					},
				},
				Suffix: models.Suffix{
					Option: "UNGUESSABLE",
				},
			},
			expectedWarnings: []models.Warning{
				{
					WarningCode:    "MALFORMED_PARAM",
					WarningMessage: "Param 'fallbackUrl' is not a valid URL",
				},
			},
		},
		{
			name: "multiple invalid fallback urls should warn for each",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
					AndroidParameters: models.AndroidParameters{
						AndroidFallbackLink: stringPtr("bad-android-url"),
					},
					IosParameters: models.IOSParameters{
						IOSFallbackLink: stringPtr("bad-ios-url"),
					},
					SocialMetaTagInfo: models.SocialMetaTagInfo{
						SocialImageLink: stringPtr("bad-image-url"),
					},
				},
				Suffix: models.Suffix{
					Option: "UNGUESSABLE",
				},
			},
			expectedWarnings: []models.Warning{
				{
					WarningCode:    "MALFORMED_PARAM",
					WarningMessage: "Param 'androidFallbackLink' is not a valid URL",
				},
				{
					WarningCode:    "MALFORMED_PARAM",
					WarningMessage: "Param 'iosFallbackLink' is not a valid URL",
				},
				{
					WarningCode:    "MALFORMED_PARAM",
					WarningMessage: "Param 'socialImageLink' is not a valid URL",
				},
			},
		},
		{
			name: "valid fallback urls should not warn",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
					AndroidParameters: models.AndroidParameters{
						AndroidFallbackLink: stringPtr("https://example.com/android"),
					},
					IosParameters: models.IOSParameters{
						IOSFallbackLink:     stringPtr("https://example.com/ios"),
						IOSIpadFallbackLink: stringPtr("https://example.com/ipad"),
					},
					OtherPlatformParameters: models.OtherPlatformParameters{
						FallbackURL: stringPtr("https://example.com/other"),
					},
					SocialMetaTagInfo: models.SocialMetaTagInfo{
						SocialImageLink: stringPtr("https://example.com/image.png"),
					},
				},
				Suffix: models.Suffix{
					Option: "UNGUESSABLE",
				},
			},
			expectedWarnings: []models.Warning{},
		},
		{
			name: "mt param without pt should warn",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
					IosParameters: models.IOSParameters{
						IOSAppStoreId: int64Ptr(123456789),
					},
					AnalyticsInfo: models.AnalyticsInfo{
						ItunesConnectAnalytics: models.ITunesConnectAnalytics{
							Mt: stringPtr("8"),
						},
					},
				},
				Suffix: models.Suffix{
					Option: "UNGUESSABLE",
				},
			},
			expectedWarnings: []models.Warning{
				{
					WarningCode:    "UNRECOGNIZED_PARAM",
					WarningMessage: "Param 'mt' is not needed, since 'pt' is not specified.",
				},
			},
		},
		{
			name: "at param without pt should warn",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
					IosParameters: models.IOSParameters{
						IOSAppStoreId: int64Ptr(123456789),
					},
					AnalyticsInfo: models.AnalyticsInfo{
						ItunesConnectAnalytics: models.ITunesConnectAnalytics{
							At: stringPtr("affiliate_token"),
							// Pt is empty - should trigger warning
						},
					},
				},
				Suffix: models.Suffix{
					Option: "UNGUESSABLE",
				},
			},
			expectedWarnings: []models.Warning{
				{
					WarningCode:    "UNRECOGNIZED_PARAM",
					WarningMessage: "Param 'at' is not needed, since 'pt' is not specified.",
				},
			},
		},
		{
			name: "ct param without pt should warn",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
					IosParameters: models.IOSParameters{
						IOSAppStoreId: int64Ptr(123456789),
					},
					AnalyticsInfo: models.AnalyticsInfo{
						ItunesConnectAnalytics: models.ITunesConnectAnalytics{
							Ct: stringPtr("campaign_token"),
							// Pt is empty - should trigger warning
						},
					},
				},
				Suffix: models.Suffix{
					Option: "UNGUESSABLE",
				},
			},
			expectedWarnings: []models.Warning{
				{
					WarningCode:    "UNRECOGNIZED_PARAM",
					WarningMessage: "Param 'ct' is not needed, since 'pt' is not specified.",
				},
			},
		},
		{
			name: "pt param without isi should warn",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
					AnalyticsInfo: models.AnalyticsInfo{
						ItunesConnectAnalytics: models.ITunesConnectAnalytics{
							Pt: stringPtr("provider_token"),
						},
					},
				},
				Suffix: models.Suffix{
					Option: "UNGUESSABLE",
				},
			},
			expectedWarnings: []models.Warning{
				{
					WarningCode:    "UNRECOGNIZED_PARAM",
					WarningMessage: "Param 'pt' is not needed, since 'isi' is not specified.",
				},
			},
		},
		{
			name: "invalid social image URL should warn",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
					SocialMetaTagInfo: models.SocialMetaTagInfo{
						SocialImageLink: stringPtr("not-a-valid-url"),
					},
				},
				Suffix: models.Suffix{
					Option: "UNGUESSABLE",
				},
			},
			expectedWarnings: []models.Warning{
				{
					WarningCode:    "MALFORMED_PARAM",
					WarningMessage: "Param 'socialImageLink' is not a valid URL",
				},
			},
		},
		{
			name: "valid params should have no warnings",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
					IosParameters: models.IOSParameters{
						IOSAppStoreId: int64Ptr(123456789),
					},
					AnalyticsInfo: models.AnalyticsInfo{
						ItunesConnectAnalytics: models.ITunesConnectAnalytics{
							Pt: stringPtr("provider_token"),
							At: stringPtr("affiliate_token"),
							Ct: stringPtr("campaign_token"),
							Mt: stringPtr("8"),
						},
					},
					SocialMetaTagInfo: models.SocialMetaTagInfo{
						SocialImageLink: stringPtr("https://example.com/image.png"),
					},
				},
				Suffix: models.Suffix{
					Option: "UNGUESSABLE",
				},
			},
			expectedWarnings: []models.Warning{},
		},
		{
			name: "invalid suffix option should warn and default to UNGUESSABLE",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
				},
				Suffix: models.Suffix{
					Option: "INVALID",
				},
			},
			expectedWarnings: []models.Warning{
				{
					WarningCode:    "INVALID_SUFFIX_OPTION",
					WarningMessage: "Param 'suffix.option' must be 'SHORT' or 'UNGUESSABLE'. Received 'INVALID', defaulting to 'UNGUESSABLE'.",
				},
			},
		},
		{
			name: "lowercase short suffix should work without warning",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
				},
				Suffix: models.Suffix{
					Option: "short",
				},
			},
			expectedWarnings: []models.Warning{},
		},
		{
			name: "uppercase SHORT suffix should work without warning",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
				},
				Suffix: models.Suffix{
					Option: "SHORT",
				},
			},
			expectedWarnings: []models.Warning{},
		},
		{
			name: "empty suffix option should default to UNGUESSABLE with warning",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
				},
				Suffix: models.Suffix{
					Option: "",
				},
			},
			expectedWarnings: []models.Warning{
				{
					WarningCode:    "INVALID_SUFFIX_OPTION",
					WarningMessage: "Param 'suffix.option' must be 'SHORT' or 'UNGUESSABLE'. Received '', defaulting to 'UNGUESSABLE'.",
				},
			},
		},
		{
			name: "UNGUESSABLE suffix should work without warning",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
				},
				Suffix: models.Suffix{
					Option: "UNGUESSABLE",
				},
			},
			expectedWarnings: []models.Warning{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, _ := setupTestService(t)

			result, err := service.CreateDurableLink(context.Background(), tt.params, nil, defaultTenantCfg)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, len(tt.expectedWarnings), len(result.Warnings),
				"Expected %d warnings but got %d", len(tt.expectedWarnings), len(result.Warnings))

			for i, expectedWarning := range tt.expectedWarnings {
				if i < len(result.Warnings) {
					assert.Equal(t, expectedWarning.WarningCode, result.Warnings[i].WarningCode)
					assert.Equal(t, expectedWarning.WarningMessage, result.Warnings[i].WarningMessage)
				}
			}

		})
	}
}

func TestCreateDurableLink_Defaults(t *testing.T) {
	defaultAppStoreID := int64Ptr(123456789)
	defaultAndroidPkg := stringPtr("com.example.app")

	tenantCfgWithDefaults := TenantConfig{
		URLScheme:             "https",
		DomainAllowList:       []string{"example.com"},
		ShortPathLength:       8,
		UnguessablePathLength: 17,
		DefaultIOSAppStoreId:  defaultAppStoreID,
		DefaultAndroidPackage: defaultAndroidPkg,
	}

	tests := []struct {
		name             string
		params           models.CreateDurableLinkRequest
		tenantCfg        TenantConfig
		expectedWarnings []models.Warning
	}{
		{
			name: "both defaults applied when not provided",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
				},
				Suffix: models.Suffix{
					Option: "UNGUESSABLE",
				},
			},
			tenantCfg: tenantCfgWithDefaults,
			expectedWarnings: []models.Warning{
				{
					WarningCode:    "DEFAULT_APPLIED",
					WarningMessage: "Using default iOS App Store ID: 123456789",
				},
				{
					WarningCode:    "DEFAULT_APPLIED",
					WarningMessage: "Using default Android package name: com.example.app",
				},
			},
		},
		{
			name: "only Android default applied when iOS already provided",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
					IosParameters: models.IOSParameters{
						IOSAppStoreId: int64Ptr(999999999),
					},
				},
				Suffix: models.Suffix{
					Option: "UNGUESSABLE",
				},
			},
			tenantCfg: tenantCfgWithDefaults,
			expectedWarnings: []models.Warning{
				{
					WarningCode:    "DEFAULT_APPLIED",
					WarningMessage: "Using default Android package name: com.example.app",
				},
			},
		},
		{
			name: "only iOS default applied when Android already provided",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
					AndroidParameters: models.AndroidParameters{
						AndroidPackageName: stringPtr("com.custom.app"),
					},
				},
				Suffix: models.Suffix{
					Option: "UNGUESSABLE",
				},
			},
			tenantCfg: tenantCfgWithDefaults,
			expectedWarnings: []models.Warning{
				{
					WarningCode:    "DEFAULT_APPLIED",
					WarningMessage: "Using default iOS App Store ID: 123456789",
				},
			},
		},
		{
			name: "no defaults applied when both already provided",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
					IosParameters: models.IOSParameters{
						IOSAppStoreId: int64Ptr(999999999),
					},
					AndroidParameters: models.AndroidParameters{
						AndroidPackageName: stringPtr("com.custom.app"),
					},
				},
				Suffix: models.Suffix{
					Option: "UNGUESSABLE",
				},
			},
			tenantCfg:        tenantCfgWithDefaults,
			expectedWarnings: []models.Warning{},
		},
		{
			name: "no defaults applied when tenant config has no defaults",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://example.com/target",
				},
				Suffix: models.Suffix{
					Option: "UNGUESSABLE",
				},
			},
			tenantCfg:        defaultTenantCfg,
			expectedWarnings: []models.Warning{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, _ := setupTestService(t)

			result, err := service.CreateDurableLink(context.Background(), tt.params, nil, tt.tenantCfg)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Check warnings
			assert.Equal(t, len(tt.expectedWarnings), len(result.Warnings),
				"Expected %d warnings but got %d", len(tt.expectedWarnings), len(result.Warnings))

			for i, expectedWarning := range tt.expectedWarnings {
				if i < len(result.Warnings) {
					assert.Equal(t, expectedWarning.WarningCode, result.Warnings[i].WarningCode)
					assert.Equal(t, expectedWarning.WarningMessage, result.Warnings[i].WarningMessage)
				}
			}

		})
	}
}

func TestCreateDurableLink_ReuseExistingShortLink(t *testing.T) {
	service, db := setupTestService(t)

	// First create an existing short link
	existingLink := &models.DurableLinkDB{
		Host:              "example.com",
		Path:              "abc123",
		Link:              "https://example.com/target",
		IsUnguessablePath: false,
		ParamsHash:        models.FromDurableLink(models.DurableLink{Link: "https://example.com/target"}, "", "", false, nil).ComputeParamsHash(),
	}
	db.Create(existingLink)

	params := models.CreateDurableLinkRequest{
		DurableLinkInfo: models.DurableLink{
			Host: "example.com",
			Link: "https://example.com/target",
		},
		Suffix: models.Suffix{
			Option: "SHORT",
		},
	}

	result, err := service.CreateDurableLink(context.Background(), params, nil, defaultTenantCfg)
	require.NoError(t, err)
	require.NotNil(t, result)

	expectedShortLink := "https://example.com/abc123"
	assert.Equal(t, expectedShortLink, result.ShortLink)
	assert.Equal(t, 0, len(result.Warnings))
}

func TestResolveShortPath(t *testing.T) {
	tests := []struct {
		name        string
		rawURL      string
		mockPath    string
		mockLink    string
		mockErr     error
		expectError error
		expectLink  string
	}{
		{
			name:       "valid short URL resolves successfully",
			rawURL:     "https://example.com/abc123",
			mockPath:   "abc123",
			mockLink:   "https://example.com/target",
			expectLink: "https://example.com/target",
		},
		{
			name:       "URL with preview prefix is normalized",
			rawURL:     "https://preview.example.com/abc123",
			mockPath:   "abc123",
			mockLink:   "https://example.com/target",
			expectLink: "https://example.com/target",
		},
		{
			name:       "URL with hyphenated preview is normalized",
			rawURL:     "https://example-preview.com/abc123",
			mockPath:   "abc123",
			mockLink:   "https://example.com/target",
			expectLink: "https://example.com/target",
		},
		{
			name:        "invalid URL format returns error",
			rawURL:      "not a valid url://",
			expectError: ErrInvalidRequestedLink,
		},
		{
			name:        "empty path returns error",
			rawURL:      "https://example.com/",
			expectError: ErrInvalidPathFormat,
		},
		{
			name:        "multiple path segments returns error",
			rawURL:      "https://example.com/abc123/extra",
			expectError: ErrInvalidPathFormat,
		},
		{
			name:        "link not found in database",
			rawURL:      "https://example.com/notfound",
			mockPath:    "", // Don't create link in DB
			mockErr:     nil,
			expectError: repository.ErrLinkNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, db := setupTestService(t)

			if tt.mockPath != "" && tt.mockErr == nil {
				// Create the link in the database
				link := &models.DurableLinkDB{
					Host:              "example.com",
					Path:              tt.mockPath,
					Link:              tt.mockLink,
					IsUnguessablePath: false,
					ParamsHash:        "hash123",
				}
				db.Create(link)
			}

			result, err := service.ResolveShortPath(context.Background(), tt.rawURL, nil, defaultTenantCfg)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectError)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expectLink, result.LongLink)
			}

		})
	}
}

func TestCreateDurableLink_ErrorPaths(t *testing.T) {
	tests := []struct {
		name        string
		params      models.CreateDurableLinkRequest
		tenantCfg   TenantConfig
		expectError error
	}{
		{
			name: "invalid host returns error",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "not a valid host://",
					Link: "https://example.com/target",
				},
				Suffix: models.Suffix{
					Option: "UNGUESSABLE",
				},
			},
			tenantCfg:   defaultTenantCfg,
			expectError: fmt.Errorf("invalid host"),
		},
		{
			name: "domain not in allow list returns error",
			params: models.CreateDurableLinkRequest{
				DurableLinkInfo: models.DurableLink{
					Host: "example.com",
					Link: "https://notallowed.com/target",
				},
				Suffix: models.Suffix{
					Option: "UNGUESSABLE",
				},
			},
			tenantCfg:   defaultTenantCfg,
			expectError: ErrDomainLinkNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, _ := setupTestService(t)

			result, err := service.CreateDurableLink(context.Background(), tt.params, nil, tt.tenantCfg)

			assert.Error(t, err)
			if tt.expectError != nil {
				assert.ErrorContains(t, err, tt.expectError.Error())
			}
			assert.Nil(t, result)

		})
	}
}

func TestGenerateDurableLinkPath(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{
			name:   "short path",
			length: 6,
		},
		{
			name:   "medium path",
			length: 12,
		},
		{
			name:   "long path",
			length: 24,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := make(map[string]bool)
			const iterations = 1000

			for range iterations {
				path := generateDurableLinkPath(tt.length)

				// Test length
				if len(path) != tt.length {
					t.Errorf("generateDurableLinkPath(%d) length = %d, want %d", tt.length, len(path), tt.length)
				}

				// Test character set - must be alphanumeric
				for _, r := range path {
					if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
						t.Errorf("generateDurableLinkPath(%d) contains invalid character: %c", tt.length, r)
					}
				}

				// Test uniqueness
				if paths[path] {
					t.Errorf("generateDurableLinkPath(%d) generated duplicate path: %s", tt.length, path)
				}
				paths[path] = true
			}

			// Test distribution - should generate both letters and numbers
			hasLetters := false
			hasNumbers := false
			for path := range paths {
				for _, r := range path {
					if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
						hasLetters = true
					}
					if r >= '0' && r <= '9' {
						hasNumbers = true
					}
				}
			}

			if !hasLetters || !hasNumbers {
				t.Errorf("generateDurableLinkPath(%d) does not generate both letters and numbers", tt.length)
			}
		})
	}
}

func TestRemovePreviewFromHost(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "prefix preview",
			input:    "preview.acme.short.link",
			expected: "acme.short.link",
		},
		{
			name:     "hyphenated preview",
			input:    "acme-preview.short.link",
			expected: "acme.short.link",
		},
		{
			name:     "no preview",
			input:    "acme.short.link",
			expected: "acme.short.link",
		},
		{
			name:     "prefix preview with nested domain",
			input:    "preview.staging.acme.short.link",
			expected: "staging.acme.short.link",
		},
		{
			name:     "hyphenated preview with dot in domain",
			input:    "myapp-preview.dev.short.link",
			expected: "myapp.dev.short.link",
		},
		{
			name:     "unrelated prefix",
			input:    "notpreview.acme.short.link",
			expected: "notpreview.acme.short.link",
		},
		{
			name:     "unrelated suffix",
			input:    "acme-somethingelse.short.link",
			expected: "acme-somethingelse.short.link",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := removePreviewFromHost(tc.input)
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}
